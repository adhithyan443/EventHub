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

// 2. Import application modules after localStorage is initialized
const { default: useAuthStore, STORAGE_KEYS } = await import("../../store/authStore.js");
const {
  default: apiClient,
  rawAuthAxios,
  requestTokenRefresh,
  resetActiveRefreshPromiseForTesting,
  isExcludedAuthEndpoint,
  EXCLUDED_AUTH_ENDPOINTS,
} = await import("../client.js");

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
  resetActiveRefreshPromiseForTesting();
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
});

test("isExcludedAuthEndpoint correctly identifies auth endpoints", () => {
  for (const endpoint of EXCLUDED_AUTH_ENDPOINTS) {
    assert.strictEqual(isExcludedAuthEndpoint(endpoint), true);
    assert.strictEqual(isExcludedAuthEndpoint(`http://localhost:8080/api/v1${endpoint}`), true);
  }
  assert.strictEqual(isExcludedAuthEndpoint("/events"), false);
  assert.strictEqual(isExcludedAuthEndpoint("/admin/users"), false);
  assert.strictEqual(isExcludedAuthEndpoint("/auth/me"), false);
});

test("successful token refresh: interceptor handles 401, refreshes tokens, updates store, and retries request", async () => {
  const initialUser = { id: "user-1", email: "user@example.com", role: "customer" };
  useAuthStore.getState().setAuth(initialUser, "expired-access-token", "valid-refresh-token");

  let refreshCalled = 0;
  let sentRefreshToken = null;

  rawAuthAxios.defaults.adapter = async (config) => {
    refreshCalled++;
    const payload = JSON.parse(config.data);
    sentRefreshToken = payload.refreshToken;

    return {
      status: 200,
      statusText: "OK",
      data: {
        access_token: "refreshed-access-token",
        refresh_token: "rotated-refresh-token",
      },
      headers: {},
      config,
    };
  };

  let eventCallAttempts = 0;
  const authHeadersSeen = [];

  apiClient.defaults.adapter = async (config) => {
    eventCallAttempts++;
    authHeadersSeen.push(config.headers.Authorization || config.headers.get?.("Authorization"));

    if (eventCallAttempts === 1) {
      throw createAxiosError(401, { message: "Token expired" }, config);
    }

    return {
      status: 200,
      statusText: "OK",
      data: { events: [{ id: 101, title: "Tech Meetup" }] },
      headers: {},
      config,
    };
  };

  const response = await apiClient.get("/events");

  assert.strictEqual(response.status, 200);
  assert.strictEqual(response.data.events.length, 1);
  assert.strictEqual(refreshCalled, 1);
  assert.strictEqual(sentRefreshToken, "valid-refresh-token");
  assert.strictEqual(eventCallAttempts, 2);
  assert.strictEqual(authHeadersSeen[0], "Bearer expired-access-token");
  assert.strictEqual(authHeadersSeen[1], "Bearer refreshed-access-token");

  // Verify Zustand store updated
  const storeState = useAuthStore.getState();
  assert.strictEqual(storeState.accessToken, "refreshed-access-token");
  assert.strictEqual(storeState.refreshToken, "rotated-refresh-token");
  assert.strictEqual(storeState.isAuthenticated, true);

  // Verify localStorage updated
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN), "refreshed-access-token");
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), "rotated-refresh-token");
});

test("token rotation: newly rotated refresh token replaces old refresh token in storage and store", async () => {
  useAuthStore.getState().setAuth({ id: "user-2" }, "acc-1", "old-refresh-token");

  rawAuthAxios.defaults.adapter = async (config) => {
    return {
      status: 200,
      statusText: "OK",
      data: {
        access_token: "acc-2",
        refresh_token: "new-rotated-refresh-token",
      },
      headers: {},
      config,
    };
  };

  const newAccess = await requestTokenRefresh();

  assert.strictEqual(newAccess, "acc-2");
  assert.strictEqual(useAuthStore.getState().refreshToken, "new-rotated-refresh-token");
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), "new-rotated-refresh-token");
});

