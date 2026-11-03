import test, { beforeEach, afterEach } from "node:test";
import assert from "node:assert";

// 1. In-memory localStorage polyfill for Node.js test environment
const mockStorage = new Map();
globalThis.localStorage = {
  getItem: (key) => (mockStorage.has(key) ? mockStorage.get(key) : null),
  setItem: (key, val) => mockStorage.set(key, String(val)),
  removeItem: (key) => mockStorage.delete(key),
  clear: () => mockStorage.clear(),
};

// 2. DOM window & document polyfill for event listener tests
const windowListeners = new Map();
const documentListeners = new Map();

globalThis.window = {
  location: { href: "http://localhost:3000" },
  addEventListener: (type, fn) => {
    if (!windowListeners.has(type)) windowListeners.set(type, new Set());
    windowListeners.get(type).add(fn);
  },
  removeEventListener: (type, fn) => {
    windowListeners.get(type)?.delete(fn);
  },
  dispatchEvent: (event) => {
    const fns = windowListeners.get(event.type);
    if (fns) fns.forEach((fn) => fn(event));
  },
};

globalThis.document = {
  visibilityState: "visible",
  addEventListener: (type, fn) => {
    if (!documentListeners.has(type)) documentListeners.set(type, new Set());
    documentListeners.get(type).add(fn);
  },
  removeEventListener: (type, fn) => {
    documentListeners.get(type)?.delete(fn);
  },
  dispatchEvent: (event) => {
    const fns = documentListeners.get(event.type);
    if (fns) fns.forEach((fn) => fn(event));
  },
};

// 3. Import authStore, client, and useSessionSync modules
const {
  default: useAuthStore,
  STORAGE_KEYS,
  resetActiveRevalidatePromiseForTesting,
} = await import("../../store/authStore.js");

const {
  rawAuthAxios,
  default: apiClient,
  resetActiveRefreshPromiseForTesting,
} = await import("../../api/client.js");

const {
  handleStorageEvent,
  handleVisibilityOrFocus,
  setupSessionSync,
  resetLastRevalidateTimeForTesting,
  REVALIDATION_COOLDOWN_MS,
} = await import("../useSessionSync.js");

const createAxiosError = (status, data, config, message = `Request failed with status ${status}`) => {
  const err = new Error(message);
  err.response = {
    status,
    statusText: status === 401 ? "Unauthorized" : status === 403 ? "Forbidden" : "Error",
    data: data || {},
    headers: {},
    config,
  };
  err.config = config;
  return err;
};

beforeEach(() => {
  mockStorage.clear();
  windowListeners.clear();
  documentListeners.clear();
  document.visibilityState = "visible";
  resetActiveRefreshPromiseForTesting();
  resetActiveRevalidatePromiseForTesting();
  resetLastRevalidateTimeForTesting();
  useAuthStore.setState({
    user: null,
    accessToken: null,
    refreshToken: null,
    isAuthenticated: false,
    isInitializing: false,
    sessionVersion: 0,
  });
});

afterEach(() => {
  resetActiveRefreshPromiseForTesting();
  resetActiveRevalidatePromiseForTesting();
  resetLastRevalidateTimeForTesting();
});

test("setupSessionSync registers and cleanly unbinds storage, visibilitychange, and focus listeners", () => {
  const cleanup = setupSessionSync();

  assert.strictEqual(windowListeners.get("storage")?.size, 1);
  assert.strictEqual(windowListeners.get("focus")?.size, 1);
  assert.strictEqual(documentListeners.get("visibilitychange")?.size, 1);

  // Invoke cleanup (as React Strict Mode or component unmount does)
  cleanup();

  assert.strictEqual(windowListeners.get("storage")?.size, 0);
  assert.strictEqual(windowListeners.get("focus")?.size, 0);
  assert.strictEqual(documentListeners.get("visibilitychange")?.size, 0);
});

