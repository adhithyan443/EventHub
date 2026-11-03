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

// 2. DOM window & document polyfill
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

// 3. Import application modules
const {
  default: useAuthStore,
  STORAGE_KEYS,
  decodeTokenPayload,
  resetActiveRevalidatePromiseForTesting,
} = await import("../store/authStore.js");

const {
  default: apiClient,
  rawAuthAxios,
  requestTokenRefresh,
  resetActiveRefreshPromiseForTesting,
} = await import("../api/client.js");

const {
  handleStorageEvent,
  handleVisibilityOrFocus,
  setupSessionSync,
  resetLastRevalidateTimeForTesting,
} = await import("../hooks/useSessionSync.js");

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

const createTestJwt = (payload) => {
  const header = Buffer.from(JSON.stringify({ alg: "HS256", typ: "JWT" })).toString("base64url");
  const body = Buffer.from(JSON.stringify(payload)).toString("base64url");
  return `${header}.${body}.mock-signature`;
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
});

// Scenario 1: Fresh login stores the correct user, access token, and refresh token
test("Scenario 1: Fresh login stores the correct user, access token, and refresh token", () => {
  const customerUser = {
    id: "user-cust-1",
    fullName: "Customer Alice",
    email: "alice@example.com",
    role: "CUSTOMER",
    status: "ACTIVE",
  };
  const accessToken = createTestJwt({ userId: customerUser.id, role: "CUSTOMER", exp: 2000000000 });
  const refreshToken = "rt-alice-valid";

  useAuthStore.getState().setAuth(customerUser, accessToken, refreshToken);

  const state = useAuthStore.getState();
  assert.strictEqual(state.isAuthenticated, true);
  assert.strictEqual(state.isInitializing, false);
  assert.deepStrictEqual(state.user, customerUser);
  assert.strictEqual(state.accessToken, accessToken);
  assert.strictEqual(state.refreshToken, refreshToken);

  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN), accessToken);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), refreshToken);
  assert.deepStrictEqual(JSON.parse(localStorage.getItem(STORAGE_KEYS.USER)), customerUser);
});

// Scenario 2: Application startup restores a valid session
test("Scenario 2: Application startup restores a valid session", async () => {
  const storedUser = {
    id: "user-cust-startup",
    fullName: "Startup User",
    email: "startup@example.com",
    role: "CUSTOMER",
    status: "ACTIVE",
  };
  const validAccessToken = createTestJwt({ userId: storedUser.id, role: "CUSTOMER", exp: 2000000000 });
  const storedRefreshToken = "rt-startup-valid";

  localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, validAccessToken);
  localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, storedRefreshToken);
  localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(storedUser));

  useAuthStore.setState({
    user: storedUser,
    accessToken: validAccessToken,
    refreshToken: storedRefreshToken,
    isAuthenticated: true,
    isInitializing: true,
  });

  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/auth/me")) {
      return {
        status: 200,
        data: { user: storedUser },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  await useAuthStore.getState().initializeAuth();

  const state = useAuthStore.getState();
  assert.strictEqual(state.isAuthenticated, true);
  assert.strictEqual(state.isInitializing, false);
  assert.strictEqual(state.user.id, storedUser.id);
  assert.strictEqual(state.user.role, "CUSTOMER");

  // 2b: Startup restoration when ONLY a refresh token exists in storage
  mockStorage.clear();
  localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, "rt-only-startup");
  useAuthStore.setState({
    user: null,
    accessToken: null,
    refreshToken: "rt-only-startup",
    isAuthenticated: false,
    isInitializing: true,
    sessionVersion: 0,
  });

  const restoredUser = {
    id: "user-restored-only",
    fullName: "Restored User",
    email: "restored@example.com",
    role: "CUSTOMER",
    status: "ACTIVE",
  };
  const restoredJwt = createTestJwt({ userId: restoredUser.id, role: "CUSTOMER", exp: 2000000000 });

  rawAuthAxios.defaults.adapter = async (config) => {
    return {
      status: 200,
      data: {
        access_token: restoredJwt,
        refresh_token: "rt-rotated-only",
        user: restoredUser,
      },
      headers: {},
      config,
    };
  };

  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/auth/me")) {
      return {
        status: 200,
        data: { user: restoredUser },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  await useAuthStore.getState().initializeAuth();

  const restoredState = useAuthStore.getState();
  assert.strictEqual(restoredState.isAuthenticated, true, "Must be authenticated after refresh restoration");
  assert.strictEqual(restoredState.isInitializing, false, "isInitializing must be false");
  assert.strictEqual(restoredState.user.id, "user-restored-only");
  assert.strictEqual(restoredState.accessToken, restoredJwt);
  assert.strictEqual(restoredState.refreshToken, "rt-rotated-only");
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), "rt-rotated-only");
});

