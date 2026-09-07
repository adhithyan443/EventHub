import { Navigate, Outlet } from "react-router-dom";
import useAuthStore from "../../store/authStore";

export default function ProtectedRoute({ allowedRoles = [] }) {
  const isInitializing = useAuthStore((state) => state.isInitializing);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const user = useAuthStore((state) => state.user);

  // During authentication hydration, display minimal loading indicator
  // to avoid flash of login screen or premature unauthorized redirects
  if (isInitializing) {
    return (
      <div className="flex h-screen w-screen items-center justify-center bg-[#f1f3ff]">
        <div className="flex flex-col items-center gap-3">
          <div className="w-8 h-8 border-3 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
          <span className="text-xs font-medium text-[#3d4947]">Restoring session...</span>
        </div>
      </div>
    );
  }

  // Not authenticated -> redirect to login
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  // Role check: if specific roles are required and user role doesn't match
  if (allowedRoles.length > 0 && (!user || !allowedRoles.includes(user.role))) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
