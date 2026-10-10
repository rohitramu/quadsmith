import { toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  type EvaluateComponentsRequest,
  type EvaluateBuildResponse,
  type GetComponentsElectricalLimitsRequest,
  type GetBuildElectricalLimitsResponse,
  EvaluateComponentsRequestSchema,
  EvaluateBuildResponseSchema,
  GetComponentsElectricalLimitsRequestSchema,
  GetBuildElectricalLimitsResponseSchema,
} from "../gen/quadsmith/evaluator_pb";
import {
  type CheckComponentsCompatibilityRequest,
  type CheckCompatibilityResponse,
  CheckComponentsCompatibilityRequestSchema,
  CheckCompatibilityResponseSchema,
} from "../gen/quadsmith/compatibility_pb";
import {
  type AssembledComponents,
  AssembledComponentsSchema,
} from "../gen/quadsmith/components_pb";
import { useState, useEffect } from "react";

export interface ProgressInfo {
  loadedBytes: number;
  totalBytes: number;
  percent: number;
}

export type WasmEngineStatus = "idle" | "loading" | "ready" | "error";

interface GoInstance {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
}

declare global {
  interface Window {
    Go?: new () => GoInstance;
    __quadsmith_engine?: {
      evaluateComponents(reqBytes: Uint8Array): Uint8Array;
      computeElectricalLimits(reqBytes: Uint8Array): Uint8Array;
      checkCompatibility(reqBytes: Uint8Array): Uint8Array;
      generateCelFilter(targetCollection: string, compsBytes: Uint8Array): string;
      ready: boolean;
    };
  }
}

class WasmEngineClient {
  private status: WasmEngineStatus = "idle";
  private isDelayed = false;
  private progress: ProgressInfo | null = null;
  private error: string | null = null;
  private loadPromise: Promise<void> | null = null;
  private subscribers = new Set<() => void>();

  public getStatus() {
    return this.status;
  }

  public getIsReady() {
    return this.status === "ready";
  }

  public getIsDelayed() {
    return this.isDelayed;
  }

  public getProgress() {
    return this.progress;
  }

  public getError() {
    return this.error;
  }

  public subscribe(callback: () => void) {
    this.subscribers.add(callback);
    return () => {
      this.subscribers.delete(callback);
    };
  }

  private notify() {
    this.subscribers.forEach((cb) => cb());
  }

  private async loadWasmExecScript(): Promise<void> {
    if (window.Go) return;

    return new Promise((resolve, reject) => {
      const existing = document.querySelector('script[src*="wasm_exec.js"]');
      if (existing) {
        existing.addEventListener("load", () => resolve());
        existing.addEventListener("error", (e) => reject(e));
        return;
      }

      const script = document.createElement("script");
      script.src = "/wasm_exec.js";
      script.async = true;
      script.onload = () => resolve();
      script.onerror = (err) => reject(new Error(`Failed to load wasm_exec.js: ${err}`));
      document.head.appendChild(script);
    });
  }

  public async load(): Promise<void> {
    if (this.status === "ready") return;
    if (this.loadPromise) return this.loadPromise;

    this.status = "loading";
    this.error = null;
    this.notify();

    // 3-second delay timer to display loading progress indicator
    const delayTimer = setTimeout(() => {
      if (this.status === "loading") {
        this.isDelayed = true;
        this.notify();
      }
    }, 3000);

    this.loadPromise = (async () => {
      try {
        await this.loadWasmExecScript();

        if (!window.Go) {
          throw new Error("Go runtime constructor wasm_exec.js not available");
        }

        const response = await fetch("/quadsmith-engine.wasm");
        if (!response.ok) {
          throw new Error(`Failed to fetch quadsmith-engine.wasm: HTTP ${response.status}`);
        }

        const contentLength = response.headers.get("content-length");
        const total = contentLength ? parseInt(contentLength, 10) : 16 * 1024 * 1024; // fallback ~16MB

        let wasmBytes: Uint8Array;
        if (response.body) {
          const reader = response.body.getReader();
          const chunks: Uint8Array[] = [];
          let loaded = 0;

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            if (value) {
              chunks.push(value);
              loaded += value.length;
              const percent = Math.min(100, Math.round((loaded / total) * 100));
              this.progress = {
                loadedBytes: loaded,
                totalBytes: total,
                percent,
              };
              this.notify();
            }
          }

          wasmBytes = new Uint8Array(loaded);
          let offset = 0;
          for (const chunk of chunks) {
            wasmBytes.set(chunk, offset);
            offset += chunk.length;
          }
        } else {
          const buffer = await response.arrayBuffer();
          wasmBytes = new Uint8Array(buffer);
        }

        const go = new window.Go();
        const compiled = (await WebAssembly.instantiate(wasmBytes, go.importObject)) as any;
        const instance = compiled.instance || compiled;
        go.run(instance);

        this.status = "ready";
        this.isDelayed = false;
        clearTimeout(delayTimer);
        this.notify();
      } catch (err: any) {
        clearTimeout(delayTimer);
        this.status = "error";
        this.error = err?.message || String(err);
        this.notify();
        throw err;
      }
    })();

    return this.loadPromise;
  }

  public evaluateComponents(req: EvaluateComponentsRequest): EvaluateBuildResponse {
    if (!window.__quadsmith_engine) {
      throw new Error("Quadsmith WASM Engine not initialized");
    }
    const bytes = toBinary(EvaluateComponentsRequestSchema, req);
    const out = window.__quadsmith_engine.evaluateComponents(bytes);
    if (typeof out === "string") {
      throw new Error(out);
    }
    return fromBinary(EvaluateBuildResponseSchema, out);
  }

  public computeElectricalLimits(
    req: GetComponentsElectricalLimitsRequest,
  ): GetBuildElectricalLimitsResponse {
    if (!window.__quadsmith_engine) {
      throw new Error("Quadsmith WASM Engine not initialized");
    }
    const bytes = toBinary(GetComponentsElectricalLimitsRequestSchema, req);
    const out = window.__quadsmith_engine.computeElectricalLimits(bytes);
    if (typeof out === "string") {
      throw new Error(out);
    }
    return fromBinary(GetBuildElectricalLimitsResponseSchema, out);
  }

  public checkCompatibility(req: CheckComponentsCompatibilityRequest): CheckCompatibilityResponse {
    if (!window.__quadsmith_engine) {
      throw new Error("Quadsmith WASM Engine not initialized");
    }
    const bytes = toBinary(CheckComponentsCompatibilityRequestSchema, req);
    const out = window.__quadsmith_engine.checkCompatibility(bytes);
    if (typeof out === "string") {
      throw new Error(out);
    }
    return fromBinary(CheckCompatibilityResponseSchema, out);
  }

  public generateCelFilter(targetCollection: string, comps: AssembledComponents): string {
    if (!window.__quadsmith_engine) {
      return "";
    }
    const bytes = toBinary(AssembledComponentsSchema, comps);
    return window.__quadsmith_engine.generateCelFilter(targetCollection, bytes);
  }
}

export const wasmEngine = new WasmEngineClient();

export function useWasmEngine() {
  const [, setTick] = useState(0);

  useEffect(() => {
    // Start loading on demand
    wasmEngine.load().catch(() => {});
    return wasmEngine.subscribe(() => {
      setTick((t) => t + 1);
    });
  }, []);

  return {
    engine: wasmEngine,
    isReady: wasmEngine.getIsReady(),
    isDelayed: wasmEngine.getIsDelayed(),
    progress: wasmEngine.getProgress(),
    error: wasmEngine.getError(),
    status: wasmEngine.getStatus(),
  };
}
