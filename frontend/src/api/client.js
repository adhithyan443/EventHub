import axios from "axios";
import useAuthStore, { STORAGE_KEYS } from "../store/authStore";

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  headers: {
    "Content-Type": "application/json",
  },
});

// Attach Authorization header from store or localStorage
apiClient.interceptors.request.use(
  (config) => {
    const accessToken =
      useAuthStore.getState().accessToken ||
      localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);

    if (accessToken) {
      config.headers.Authorization = `Bearer ${accessToken}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Refresh expired access token on 401 response and retry request
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    // Do not intercept auth endpoints to avoid infinite recursion
    const isAuthRoute =
      originalRequest?.url?.includes("/auth/login") ||
      originalRequest?.url?.includes("/auth/refresh-token") ||
      originalRequest?.url?.includes("/auth/logout");

    if (
      error.response?.status === 401 &&
      originalRequest &&
      !originalRequest._retry &&
      !isAuthRoute
    ) {
      originalRequest._retry = true;

      const refreshToken =
        useAuthStore.getState().refreshToken ||
        localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);

      if (refreshToken) {
        try {
          const res = await axios.post(
            `${import.meta.env.VITE_API_BASE_URL}/auth/refresh-token`,
            { refreshToken }
          );

          const newAccessToken = res.data?.access_token || res.data?.accessToken;
          const newRefreshToken =
            res.data?.refresh_token || res.data?.refreshToken || refreshToken;

          if (newAccessToken) {
            useAuthStore.getState().setTokens(newAccessToken, newRefreshToken);
            originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
            return apiClient(originalRequest);
          }
        } catch (refreshErr) {
          // Refresh token expired or revoked -> clear session
          useAuthStore.getState().clearAuth();
          return Promise.reject(refreshErr);
        }
      } else {
        useAuthStore.getState().clearAuth();
      }
    }

    return Promise.reject(error);
  }
);

export default apiClient;