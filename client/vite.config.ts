import react from "@vitejs/plugin-react";
import vike from "vike/plugin";
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [vike(), react(), tailwindcss()],
});
