import { useEffect } from "react";
import useAuthStore, { STORAGE_KEYS } from "../store/authStore.js";

export const REVALIDATION_COOLDOWN_MS = 10000;

let lastRevalidateTime = 0;

export const resetLastRevalidateTimeForTesting = () => {
  lastRevalidateTime = 0;
};

export const getLastRevalidateTimeForTesting = () => lastRevalidateTime;

/**
 * Handles browser `storage` events fired across other same-origin tabs.
 * Synchronizes in-memory auth state whenever authentication tokens or user objects change.
 */
export const handleStorageEvent = (event) => {
  // When localStorage.clear() is called, event.key is null per Web Storage spec.
  // Otherwise, only sync if the mutated key is an EventHub auth storage key.
  if (!event || !event.key || Object.values(STORAGE_KEYS).includes(event.key)) {
    return useAuthStore.getState().syncFromStorage();
  }
  return { synced: false, ignored: true };
};

/**
 * Handles application active lifecycle events (`visibilitychange` and `focus`).
 * Revalidates the session against the backend when returning to an active tab,
 * applying a cooldown window and in-flight guard to eliminate duplicate requests.
 */
export const handleVisibilityOrFocus = async (options = {}) => {
  const cooldown = options.cooldownMs ?? REVALIDATION_COOLDOWN_MS;
  const now = options.now ?? Date.now();

  // If document is explicitly hidden (tab in background), skip revalidation
  if (typeof document !== "undefined" && document.visibilityState && document.visibilityState === "hidden") {
    return { revalidated: false, reason: "document_hidden" };
  }

  const { isAuthenticated, isInitializing } = useAuthStore.getState();
  if (!isAuthenticated || isInitializing) {
    return { revalidated: false, reason: "unauthenticated_or_initializing" };
  }

  // Throttle revalidation to prevent rapid duplicate calls when focus and visibility fire together
  if (!options.force && now - lastRevalidateTime < cooldown) {
    return { revalidated: false, reason: "cooldown_active" };
  }

  lastRevalidateTime = now;

  try {
    const result = await useAuthStore.getState().revalidateSession();
    return { revalidated: true, result };
  } catch (err) {
    return { revalidated: false, error: err };
  }
};

/**
 * Subscribes to window storage, document visibilitychange, and window focus events.
 * Returns an unbind cleanup function.
 */
export const setupSessionSync = (options = {}) => {
  if (typeof window === "undefined") {
    return () => {};
  }

  const storageListener = (e) => handleStorageEvent(e);
  const visibilityListener = () => handleVisibilityOrFocus(options);
  const focusListener = () => handleVisibilityOrFocus(options);

  window.addEventListener("storage", storageListener);
  if (typeof document !== "undefined" && typeof document.addEventListener === "function") {
    document.addEventListener("visibilitychange", visibilityListener);
  }
  window.addEventListener("focus", focusListener);

  return () => {
    window.removeEventListener("storage", storageListener);
    if (typeof document !== "undefined" && typeof document.removeEventListener === "function") {
      document.removeEventListener("visibilitychange", visibilityListener);
    }
    window.removeEventListener("focus", focusListener);
  };
};

/**
 * React hook mounted at application level to keep authentication state
 * synchronized across browser tabs and revalidated on active user return.
 */
export default function useSessionSync() {
  useEffect(() => {
    return setupSessionSync();
  }, []);
}
