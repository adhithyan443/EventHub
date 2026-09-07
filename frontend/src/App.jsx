import { useEffect } from "react";
import AppRoutes from "./routes/AppRoutes";
import useAuthStore from "./store/authStore";

function App() {
  const initializeAuth = useAuthStore((state) => state.initializeAuth);

  // Restore authenticated session on application startup
  useEffect(() => {
    initializeAuth();
  }, [initializeAuth]);

  return <AppRoutes />;
}

export default App;