// Scenario 3: An expired access token triggers one refresh operation and retries the original request
test("Scenario 3: An expired access token triggers one refresh operation and retries original request", async () => {
  const user = { id: "user-3", role: "CUSTOMER", email: "user3@example.com" };
  const initialAccessToken = "expired-token-3";
  const initialRefreshToken = "valid-refresh-3";
  const refreshedAccessToken = createTestJwt({ userId: user.id, role: "CUSTOMER", exp: 2000000000 });

  useAuthStore.getState().setAuth(user, initialAccessToken, initialRefreshToken);

  let refreshCallCount = 0;
  let protectedCallCount = 0;

  rawAuthAxios.defaults.adapter = async (config) => {
    refreshCallCount++;
    return {
      status: 200,
      data: {
        access_token: refreshedAccessToken,
        refresh_token: "rotated-refresh-3",
        user: { ...user, status: "ACTIVE" },
      },
      headers: {},
      config,
    };
  };

  apiClient.defaults.adapter = async (config) => {
    protectedCallCount++;
    const authHeader = config.headers?.Authorization || config.headers?.authorization;
    if (authHeader === `Bearer ${initialAccessToken}`) {
      throw createAxiosError(401, { code: "UNAUTHORIZED", message: "token expired" }, config);
    }
    if (authHeader === `Bearer ${refreshedAccessToken}`) {
      return { status: 200, data: { success: true, bookingId: "bk-123" }, config };
    }
    throw createAxiosError(400, { message: "unexpected token" }, config);
  };

  const response = await apiClient.post("/bookings", { ticketCount: 2 });
  assert.strictEqual(response.status, 200);
  assert.strictEqual(response.data.success, true);
  assert.strictEqual(refreshCallCount, 1);
  assert.strictEqual(protectedCallCount, 2);
  assert.strictEqual(useAuthStore.getState().accessToken, refreshedAccessToken);
});

// Scenario 4: Concurrent 401 responses share one refresh operation
test("Scenario 4: Concurrent 401 responses share one refresh operation", async () => {
  const user = { id: "user-concurrent", role: "CUSTOMER" };
  useAuthStore.getState().setAuth(user, "stale-access", "valid-refresh");

  let refreshCount = 0;
  const newAccessToken = createTestJwt({ userId: user.id, role: "CUSTOMER", exp: 2000000000 });

  rawAuthAxios.defaults.adapter = async (config) => {
    refreshCount++;
    await new Promise((resolve) => setTimeout(resolve, 30));
    return {
      status: 200,
      data: {
        access_token: newAccessToken,
        refresh_token: "rotated-refresh-new",
      },
      headers: {},
      config,
    };
  };

  apiClient.defaults.adapter = async (config) => {
    const authHeader = config.headers?.Authorization;
    if (authHeader === "Bearer stale-access") {
      throw createAxiosError(401, { message: "expired" }, config);
    }
    return { status: 200, data: { url: config.url }, config };
  };

  const [res1, res2, res3] = await Promise.all([
    apiClient.get("/events/1/seats"),
    apiClient.get("/events/1/pricing"),
    apiClient.get("/users/me/preferences"),
  ]);

  assert.strictEqual(res1.status, 200);
  assert.strictEqual(res2.status, 200);
  assert.strictEqual(res3.status, 200);
  assert.strictEqual(refreshCount, 1, "Only 1 refresh HTTP request must be made for concurrent 401s");
});

// Scenario 5: Rotated refresh tokens replace the previous tokens
test("Scenario 5: Rotated refresh tokens replace the previous tokens", async () => {
  const user = { id: "user-rot", role: "CUSTOMER" };
  useAuthStore.getState().setAuth(user, "acc-1", "rt-old");

  rawAuthAxios.defaults.adapter = async (config) => {
    const body = JSON.parse(config.data);
    assert.strictEqual(body.refreshToken, "rt-old");
    return {
      status: 200,
      data: {
        access_token: "acc-2",
        refresh_token: "rt-brand-new",
        user,
      },
      headers: {},
      config,
    };
  };

  const newAccess = await requestTokenRefresh();
  assert.strictEqual(newAccess, "acc-2");
  assert.strictEqual(useAuthStore.getState().refreshToken, "rt-brand-new");
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), "rt-brand-new");
});

