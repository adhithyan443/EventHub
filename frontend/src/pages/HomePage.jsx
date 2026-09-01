import { useNavigate } from "react-router-dom";

import { logout } from "../api/authApi";
import useAuthStore from "../store/authStore";

export default function HomePage() {
    const navigate = useNavigate();

    const clearAuth = useAuthStore((state) => state.clearAuth);
    const user = useAuthStore((state) => state.user);

    async function handleLogout() {
        try {        
            await logout();
         
            clearAuth();

            navigate("/login", { replace: true });
        } catch (error) {
            console.error("Logout failed:", error);
           
            clearAuth();
            navigate("/login", { replace: true });
        }
    }

    return (
        <div className="min-h-screen flex flex-col items-center justify-center gap-4">
            <h1 className="text-3xl font-semibold">
                Welcome to EventHub
            </h1>

            {user && (
                <p className="text-ink/60">
                    Welcome, {user.fullName}
                </p>
            )}

            <button
                onClick={handleLogout}
                className="px-6 py-2 rounded-lg bg-primary text-white font-medium"
            >
                Logout
            </button>
        </div>
    );
}