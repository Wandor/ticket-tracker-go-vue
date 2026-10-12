import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

// The UI calls /api/* and the dev server forwards to the API, so no CORS setup is needed.
const apiUrl = process.env.API_URL ?? "http://localhost:8080";

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      "/api": { target: apiUrl, rewrite: (path) => path.replace(/^\/api/, "") },
    },
  },
});