// Scenario 6: Definitive refresh failure clears invalid authentication state
test("Scenario 6: Definitive refresh failure clears invalid authentication state", async () => {
  const user = { id: "user-def", role: "CUSTOMER" };
  useAuthStore.getState().setAuth(user, "acc-stale", "rt-revoked");

  rawAuthAxios.defaults.adapter = async (config) => {
    throw createAxiosError(401, { code: "UNAUTHORIZED", message: "invalid refresh token" }, config);
  };

  await assert.rejects(
    async () => {
      await requestTokenRefresh();
    },
    (err) => {
      assert.strictEqual(err.isDefinitiveAuthError, true);
      return true;
    }
  );

  const state = useAuthStore.getState();
  assert.strictEqual(state.isAuthenticated, false);
  assert.strictEqual(state.accessToken, null);
  assert.strictEqual(state.refreshToken, null);
  assert.strictEqual(state.user, null);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN), null);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), null);
});

// Scenario 7: Temporary network or 5xx failures preserve the session according to existing policy
test("Scenario 7: Temporary network or 5xx failures preserve the session", async () => {
  const user = { id: "user-transient", role: "CUSTOMER" };
  useAuthStore.getState().setAuth(user, "acc-saved", "rt-saved");

  rawAuthAxios.defaults.adapter = async (config) => {
    throw createAxiosError(500, { code: "INTERNAL_ERROR", message: "gateway timeout" }, config);
  };

  await assert.rejects(
    async () => {
      await requestTokenRefresh();
    },
    (err) => {
      assert.strictEqual(err.isTransientError, true);
      return true;
    }
  );

  const state = useAuthStore.getState();
  assert.strictEqual(state.isAuthenticated, true);
  assert.strictEqual(state.refreshToken, "rt-saved");
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), "rt-saved");
});

// Scenario 8: Logout during an in-flight refresh cannot restore the old session
test("Scenario 8: Logout during an in-flight refresh cannot restore old session", async () => {
  const user = { id: "user-logout-race", role: "CUSTOMER" };
  useAuthStore.getState().setAuth(user, "acc-before", "rt-before");

  rawAuthAxios.defaults.adapter = async (config) => {
    await new Promise((resolve) => setTimeout(resolve, 30));
    return {
      status: 200,
      data: {
        access_token: "acc-late-arrival",
        refresh_token: "rt-late-arrival",
      },
      headers: {},
      config,
    };
  };

  const refreshPromise = requestTokenRefresh();

  // User initiates logout while HTTP request is in transit
  useAuthStore.getState().clearAuth();
  assert.strictEqual(useAuthStore.getState().isAuthenticated, false);

  await assert.rejects(
    async () => {
      await refreshPromise;
    },
    (err) => {
      assert.strictEqual(err.isSessionAborted, true);
      return true;
    }
  );

  const state = useAuthStore.getState();
  assert.strictEqual(state.isAuthenticated, false);
  assert.strictEqual(state.accessToken, null);
  assert.strictEqual(state.refreshToken, null);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN), null);
});

// Scenario 9: Switching accounts during an in-flight request cannot overwrite the new account's state
test("Scenario 9: Switching accounts during in-flight request cannot overwrite new account state", async () => {
  const userA = { id: "user-A", email: "a@example.com", role: "CUSTOMER" };
  const userB = { id: "user-B", email: "b@example.com", role: "ORGANIZER" };

  useAuthStore.getState().setAuth(userA, "acc-A", "rt-A");

  rawAuthAxios.defaults.adapter = async (config) => {
    await new Promise((resolve) => setTimeout(resolve, 30));
    return {
      status: 200,
      data: {
        access_token: "acc-A-refreshed",
        refresh_token: "rt-A-rotated",
        user: userA,
      },
      headers: {},
      config,
    };
  };

  const refreshPromise = requestTokenRefresh();

  // Fast account switch: User B logs in before User A's refresh returns
  useAuthStore.getState().setAuth(userB, "acc-B", "rt-B");

  await assert.rejects(
    async () => {
      await refreshPromise;
    },
    (err) => {
      assert.strictEqual(err.isSessionAborted, true);
      return true;
    }
  );

  const finalState = useAuthStore.getState();
  assert.strictEqual(finalState.user.id, "user-B");
  assert.strictEqual(finalState.accessToken, "acc-B");
  assert.strictEqual(finalState.refreshToken, "rt-B");
  assert.strictEqual(finalState.user.role, "ORGANIZER");
});

