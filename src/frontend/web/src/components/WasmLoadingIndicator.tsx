import { Loader2, Download } from "lucide-react";
import type { ProgressInfo } from "../lib/wasmEngine";

export interface WasmLoadingIndicatorProps {
  progress: ProgressInfo | null;
  className?: string;
}

export function WasmLoadingIndicator({ progress, className = "" }: WasmLoadingIndicatorProps) {
  const percent = progress?.percent ?? 0;
  const loadedMb = progress ? (progress.loadedBytes / (1024 * 1024)).toFixed(2) : "0.00";
  const totalMb = progress ? (progress.totalBytes / (1024 * 1024)).toFixed(2) : "16.00";

  return (
    <div
      className={`bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl p-6 shadow-sm relative overflow-hidden transition-all duration-300 ${className}`}
    >
      {/* Top Accent Strip */}
      <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500" />

      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-indigo-100 dark:bg-indigo-950 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
            <Loader2 className="w-4 h-4 animate-spin" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-zinc-900 dark:text-zinc-100 leading-tight">
              Live Build Evaluator
            </h3>
            <p className="text-[11px] text-zinc-500 dark:text-zinc-400">
              Initializing WebAssembly Core
            </p>
          </div>
        </div>
        <span className="text-[10px] uppercase font-mono tracking-wider font-semibold text-amber-700 dark:text-amber-400 bg-amber-500/10 px-2.5 py-0.5 rounded-full border border-amber-500/20">
          &gt; 3s Delayed
        </span>
      </div>

      {/* Progress Box */}
      <div className="p-4 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50/80 dark:bg-zinc-950/60 space-y-3">
        <div className="flex items-center justify-between text-xs">
          <span className="font-medium text-zinc-700 dark:text-zinc-300 flex items-center gap-1.5">
            <Download className="w-3.5 h-3.5 text-indigo-500" />
            Downloading Physics Engine
          </span>
          <span className="font-mono text-indigo-600 dark:text-indigo-400 font-bold">
            {percent}%
          </span>
        </div>

        {/* Track & Bar */}
        <div className="w-full bg-zinc-200 dark:bg-zinc-800 rounded-full h-2.5 overflow-hidden relative">
          <div
            className="bg-indigo-600 dark:bg-indigo-500 h-2.5 rounded-full transition-all duration-300 ease-out"
            style={{ width: `${percent}%` }}
          />
        </div>

        <div className="flex items-center justify-between text-[11px] text-zinc-500 dark:text-zinc-400 font-mono">
          <span>
            {loadedMb} MB / {totalMb} MB
          </span>
          <span className="text-zinc-400 dark:text-zinc-500">quadsmith-engine.wasm</span>
        </div>
      </div>

      <p className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-4 leading-normal">
        Compiling aerodynamic momentum theory and battery sag models in browser memory for zero-latency slider calculations.
      </p>
    </div>
  );
}
