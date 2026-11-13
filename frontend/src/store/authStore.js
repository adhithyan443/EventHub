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

// Helper to decode JWT payload without external libraries
export const decodeTokenPayload = (token) => {
  if (!token || typeof token !== "string") return null;
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    let base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    while (base64.length % 4 !== 0) {
      base64 += "=";
    }
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split("")
        .map((c) => "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2))
        .join("")
    );
    return JSON.parse(jsonPayload);
  } catch {
    return null;
  }
};

// Helper to inspect JWT expiration without external libraries
export const isTokenExpired = (token) => {
  const payload = decodeTokenPayload(token);
  if (!payload) return true;
  if (!payload.exp) return false;
  // 10-second grace window to refresh before actual expiration
  return payload.exp * 1000 < Date.now() + 10000;
};

const initialStored = getStoredAuth();

let activeRevalidatePromise = null;

export const resetActiveRevalidatePromiseForTesting = () => {
  activeRevalidatePromise = null;
};

const useAuthStore = create((set, get) => ({
  user: initialStored.user,
  accessToken: initialStored.accessToken,
  refreshToken: initialStored.refreshToken,
  isAuthenticated: Boolean(initialStored.accessToken && initialStored.user),
  // If tokens exist in storage, start in initializing state until verified
  isInitializing: Boolean(initialStored.accessToken || initialStored.refreshToken),
  sessionVersion: 0,

  setAuth: (user, accessToken, refreshToken) => {
    setStoredAuth(user, accessToken, refreshToken);
    set((state) => ({
      user,
      accessToken,
      refreshToken,
      isAuthenticated: true,
      isInitializing: false,
      sessionVersion: state.sessionVersion + 1,
    }));
  },

  setTokens: (accessToken, refreshToken, updatedUser) => {
    let currentUser = updatedUser || get().user;
    if (!updatedUser && currentUser && accessToken) {
      const payload = decodeTokenPayload(accessToken);
      if (payload?.role && payload.role !== currentUser.role) {
        currentUser = { ...currentUser, role: payload.role };
      }
    }
    setStoredAuth(currentUser, accessToken, refreshToken);
    set((state) => ({
      user: currentUser,
      accessToken,
      refreshToken: refreshToken || state.refreshToken,
      isAuthenticated: true,
      isInitializing: false,
      sessionVersion: state.sessionVersion + 1,
    }));
  },

  clearAuth: () => {
    clearStoredAuth();
    set((state) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
      isInitializing: false,
      sessionVersion: state.sessionVersion + 1,
    }));
  },

  updateUser: (partialUser) => {
    const current = get();
    if (!current.user) return;
    const profileImage =
      partialUser.profile_image ??
      partialUser.profileImage ??
      partialUser.ProfileImage ??
      current.user.profileImage ??
      current.user.ProfileImage ??
      "";
    const mergedUser = {
      ...current.user,
      ...partialUser,
      profileImage,
      ProfileImage: profileImage,
    };
    setStoredAuth(mergedUser, current.accessToken, current.refreshToken);
    set((state) => ({
      user: mergedUser,
      sessionVersion: state.sessionVersion + 1,
    }));
    return mergedUser;
  },

  // Synchronize in-memory auth state with localStorage across browser tabs
  syncFromStorage: () => {
    const stored = getStoredAuth();
    const current = get();

    // 1. If storage has no session, clear current state if authenticated
    if (!stored.accessToken || !stored.refreshToken || !stored.user) {
      if (current.isAuthenticated) {
        get().clearAuth();
        return { synced: true, isAuthenticated: false };
      }
      return { synced: false, isAuthenticated: false };
    }

    // 2. Storage has tokens and user. Check if anything changed
    const isTokenChanged =
      stored.accessToken !== current.accessToken ||
      stored.refreshToken !== current.refreshToken;
    const isUserChanged =
      JSON.stringify(stored.user) !== JSON.stringify(current.user);

    if (isTokenChanged || isUserChanged || !current.isAuthenticated) {
      const oldRole = current.user?.role;
      const newRole = stored.user?.role;
      const roleChanged = Boolean(oldRole && newRole && oldRole !== newRole);

      set((state) => ({
        user: stored.user,
        accessToken: stored.accessToken,
        refreshToken: stored.refreshToken,
        isAuthenticated: true,
        isInitializing: false,
        sessionVersion: state.sessionVersion + 1,
      }));

      return {
        synced: true,
        isAuthenticated: true,
        roleChanged,
        oldRole,
        newRole,
        user: stored.user,
        accessToken: stored.accessToken,
      };
    }

    return { synced: false, isAuthenticated: true };
  },

  // Authoritatively revalidate the active session against the backend
  revalidateSession: async () => {
    if (activeRevalidatePromise) {
      return activeRevalidatePromise;
    }

    const state = get();
    if (!state.isAuthenticated || !state.accessToken) {
      return { roleChanged: false, isAuthenticated: false };
    }

    activeRevalidatePromise = (async () => {
      try {
        const { getCurrentUser } = await import("../api/authApi.js");
        const { requestTokenRefresh } = await import("../api/client.js");

        const meResponse = await getCurrentUser(state.accessToken);
        const dbUser = meResponse?.user;

        if (!dbUser) {
          return { roleChanged: false, isAuthenticated: true };
        }

        // Check if user logged out or switched while /auth/me was in-flight
        const currentState = get();
        if (!currentState.isAuthenticated || !currentState.user || currentState.user.id !== state.user?.id) {
          return { roleChanged: false, isSessionAborted: true };
        }

        const currentRole = currentState.user?.role;
        const newRole = dbUser.role;

        if (currentRole && newRole && currentRole !== newRole) {
          // Role mismatch: request fresh tokens minted with the updated database role
          const newAccessToken = await requestTokenRefresh();
          const refreshedState = get();

          return {
            roleChanged: true,
            oldRole: currentRole,
            newRole,
            user: refreshedState.user || { ...currentState.user, ...dbUser },
            accessToken: newAccessToken || refreshedState.accessToken,
          };
        }

        // Role has not changed; sync any updated user fields (e.g. name, phone, status)
        const mergedUser = { ...currentState.user, ...dbUser };
        setStoredAuth(mergedUser, currentState.accessToken, currentState.refreshToken);
        set({ user: mergedUser });

        return {
          roleChanged: false,
          oldRole: currentRole,
          newRole: currentRole,
          user: mergedUser,
          accessToken: currentState.accessToken,
        };
      } catch (err) {
        return {
          roleChanged: false,
          error: err,
        };
      } finally {
        activeRevalidatePromise = null;
      }
    })();

    return activeRevalidatePromise;
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
      const { getCurrentUser } = await import("../api/authApi.js");
      const { requestTokenRefresh } = await import("../api/client.js");

      // 2. If access token is present and not expired, verify with backend /auth/me
      if (stored.accessToken && !isTokenExpired(stored.accessToken)) {
        try {
          const meResponse = await getCurrentUser(stored.accessToken);
          if (meResponse?.user) {
            // Check if backend role differs from token claims or stored user
            const tokenClaims = decodeTokenPayload(stored.accessToken);
            const tokenRole = tokenClaims?.role;
            const dbRole = meResponse.user?.role;
            const storedRole = stored.user?.role;

            if (dbRole && ((tokenRole && tokenRole !== dbRole) || (storedRole && storedRole !== dbRole))) {
              if (stored.refreshToken) {
                try {
                  const refreshedToken = await requestTokenRefresh();
                  if (refreshedToken) {
                    set({ isInitializing: false });
                    return;
                  }
                } catch {
                  // Fall back to setting auth with meResponse user if refresh encounters transient error
                }
              }
            }

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
          const newAccessToken = await requestTokenRefresh();
          if (newAccessToken) {
            // Verify new user profile with refreshed access token
            let user = get().user || stored.user;
            try {
              const userRes = await getCurrentUser(newAccessToken);
              if (userRes?.user) user = userRes.user;
            } catch {
              // Keep cached user if /me temporarily fails
            }

            const currentStore = get();
            get().setAuth(
              user,
              currentStore.accessToken || newAccessToken,
              currentStore.refreshToken || stored.refreshToken
            );
            return;
          }
        } catch (refreshErr) {
          // Distinguish transient vs definitive failure
          if (refreshErr?.isTransientError) {
            // Temporary network/server issue: keep stored tokens intact
            set({ isInitializing: false });
            return;
          }
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