test("concurrent 401 requests: multiple simultaneous requests share one in-flight refresh and all succeed", async () => {
  useAuthStore.getState().setAuth({ id: "user-3" }, "expired-token", "refresh-token-3");

  let refreshExecutionCount = 0;
  rawAuthAxios.defaults.adapter = async (config) => {
    refreshExecutionCount++;
    // Simulate real network delay for token exchange
    await new Promise((resolve) => setTimeout(resolve, 40));
    return {
      status: 200,
      statusText: "OK",
      data: {
        access_token: "shared-new-token",
        refresh_token: "shared-new-refresh",
      },
      headers: {},
      config,
    };
  };

  const attemptCounts = { req1: 0, req2: 0, req3: 0 };
  apiClient.defaults.adapter = async (config) => {
    const url = config.url;
    attemptCounts[url] = (attemptCounts[url] || 0) + 1;

    if (attemptCounts[url] === 1) {
      throw createAxiosError(401, { message: "Access token expired" }, config);
    }

    return {
      status: 200,
      statusText: "OK",
      data: { url, tokenUsed: config.headers.Authorization || config.headers.get?.("Authorization") },
      headers: {},
      config,
    };
  };

  // Launch 3 requests simultaneously
  const [res1, res2, res3] = await Promise.all([
    apiClient.get("req1"),
    apiClient.get("req2"),
    apiClient.get("req3"),
  ]);

  // CRITICAL: Refresh endpoint must have been called exactly ONCE across all 3 concurrent requests
  assert.strictEqual(refreshExecutionCount, 1, "Expected exactly 1 refresh call for all concurrent requests");

  assert.strictEqual(res1.data.tokenUsed, "Bearer shared-new-token");
  assert.strictEqual(res2.data.tokenUsed, "Bearer shared-new-token");
  assert.strictEqual(res3.data.tokenUsed, "Bearer shared-new-token");

  assert.strictEqual(useAuthStore.getState().accessToken, "shared-new-token");
  assert.strictEqual(useAuthStore.getState().refreshToken, "shared-new-refresh");
});

test("initialization coordination: request interceptor waits for in-flight refresh so concurrent requests use fresh token", async () => {
  useAuthStore.getState().setAuth({ id: "user-4" }, "expired-init-token", "init-refresh-token");

  let refreshCount = 0;
  rawAuthAxios.defaults.adapter = async (config) => {
    refreshCount++;
    await new Promise((resolve) => setTimeout(resolve, 30));
    return {
      status: 200,
      statusText: "OK",
      data: {
        access_token: "coordination-token",
        refresh_token: "coordination-refresh",
      },
      headers: {},
      config,
    };
  };

  let apiAttempts = 0;
  apiClient.defaults.adapter = async (config) => {
    apiAttempts++;
    const authHeader = config.headers.Authorization || config.headers.get?.("Authorization");
    assert.strictEqual(authHeader, "Bearer coordination-token");
    return {
      status: 200,
      data: { success: true, auth: authHeader },
      config,
    };
  };

  // 1. Start refresh as if triggered by initializeAuth
  const refreshPromise = requestTokenRefresh();

  // 2. While refresh is in-flight, an API call is dispatched
  const apiPromise = apiClient.get("/dashboard");

  const [refreshedToken, apiResponse] = await Promise.all([refreshPromise, apiPromise]);

  assert.strictEqual(refreshedToken, "coordination-token");
  assert.strictEqual(apiResponse.data.success, true);
  assert.strictEqual(refreshCount, 1, "Refresh endpoint must only be called once");
  assert.strictEqual(apiAttempts, 1, "Request awaited refresh and succeeded on first attempt with fresh token");
});

test("logout race protection: clearAuth during in-flight refresh discards tokens and preserves logged-out state", async () => {
  useAuthStore.getState().setAuth({ id: "user-5" }, "token-before-logout", "refresh-before-logout");

  let resolveRefresh;
  rawAuthAxios.defaults.adapter = () => {
    return new Promise((resolve) => {
      resolveRefresh = () =>
        resolve({
          status: 200,
          statusText: "OK",
          data: {
            access_token: "zombie-access-token",
            refresh_token: "zombie-refresh-token",
          },
          headers: {},
        });
    });
  };

  // 1. Refresh starts
  const refreshPromise = requestTokenRefresh();

  // 2. User explicitly logs out while refresh HTTP call is still in flight
  useAuthStore.getState().clearAuth();
  assert.strictEqual(useAuthStore.getState().isAuthenticated, false);
  assert.strictEqual(useAuthStore.getState().user, null);

  // 3. Backend refresh call finally resolves
  resolveRefresh();

  // 4. The refresh promise must reject with isSessionAborted
  await assert.rejects(async () => {
    await refreshPromise;
  }, (err) => {
    assert.strictEqual(err.isSessionAborted, true);
    return true;
  });

  // 5. Store and storage MUST remain completely empty (no zombie session restored!)
  const state = useAuthStore.getState();
  assert.strictEqual(state.isAuthenticated, false);
  assert.strictEqual(state.accessToken, null);
  assert.strictEqual(state.refreshToken, null);
  assert.strictEqual(state.user, null);

  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN), null);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), null);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.USER), null);
});

