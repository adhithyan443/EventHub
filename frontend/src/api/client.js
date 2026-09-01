import axios from "axios"
import useAuthStore from "../store/authStore";

const apiClient = axios.create({
    baseURL: import.meta.env.VITE_API_BASE_URL,
    headers: {
        "Content-Type": "application/json"
    },
});


apiClient.interceptors.request.use(
    (config) => {
        const accessToken = useAuthStore.getState().accessToken;

        if (accessToken) {
            config.headers.Authorization = `Bearer ${accessToken}`;
        }
        return config
    },

    (error) => {
        return Promise.reject(error);
    }
)

export default apiClient;