// Scenario 10: Cross-tab login, logout, and token rotation synchronize correctly
test("Scenario 10: Cross-tab login, logout, and token rotation synchronize correctly", () => {
  // 1. Cross-tab login
  const remoteUser = { id: "user-tab-2", role: "CUSTOMER", email: "tab2@example.com" };
  localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, "acc-tab2");
  localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, "rt-tab2");
  localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(remoteUser));

  const loginEventResult = handleStorageEvent({ key: STORAGE_KEYS.ACCESS_TOKEN });
  assert.strictEqual(loginEventResult.synced, true);
  assert.strictEqual(loginEventResult.isAuthenticated, true);
  assert.strictEqual(useAuthStore.getState().user.id, "user-tab-2");

  // 2. Cross-tab token rotation
  localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, "acc-tab2-rotated");
  localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, "rt-tab2-rotated");
  const rotateEventResult = handleStorageEvent({ key: STORAGE_KEYS.REFRESH_TOKEN });
  assert.strictEqual(rotateEventResult.synced, true);
  assert.strictEqual(useAuthStore.getState().accessToken, "acc-tab2-rotated");

  // 3. Cross-tab logout (localStorage.clear sets event.key = null)
  localStorage.clear();
  const logoutEventResult = handleStorageEvent({ key: null });
  assert.strictEqual(logoutEventResult.synced, true);
  assert.strictEqual(logoutEventResult.isAuthenticated, false);
  const clearedState = useAuthStore.getState();
  assert.strictEqual(clearedState.isAuthenticated, false);
  assert.strictEqual(clearedState.user, null);
  assert.strictEqual(clearedState.accessToken, null);
  assert.strictEqual(clearedState.refreshToken, null);
  assert.strictEqual(clearedState.isInitializing, false);
});