test("handleStorageEvent ignores non-auth storage keys", () => {
  useAuthStore.getState().setAuth({ id: "user-1", role: "CUSTOMER" }, "tok-1", "ref-1");
  const versionBefore = useAuthStore.getState().sessionVersion;

  const result = handleStorageEvent({ key: "theme_preference" });

  assert.strictEqual(result.synced, false);
  assert.strictEqual(result.ignored, true);
  assert.strictEqual(useAuthStore.getState().sessionVersion, versionBefore);
});

test("handleStorageEvent updates state when another tab writes new tokens and role", () => {
  useAuthStore.getState().setAuth({ id: "user-2", role: "CUSTOMER" }, "old-tok", "old-ref");

  // Another tab updates localStorage
  localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify({ id: "user-2", role: "ORGANIZER" }));
  localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, "new-remote-tok");
  localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, "new-remote-ref");

  const result = handleStorageEvent({ key: STORAGE_KEYS.USER });

  assert.strictEqual(result.synced, true);
  assert.strictEqual(result.roleChanged, true);
  assert.strictEqual(result.newRole, "ORGANIZER");
  assert.strictEqual(useAuthStore.getState().user.role, "ORGANIZER");
  assert.strictEqual(useAuthStore.getState().accessToken, "new-remote-tok");
  assert.strictEqual(useAuthStore.getState().refreshToken, "new-remote-ref");
});

test("handleStorageEvent synchronizes remote logout (localStorage.clear or auth key removed)", () => {
  useAuthStore.getState().setAuth({ id: "user-3", role: "CUSTOMER" }, "tok-3", "ref-3");
  assert.strictEqual(useAuthStore.getState().isAuthenticated, true);

  // Another tab calls localStorage.clear() -> event.key is null
  localStorage.clear();
  const result = handleStorageEvent({ key: null });

  assert.strictEqual(result.synced, true);
  assert.strictEqual(result.isAuthenticated, false);
  assert.strictEqual(useAuthStore.getState().isAuthenticated, false);
  assert.strictEqual(useAuthStore.getState().user, null);
  assert.strictEqual(useAuthStore.getState().accessToken, null);
});

test("handleStorageEvent handles token rotation from another tab", () => {
  useAuthStore.getState().setAuth({ id: "user-4", role: "CUSTOMER" }, "acc-4", "old-ref-4");

  localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, "rotated-ref-4");
  const result = handleStorageEvent({ key: STORAGE_KEYS.REFRESH_TOKEN });

  assert.strictEqual(result.synced, true);
  assert.strictEqual(useAuthStore.getState().refreshToken, "rotated-ref-4");
});

test("handleVisibilityOrFocus skips revalidation when tab is hidden", async () => {
  useAuthStore.getState().setAuth({ id: "user-5", role: "CUSTOMER" }, "acc-5", "ref-5");
  document.visibilityState = "hidden";

  let apiCalled = false;
  apiClient.defaults.adapter = async (config) => {
    apiCalled = true;
    return { status: 200, data: {}, config };
  };

  const result = await handleVisibilityOrFocus();

  assert.strictEqual(result.revalidated, false);
  assert.strictEqual(result.reason, "document_hidden");
  assert.strictEqual(apiCalled, false);
});

test("handleVisibilityOrFocus skips revalidation when user is unauthenticated or initializing", async () => {
  useAuthStore.setState({ isAuthenticated: false, isInitializing: false });
  const result1 = await handleVisibilityOrFocus();
  assert.strictEqual(result1.revalidated, false);
  assert.strictEqual(result1.reason, "unauthenticated_or_initializing");

  useAuthStore.setState({ isAuthenticated: true, isInitializing: true });
  const result2 = await handleVisibilityOrFocus();
  assert.strictEqual(result2.revalidated, false);
  assert.strictEqual(result2.reason, "unauthenticated_or_initializing");
});

