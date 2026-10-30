import axios from "axios";
import useAuthStore, { STORAGE_KEYS } from "../store/authStore.js";

export const EXCLUDED_AUTH_ENDPOINTS = [
  "/auth/login",
  "/auth/refresh-token",
  "/auth/logout",
  "/auth/register",
  "/auth/verify-otp",
  "/auth/resend-otp",
  "/auth/forgot-password",
  "/auth/reset-password",
  "/auth/google",
];

export const isExcludedAuthEndpoint = (url) => {
  if (!url || typeof url !== "string") return false;
  return EXCLUDED_AUTH_ENDPOINTS.some((endpoint) => url.includes(endpoint));
};

export const getBaseURL = () =>
  (typeof import.meta !== "undefined" && import.meta.env?.VITE_API_BASE_URL) ||
  (typeof globalThis !== "undefined" && globalThis.process?.env?.VITE_API_BASE_URL) ||
  "http://localhost:8080/api/v1";

// Dedicated un-intercepted Axios instance for token refresh and auth operations
export const rawAuthAxios = axios.create({
  baseURL: getBaseURL(),
  headers: {
    "Content-Type": "application/json",
  },
});

const apiClient = axios.create({
  baseURL: getBaseURL(),
  headers: {
    "Content-Type": "application/json",
  },
});

let activeRefreshPromise = null;

export const getActiveRefreshPromise = () => activeRefreshPromise;

export const resetActiveRefreshPromiseForTesting = () => {
  activeRefreshPromise = null;
};

export const requestTokenRefresh = async () => {
  // If a refresh is already in-flight, reuse it so concurrent requests
  // never send separate refresh requests or trigger duplicate token rotation.
  if (activeRefreshPromise) {
    return activeRefreshPromise;
  }

  const startVersion = useAuthStore.getState().sessionVersion;
  const refreshToken =
    useAuthStore.getState().refreshToken ||
    localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);

  if (!refreshToken) {
    useAuthStore.getState().clearAuth();
    const error = new Error("No refresh token available");
    error.isDefinitiveAuthError = true;
    return Promise.reject(error);
  }

  activeRefreshPromise = (async () => {
    try {
      const res = await rawAuthAxios.post("/auth/refresh-token", {
        refreshToken,
      });

      const newAccessToken =
        res.data?.access_token || res.data?.accessToken;
      const newRefreshToken =
        res.data?.refresh_token || res.data?.refreshToken || refreshToken;

      if (!newAccessToken) {
        throw new Error("Invalid refresh response: missing access token");
      }

      // Verify session consistency: If user logged out or switched sessions during the HTTP call,
      // discard the new tokens immediately without touching store or storage.
      const currentState = useAuthStore.getState();
      if (
        currentState.sessionVersion !== startVersion ||
        !currentState.isAuthenticated
      ) {
        const abortedErr = new Error(
          "Session changed or logged out during token refresh"
        );
        abortedErr.isSessionAborted = true;
        throw abortedErr;
      }

      useAuthStore.getState().setTokens(newAccessToken, newRefreshToken);
      return newAccessToken;
    } catch (err) {
      if (err?.isSessionAborted) {
        throw err;
      }

      const status = err?.response?.status;
      // 401: Refresh token invalid, expired, or revoked.
      // 400 with VALIDATION_ERROR: Invalid/empty payload rejected by server.
      const isDefinitive =
        status === 401 ||
        (status === 400 &&
          (err?.response?.data?.code === "VALIDATION_ERROR" || !status));

      if (isDefinitive) {
        // Only clear auth if user session hasn't already been mutated/cleared
        if (useAuthStore.getState().sessionVersion === startVersion) {
          useAuthStore.getState().clearAuth();
        }
        err.isDefinitiveAuthError = true;
      } else {
        // Network errors, timeouts, or 5xx server issues are transient.
        // Keep stored authentication and localStorage intact.
        err.isTransientError = true;
      }

      throw err;
    } finally {
      activeRefreshPromise = null;
    }
  })();

  return activeRefreshPromise;
};

// Request interceptor: attach current access token and coordinate with in-flight refresh
apiClient.interceptors.request.use(
  async (config) => {
    const isAuthRoute = isExcludedAuthEndpoint(config.url);

    // If a refresh is currently running and this is not an auth route,
    // await the in-flight refresh so this request is dispatched with the fresh token.
    if (!isAuthRoute && activeRefreshPromise) {
      try {
        await activeRefreshPromise;
      } catch {
        // Ignore here; if refresh failed, request continues and is handled by response interceptor
      }
    }

    const accessToken =
      useAuthStore.getState().accessToken ||
      localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);

    if (accessToken) {
      if (typeof config.headers?.set === "function") {
        config.headers.set("Authorization", `Bearer ${accessToken}`);
      } else {
        config.headers = config.headers || {};
        config.headers.Authorization = `Bearer ${accessToken}`;
      }
    }

    // Allow the browser to set Content-Type with multipart boundary for FormData payloads
    if (config.data instanceof FormData) {
      if (typeof config.headers?.setContentType === "function") {
        config.headers.setContentType(null);
      } else if (config.headers) {
        config.headers["Content-Type"] = null;
      }
    }

    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor: automatically refresh token on 401 and retry original request
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    // Reject immediately if not 401, no request config, already retried, or excluded auth endpoint
    if (
      !error.response ||
      error.response.status !== 401 ||
      !originalRequest ||
      originalRequest._retry ||
      isExcludedAuthEndpoint(originalRequest.url)
    ) {
      return Promise.reject(error);
    }

    originalRequest._retry = true;

    try {
      const newAccessToken = await requestTokenRefresh();

      if (typeof originalRequest.headers?.set === "function") {
        originalRequest.headers.set(
          "Authorization",
          `Bearer ${newAccessToken}`
        );
      } else {
        originalRequest.headers = originalRequest.headers || {};
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
      }

      return apiClient(originalRequest);
    } catch (refreshErr) {
      return Promise.reject(refreshErr);
    }
  }
);

export default apiClient;
