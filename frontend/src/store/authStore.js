import { create } from "zustand";

export const STORAGE_KEYS = {
  ACCESS_TOKEN: "eventhub_access_token",
  REFRESH_TOKEN: "eventhub_refresh_token",
  USER: "eventhub_user",
};

// Safely read persisted session from localStorage
export const getStoredAuth = () => {
  try {
    const accessToken = localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
    const refreshToken = localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
    const rawUser = localStorage.getItem(STORAGE_KEYS.USER);
    const user = rawUser ? JSON.parse(rawUser) : null;
    return { accessToken, refreshToken, user };
  } catch {
    return { accessToken: null, refreshToken: null, user: null };
  }
};

// Safely persist session to localStorage
export const setStoredAuth = (user, accessToken, refreshToken) => {
  try {
    if (accessToken) localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, accessToken);
    else localStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);

    if (refreshToken) localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, refreshToken);
    else localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);

    if (user) localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(user));
    else localStorage.removeItem(STORAGE_KEYS.USER);
  } catch {
    // Ignore storage write errors (e.g. storage quota, private mode)
  }
};

// Safely clear session from localStorage
export const clearStoredAuth = () => {
  try {
    localStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);
    localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
    localStorage.removeItem(STORAGE_KEYS.USER);
  } catch {
    // Ignore storage deletion errors
  }
};

// Helper to inspect JWT expiration without external libraries
export const isTokenExpired = (token) => {
  if (!token || typeof token !== "string") return true;
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return true;
    const base64Url = parts[1];
    const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split("")
        .map((c) => "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2))
        .join("")
    );
    const { exp } = JSON.parse(jsonPayload);
    if (!exp) return false;
    // 10-second grace window to refresh before actual expiration
    return exp * 1000 < Date.now() + 10000;
  } catch {
    return true;
  }
};

const initialStored = getStoredAuth();

const useAuthStore = create((set, get) => ({
  user: initialStored.user,
  accessToken: initialStored.accessToken,
  refreshToken: initialStored.refreshToken,
  isAuthenticated: Boolean(initialStored.accessToken && initialStored.user),
  // If tokens exist in storage, start in initializing state until verified
  isInitializing: Boolean(initialStored.accessToken || initialStored.refreshToken),

  setAuth: (user, accessToken, refreshToken) => {
    setStoredAuth(user, accessToken, refreshToken);
    set({
      user,
      accessToken,
      refreshToken,
      isAuthenticated: true,
      isInitializing: false,
    });
  },

  setTokens: (accessToken, refreshToken) => {
    const currentUser = get().user;
    setStoredAuth(currentUser, accessToken, refreshToken);
    set({
      accessToken,
      refreshToken: refreshToken || get().refreshToken,
      isAuthenticated: true,
    });
  },

  clearAuth: () => {
    clearStoredAuth();
    set({
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
      isInitializing: false,
    });
  },

  // Asynchronously hydrate and validate the stored session on application startup
  initializeAuth: async () => {
    const stored = getStoredAuth();

    // 1. If no tokens exist, mark initialization complete immediately
    if (!stored.accessToken && !stored.refreshToken) {
      get().clearAuth();
      return;
    }

    try {
      // Dynamically import API helpers to prevent circular module dependencies
      const { getCurrentUser, refreshToken: refreshApi } = await import("../api/authApi");

      // 2. If access token is present and not expired, verify with backend /auth/me
      if (stored.accessToken && !isTokenExpired(stored.accessToken)) {
        try {
          const meResponse = await getCurrentUser(stored.accessToken);
          if (meResponse?.user) {
            get().setAuth(meResponse.user, stored.accessToken, stored.refreshToken);
            return;
          }
        } catch (meError) {
          // If not a 401 error (e.g. temporary network offline), preserve existing stored state
          if (meError?.response?.status !== 401) {
            set({ isInitializing: false });
            return;
          }
          // If 401, access token is invalid/revoked -> fall through to refresh
        }
      }

      // 3. If access token is expired or returned 401, attempt refresh token exchange
      if (stored.refreshToken) {
        try {
          const refreshResponse = await refreshApi(stored.refreshToken);
          const newAccessToken =
            refreshResponse?.access_token || refreshResponse?.accessToken;
          const newRefreshToken =
            refreshResponse?.refresh_token || refreshResponse?.refreshToken || stored.refreshToken;

          if (newAccessToken) {
            // Verify new user profile with refreshed access token
            let user = stored.user;
            try {
              const userRes = await getCurrentUser(newAccessToken);
              if (userRes?.user) user = userRes.user;
            } catch {
              // Keep cached user if /me temporarily fails
            }

            get().setAuth(user, newAccessToken, newRefreshToken);
            return;
          }
        } catch {
          // Refresh token is expired or revoked by backend
          get().clearAuth();
          return;
        }
      }

      // If all attempts fail, clear session
      get().clearAuth();
    } catch {
      // In case of unexpected initialization failure, clear session safely
      get().clearAuth();
    }
  },
}));

export default useAuthStore;