test("handleVisibilityOrFocus throttles duplicate calls within cooldown period", async () => {
  useAuthStore.getState().setAuth({ id: "user-6", role: "CUSTOMER" }, "acc-6", "ref-6");

  let meCalls = 0;
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/auth/me")) {
      meCalls++;
      return {
        status: 200,
        data: { user: { id: "user-6", role: "CUSTOMER" } },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  const fixedTime = 1000000;
  // First call (e.g. from visibilitychange)
  const res1 = await handleVisibilityOrFocus({ now: fixedTime });
  assert.strictEqual(res1.revalidated, true);
  assert.strictEqual(meCalls, 1);

  // Immediate second call (e.g. from focus event 50ms later)
  const res2 = await handleVisibilityOrFocus({ now: fixedTime + 50 });
  assert.strictEqual(res2.revalidated, false);
  assert.strictEqual(res2.reason, "cooldown_active");
  assert.strictEqual(meCalls, 1, "Duplicate request within cooldown must be suppressed");

  // Call after cooldown expires
  const res3 = await handleVisibilityOrFocus({ now: fixedTime + REVALIDATION_COOLDOWN_MS + 10 });
  assert.strictEqual(res3.revalidated, true);
  assert.strictEqual(meCalls, 2);
});

test("handleVisibilityOrFocus detects role promotion on tab return and refreshes token", async () => {
  useAuthStore.getState().setAuth({ id: "user-7", role: "CUSTOMER" }, "acc-customer", "ref-tok");

  let refreshCalled = 0;
  rawAuthAxios.defaults.adapter = async (config) => {
    refreshCalled++;
    return {
      status: 200,
      data: {
        access_token: "acc-organizer-promoted",
        refresh_token: "ref-rotated",
        user: { id: "user-7", role: "ORGANIZER" },
      },
      config,
    };
  };

  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/auth/me")) {
      return {
        status: 200,
        data: { user: { id: "user-7", role: "ORGANIZER" } },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  const res = await handleVisibilityOrFocus({ now: 2000000 });

  assert.strictEqual(res.revalidated, true);
  assert.strictEqual(res.result.roleChanged, true);
  assert.strictEqual(res.result.oldRole, "CUSTOMER");
  assert.strictEqual(res.result.newRole, "ORGANIZER");
  assert.strictEqual(refreshCalled, 1);

  assert.strictEqual(useAuthStore.getState().user.role, "ORGANIZER");
  assert.strictEqual(useAuthStore.getState().accessToken, "acc-organizer-promoted");
});

test("handleVisibilityOrFocus preserves session on transient 500 error", async () => {
  useAuthStore.getState().setAuth({ id: "user-8", role: "CUSTOMER" }, "acc-8", "ref-8");

  apiClient.defaults.adapter = async (config) => {
    throw createAxiosError(500, { message: "Internal server error" }, config);
  };

  const res = await handleVisibilityOrFocus({ now: 3000000 });

  assert.strictEqual(res.revalidated, true);
  assert.ok(res.result.error);
  assert.strictEqual(useAuthStore.getState().isAuthenticated, true);
  assert.strictEqual(useAuthStore.getState().user.role, "CUSTOMER");
});

test("simulated window and document event dispatching triggers sync and revalidation", async () => {
  useAuthStore.getState().setAuth({ id: "user-9", role: "CUSTOMER" }, "acc-9", "ref-9");

  let meCalls = 0;
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/auth/me")) {
      meCalls++;
      return {
        status: 200,
        data: { user: { id: "user-9", role: "CUSTOMER", fullName: "Updated via Focus" } },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  const cleanup = setupSessionSync();

  // 1. Dispatch focus event on window
  window.dispatchEvent({ type: "focus" });
  // Wait a microtask for async handler
  await new Promise((resolve) => setTimeout(resolve, 20));
  assert.strictEqual(meCalls, 1);
  assert.strictEqual(useAuthStore.getState().user.fullName, "Updated via Focus");

  // 2. Dispatch storage event on window
  localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify({ id: "user-9", role: "ORGANIZER" }));
  window.dispatchEvent({ type: "storage", key: STORAGE_KEYS.USER });
  assert.strictEqual(useAuthStore.getState().user.role, "ORGANIZER");

  cleanup();
});
