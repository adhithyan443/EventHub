import apiClient from "./client"

export const register = async (data) => {
    const response = await apiClient.post("/auth/register", data);
    return response.data;
}

export const verifyOTP = async (data) => {
    const response = await apiClient.post("/auth/verify-otp", data);
    return response.data
}

export const resendOTP = async (email) => {
    const response = await apiClient.post("/auth/resend-otp", {
        email
    });

    return response.data;
}

export const login = async (data) => {
    const response = await apiClient.post("/auth/login", data);
    return response.data;
};

export const refreshToken = async (refreshToken) => {
    const response = await apiClient.post("/auth/refresh-token", {
        refreshToken,
    });
    return response.data;
};

export const logout = async () => {
    const response = await apiClient.post("/auth/logout");
    return response.data;
};

export const forgotPassword = async (email) => {
    const response = await apiClient.post("/auth/forgot-password", {
        email,
    });
    return response.data;
};

export const resetPassword = async (token, newPassword) => {
    const response = await apiClient.post("/auth/reset-password", {
        token,
        newPassword,
    });
    return response.data;
};

export const startGoogleLogin = () => {
    window.location.href =
        `${import.meta.env.VITE_API_BASE_URL}/auth/google`;
};


export const getCurrentUser = async (accessToken) => {
    const response = await apiClient.get("/auth/me", {
        headers: {
            Authorization: `Bearer ${accessToken}`,
        },
    });

    return response.data;
};