// Scenario 11: Returning to an active tab revalidates the session (with cooldown)
test("Scenario 11: Returning to active tab revalidates session with cooldown throttling", async () => {
  const user = { id: "user-tab-return", role: "CUSTOMER" };
  useAuthStore.getState().setAuth(user, "acc-tab-active", "rt-tab-active");

  let revalCalls = 0;
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/auth/me")) {
      revalCalls++;
      return {
        status: 200,
        data: { user: { ...user, fullName: "Updated Alice" } },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  // First tab return
  const first = await handleVisibilityOrFocus({ now: 100000, cooldownMs: 10000 });
  assert.strictEqual(first.revalidated, true);
  assert.strictEqual(revalCalls, 1);

  // Rapid immediate re-focus within cooldown window (only 2 seconds later)
  const rapidSecond = await handleVisibilityOrFocus({ now: 102000, cooldownMs: 10000 });
  assert.strictEqual(rapidSecond.revalidated, false);
  assert.strictEqual(rapidSecond.reason, "cooldown_active");
  assert.strictEqual(revalCalls, 1, "Throttled: duplicate request prevented during cooldown");

  // Re-focus after cooldown expiry (11 seconds later)
  const third = await handleVisibilityOrFocus({ now: 111500, cooldownMs: 10000 });
  assert.strictEqual(third.revalidated, true);
  assert.strictEqual(revalCalls, 2);
});

// Scenario 12: Promotion from CUSTOMER to ORGANIZER with updated JWT claims
test("Scenario 12: Promotion from CUSTOMER to ORGANIZER with updated JWT claims", async () => {
  const user = { id: "user-promoted", role: "CUSTOMER", email: "promoted@example.com" };
  const initialJwt = createTestJwt({ userId: user.id, role: "CUSTOMER", exp: 2000000000 });
  const promotedJwt = createTestJwt({ userId: user.id, role: "ORGANIZER", exp: 2000000000 });

  useAuthStore.getState().setAuth(user, initialJwt, "rt-promoted");

  // Admin approves application in DB: /auth/me now returns role ORGANIZER
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/auth/me")) {
      return {
        status: 200,
        data: { user: { ...user, role: "ORGANIZER" } },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  // Backend refresh token endpoint returns fresh JWT containing role ORGANIZER
  rawAuthAxios.defaults.adapter = async (config) => {
    return {
      status: 200,
      data: {
        access_token: promotedJwt,
        refresh_token: "rt-promoted-rotated",
        user: { ...user, role: "ORGANIZER" },
      },
      headers: {},
      config,
    };
  };

  const revalResult = await useAuthStore.getState().revalidateSession();
  assert.strictEqual(revalResult.roleChanged, true);
  assert.strictEqual(revalResult.oldRole, "CUSTOMER");
  assert.strictEqual(revalResult.newRole, "ORGANIZER");

  const currentState = useAuthStore.getState();
  assert.strictEqual(currentState.user.role, "ORGANIZER");
  assert.strictEqual(currentState.accessToken, promotedJwt);

  const decodedClaims = decodeTokenPayload(currentState.accessToken);
  assert.strictEqual(decodedClaims.role, "ORGANIZER");
});

// Scenario 13: Access control: blocking CUSTOMER from allowedRoles={["ORGANIZER"]} while permitting ORGANIZER
test("Scenario 13: Access control blocks CUSTOMER from allowedRoles=['ORGANIZER'] and permits ORGANIZER", () => {
  const allowedRoles = ["ORGANIZER"];

  const checkAccess = (user, isAuthenticated, isInitializing) => {
    if (isInitializing) return "SPINNER";
    if (!isAuthenticated) return "REDIRECT_LOGIN";
    if (allowedRoles.length > 0 && (!user || !allowedRoles.includes(user.role))) {
      return "REDIRECT_UNAUTHORIZED";
    }
    return "PERMITTED";
  };

  // Unauthenticated user
  assert.strictEqual(checkAccess(null, false, false), "REDIRECT_LOGIN");

  // Hydrating session
  assert.strictEqual(checkAccess(null, false, true), "SPINNER");

  // Customer attempting to access organizer route
  const customer = { id: "cust-1", role: "CUSTOMER" };
  assert.strictEqual(checkAccess(customer, true, false), "REDIRECT_UNAUTHORIZED");

  // Promoted organizer accessing organizer route
  const organizer = { id: "org-1", role: "ORGANIZER" };
  assert.strictEqual(checkAccess(organizer, true, false), "PERMITTED");
});

// Scenario 14: Uninterrupted active customer checkout/browsing
test("Scenario 14: Uninterrupted active customer checkout/browsing", async () => {
  const customer = { id: "user-checkout", role: "CUSTOMER", email: "cust@example.com" };
  const currentToken = createTestJwt({ userId: customer.id, role: "CUSTOMER", exp: 2000000000 });
  useAuthStore.getState().setAuth(customer, currentToken, "rt-checkout");

  // Track active customer checkout state (e.g. selected seats, discount code, form fields)
  const clientCheckoutState = {
    selectedSeats: ["A1", "A2"],
    discountCode: "FESTIVE20",
    step: "PAYMENT",
    amount: 1500,
  };

  let revalCount = 0;
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/auth/me")) {
      revalCount++;
      return {
        status: 200,
        data: { user: customer },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  // Customer returns from checking notes or ticket details in another tab
  const revalResult = await useAuthStore.getState().revalidateSession();
  assert.strictEqual(revalResult.roleChanged, false);
  assert.strictEqual(revalResult.newRole, "CUSTOMER");
  assert.strictEqual(useAuthStore.getState().accessToken, currentToken, "Tokens are not mutated when role is unchanged");
  assert.strictEqual(useAuthStore.getState().user.role, "CUSTOMER");
  assert.deepStrictEqual(
    clientCheckoutState,
    {
      selectedSeats: ["A1", "A2"],
      discountCode: "FESTIVE20",
      step: "PAYMENT",
      amount: 1500,
    },
    "Client checkout progress must remain completely intact across background revalidations"
  );

  // Customer performs checkout booking submission with an access token that expires during action
  let checkoutAttempts = 0;
  const refreshedToken = createTestJwt({ userId: customer.id, role: "CUSTOMER", exp: 2000000000 });

  rawAuthAxios.defaults.adapter = async (config) => {
    return {
      status: 200,
      data: {
        access_token: refreshedToken,
        refresh_token: "rt-checkout-next",
        user: customer,
      },
      headers: {},
      config,
    };
  };

  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/bookings/checkout")) {
      checkoutAttempts++;
      const authHeader =
        (typeof config.headers?.get === "function" ? config.headers.get("Authorization") : null) ||
        config.headers?.Authorization ||
        config.headers?.authorization;
      if (checkoutAttempts === 1) {
        throw createAxiosError(401, { code: "UNAUTHORIZED", message: "token expired" }, config);
      }
      assert.strictEqual(authHeader, `Bearer ${refreshedToken}`, "Retried request must include refreshed token in Authorization header");
      return {
        status: 200,
        data: {
          success: true,
          bookingId: "booking-success-789",
          status: "CONFIRMED",
        },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  const checkoutResponse = await apiClient.post("/bookings/checkout", {
    eventId: "ev-100",
    seatCount: 2,
    amount: 1500,
  });

  assert.strictEqual(checkoutResponse.status, 200);
  assert.strictEqual(checkoutResponse.data.success, true);
  assert.strictEqual(checkoutResponse.data.bookingId, "booking-success-789");
  assert.strictEqual(checkoutAttempts, 2, "Checkout transparently retried with refreshed token");
  assert.strictEqual(useAuthStore.getState().accessToken, refreshedToken);
});