test("session version protection: newer login during in-flight refresh is not overwritten by older refresh", async () => {
  // User A starts session
  useAuthStore.getState().setAuth({ id: "user-A", name: "Alice" }, "token-A", "refresh-A");

  let resolveRefreshA;
  rawAuthAxios.defaults.adapter = () => {
    return new Promise((resolve) => {
      resolveRefreshA = () =>
        resolve({
          status: 200,
          data: {
            access_token: "refreshed-token-A",
            refresh_token: "refreshed-refresh-A",
          },
          headers: {},
        });
    });
  };

  // Refresh for User A starts
  const refreshPromiseA = requestTokenRefresh();

  // User A logs out and User B logs in before User A's refresh resolves
  useAuthStore.getState().clearAuth();
  useAuthStore.getState().setAuth({ id: "user-B", name: "Bob" }, "token-B", "refresh-B");

  // User A's HTTP call resolves
  resolveRefreshA();

  await assert.rejects(async () => {
    await refreshPromiseA;
  }, (err) => err.isSessionAborted === true);

  // Verify User B's session remains intact and was NOT overwritten
  const state = useAuthStore.getState();
  assert.strictEqual(state.user?.name, "Bob");
  assert.strictEqual(state.accessToken, "token-B");
  assert.strictEqual(state.refreshToken, "refresh-B");
});

test("transient error: 500 server error or network error marks error as transient and preserves stored session", async () => {
  useAuthStore.getState().setAuth({ id: "user-6" }, "access-6", "refresh-6");

  rawAuthAxios.defaults.adapter = async (config) => {
    throw createAxiosError(500, { message: "Internal server error" }, config);
  };

  await assert.rejects(async () => {
    await requestTokenRefresh();
  }, (err) => {
    assert.strictEqual(err.isTransientError, true);
    assert.strictEqual(err.isDefinitiveAuthError, undefined);
    return true;
  });

  // Session must NOT be cleared on transient failure!
  const state = useAuthStore.getState();
  assert.strictEqual(state.isAuthenticated, true);
  assert.strictEqual(state.accessToken, "access-6");
  assert.strictEqual(state.refreshToken, "refresh-6");
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), "refresh-6");
});

test("definitive failure: 401 from refresh endpoint clears session and marks definitive auth error", async () => {
  useAuthStore.getState().setAuth({ id: "user-7" }, "access-7", "revoked-refresh-7");

  rawAuthAxios.defaults.adapter = async (config) => {
    throw createAxiosError(401, { code: "UNAUTHORIZED", message: "invalid refresh token" }, config);
  };

  await assert.rejects(async () => {
    await requestTokenRefresh();
  }, (err) => {
    assert.strictEqual(err.isDefinitiveAuthError, true);
    return true;
  });

  // Session MUST be cleared on definitive failure
  const state = useAuthStore.getState();
  assert.strictEqual(state.isAuthenticated, false);
  assert.strictEqual(state.accessToken, null);
  assert.strictEqual(state.refreshToken, null);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN), null);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), null);
});

test("definitive failure: missing refresh token immediately clears session and rejects", async () => {
  useAuthStore.getState().setAuth({ id: "user-8" }, "access-only", null);
  mockStorage.clear();

  let refreshCalled = false;
  rawAuthAxios.defaults.adapter = async () => {
    refreshCalled = true;
    return { status: 200, data: {} };
  };

  await assert.rejects(async () => {
    await requestTokenRefresh();
  }, (err) => {
    assert.strictEqual(err.isDefinitiveAuthError, true);
    return true;
  });

  assert.strictEqual(refreshCalled, false, "Should not make network call if no refresh token exists");
  assert.strictEqual(useAuthStore.getState().isAuthenticated, false);
});

