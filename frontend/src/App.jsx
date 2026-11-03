import { useEffect } from "react";
import { ToastContainer } from "react-toastify";
import AppRoutes from "./routes/AppRoutes";
import useAuthStore from "./store/authStore";
import useSessionSync from "./hooks/useSessionSync";

function App() {
  const initializeAuth = useAuthStore((state) => state.initializeAuth);

  // Restore authenticated session on application startup
  useEffect(() => {
    initializeAuth();
  }, [initializeAuth]);

  // Synchronize authentication state across browser tabs and revalidate on tab focus/visibility
  useSessionSync();

  return (
    <>
      <AppRoutes />
      <ToastContainer
        position="top-right"
        autoClose={4000}
        hideProgressBar={false}
        newestOnTop={true}
        closeOnClick={true}
        rtl={false}
        pauseOnFocusLoss={true}
        draggable={true}
        pauseOnHover={true}
        theme="light"
        limit={4}
      />
    </>
  );
}

export default App;