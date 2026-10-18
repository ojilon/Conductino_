import path from "path";
import { fileURLToPath } from "url";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { viteSingleFile } from "vite-plugin-singlefile";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss(), viteSingleFile()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  server: {
    // Listen on every interface and accept proxied hostnames so the dev
    // server works inside sandboxes/previews (proxied under *.e2b.app etc.).
    // Dev-only setting — the production build is a single static file.
    host: "0.0.0.0",
    port: 5173,
    allowedHosts: true,
  },
});