test("403 Forbidden responses bypass refresh logic entirely", async () => {
  useAuthStore.getState().setAuth({ id: "user-9" }, "access-9", "refresh-9");

  let refreshCalled = false;
  rawAuthAxios.defaults.adapter = async () => {
    refreshCalled = true;
    return { status: 200, data: {} };
  };

  apiClient.defaults.adapter = async (config) => {
    throw createAxiosError(403, { code: "FORBIDDEN", message: "Forbidden" }, config);
  };

  await assert.rejects(async () => {
    await apiClient.get("/admin/restricted");
  }, (err) => err.response?.status === 403);

  assert.strictEqual(refreshCalled, false, "403 response must not trigger refresh");
  assert.strictEqual(useAuthStore.getState().isAuthenticated, true);
});

test("excluded auth endpoints bypass refresh logic entirely on 401", async () => {
  let refreshCalled = false;
  rawAuthAxios.defaults.adapter = async () => {
    refreshCalled = true;
    return { status: 200, data: {} };
  };

  apiClient.defaults.adapter = async (config) => {
    throw createAxiosError(401, { code: "UNAUTHORIZED", message: "Invalid credentials" }, config);
  };

  await assert.rejects(async () => {
    await apiClient.post("/auth/login", { email: "a@b.com", password: "bad" });
  }, (err) => err.response?.status === 401);

  assert.strictEqual(refreshCalled, false, "Login endpoint 401 must not trigger refresh");
});

test("retry-loop prevention: retried request that returns 401 does not trigger second refresh", async () => {
  useAuthStore.getState().setAuth({ id: "user-10" }, "access-10", "refresh-10");

  let refreshCount = 0;
  rawAuthAxios.defaults.adapter = async (config) => {
    refreshCount++;
    return {
      status: 200,
      data: {
        access_token: "new-access-10",
        refresh_token: "new-refresh-10",
      },
      headers: {},
      config,
    };
  };

  let attempts = 0;
  apiClient.defaults.adapter = async (config) => {
    attempts++;
    // Always return 401 (e.g. user revoked in backend)
    throw createAxiosError(401, { message: "Still unauthorized" }, config);
  };

  await assert.rejects(async () => {
    await apiClient.get("/account/status");
  }, (err) => err.response?.status === 401);

  // Initial call (1) -> 401 -> refresh (1) -> retry (attempt 2) -> 401 -> HALT (no second refresh)
  assert.strictEqual(refreshCount, 1, "Refresh should only be attempted once, preventing infinite retry loop");
  assert.strictEqual(attempts, 2, "Request should only be attempted twice (initial + 1 retry)");
});

test("initializeAuth: transient refresh failure leaves stored tokens and sets isInitializing false", async () => {
  // Store valid refresh token in storage
  localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, "transient-refresh-token");
  localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify({ id: "init-user", role: "customer" }));
  useAuthStore.setState({
    accessToken: null,
    refreshToken: "transient-refresh-token",
    user: { id: "init-user", role: "customer" },
    isAuthenticated: false,
    isInitializing: true,
  });

  // Mock refresh failing with 500 error
  rawAuthAxios.defaults.adapter = async (config) => {
    throw createAxiosError(500, { message: "Server error" }, config);
  };

  await useAuthStore.getState().initializeAuth();

  // Initialization completed without wiping stored refresh token
  assert.strictEqual(useAuthStore.getState().isInitializing, false);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), "transient-refresh-token");
});

test("initializeAuth: definitive refresh failure clears storage and auth state", async () => {
  localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, "expired-refresh-token");
  useAuthStore.setState({
    accessToken: null,
    refreshToken: "expired-refresh-token",
    isAuthenticated: false,
    isInitializing: true,
  });

  rawAuthAxios.defaults.adapter = async (config) => {
    throw createAxiosError(401, { code: "UNAUTHORIZED", message: "invalid refresh token" }, config);
  };

  await useAuthStore.getState().initializeAuth();

  assert.strictEqual(useAuthStore.getState().isInitializing, false);
  assert.strictEqual(useAuthStore.getState().isAuthenticated, false);
  assert.strictEqual(localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN), null);
});
