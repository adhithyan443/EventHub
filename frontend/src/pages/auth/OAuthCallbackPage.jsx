import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { getCurrentUser } from "../../api/authApi";
import useAuthStore from "../../store/authStore";

export default function OAuthCallbackPage() {
    const navigate = useNavigate();

    const setAuth = useAuthStore((state) => state.setAuth);

    const [error, setError] = useState("");

    useEffect(() => {
        async function handleOAuthCallback() {
            try {
                // Read the tokens sent by the backend in the callback URL.
                const params = new URLSearchParams(window.location.search);

                const accessToken = params.get("access_token");
                const refreshToken = params.get("refresh_token");

                if (!accessToken || !refreshToken) {
                    throw new Error("Authentication tokens are missing");
                }

                // Use the access token to ask the backend who is logged in.
                const response = await getCurrentUser(accessToken);

                const user = response.user;

                // Store user and tokens in Zustand.
                setAuth(user, accessToken, refreshToken);

                // Navigate according to the user's role.
                switch (user.role) {
                    case "ADMIN":
                        navigate("/admin", { replace: true });
                        break;

                    case "ORGANIZER":
                        navigate("/organizer", { replace: true });
                        break;

                    case "CUSTOMER":
                        navigate("/events", { replace: true });
                        break;

                    default:
                        throw new Error("Invalid user role");
                }
            } catch (error) {
                console.error("Google authentication failed:", error);

                setError(
                    error.response?.data?.message ||
                    error.message ||
                    "Google authentication failed. Please try again."
                );
            }
        }

        handleOAuthCallback();
    }, [navigate, setAuth]);

    if (error) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="text-center">
                    <p className="text-red-500 mb-4">{error}</p>

                    <button
                        onClick={() => navigate("/login")}
                        className="text-primary font-semibold"
                    >
                        Back to Login
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen flex items-center justify-center">
            <p className="text-ink/60">
                Completing Google sign in...
            </p>
        </div>
    );
}