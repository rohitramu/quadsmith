import crypto from "crypto";
import fs from "fs";
import path from "path";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

function computeWasmHash(): string {
  const hash = crypto.createHash("sha256");
  let found = false;
  for (const file of ["quadsmith-engine.wasm", "wasm_exec.js"]) {
    const filePath = path.resolve(import.meta.dirname, "public", file);
    if (fs.existsSync(filePath)) {
      hash.update(fs.readFileSync(filePath));
      found = true;
    }
  }
  return found ? hash.digest("hex").slice(0, 16) : "";
}

const wasmHash = computeWasmHash();

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  define: {
    __WASM_HASH__: JSON.stringify(wasmHash),
  },
  resolve: {
    alias: {
      "@quadsmith/shared": path.resolve(import.meta.dirname, "../shared/src"),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules")) {
            return "vendor";
          }
        },
      },
    },
  },
});
