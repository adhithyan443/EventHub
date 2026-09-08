import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";
import tailwindcss from '@tailwindcss/vite'

export default defineConfig(({ mode }) => {
  // eslint-disable-next-line no-undef
  const env = loadEnv(mode, path.resolve(__dirname, ".."), "");

  return {
    envDir: path.resolve(__dirname, ".."),
    plugins: [react(), tailwindcss()],
    define: {
      "import.meta.env.VITE_API_BASE_URL": JSON.stringify(
        env.VITE_API_BASE_URL
      ),
    },

    server: {
      watch: {
        usePolling: true,
        interval: 500,
      },
    },
  };
});








