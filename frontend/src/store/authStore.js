import { create } from "zustand";

const useAuthStore = create((set) => ({

    user: null,
    accessToken: null,
    refreshToken: null,
    isAuthenticated: false,

    setAuth: (user, accessToken, refreshToken) => {
        set({
            user,
            accessToken,
            refreshToken,
            isAuthenticated: true,
        });
    },

    clearAuth: () => {
        set({
            user: null,
            accessToken: null,
            refreshToken: null,
            isAuthenticated: false,
        });
    },
}));

export default useAuthStore;