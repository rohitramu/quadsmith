import { Loader2, Download, FlaskConical } from "lucide-react";
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
      className={`bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-sm relative overflow-hidden transition-all duration-300 ${className}`}
    >
      {/* Top Accent Strip */}
      <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500" />

      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-indigo-950 flex items-center justify-center text-indigo-400">
            <Loader2 className="w-4 h-4 animate-spin" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-bold text-zinc-100 leading-tight">Build Evaluator</h3>
              <span
                data-testid="wasm-evaluator-experimental-badge"
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-amber-500/15 text-amber-400 border border-amber-500/30"
              >
                <FlaskConical className="w-2.5 h-2.5 shrink-0" />
                <span>Experimental</span>
              </span>
            </div>
            <p className="text-[11px] text-zinc-400">Loading physics engine...</p>
          </div>
        </div>
        <span className="text-[10px] uppercase font-mono tracking-wider font-semibold text-amber-400 bg-amber-500/10 px-2.5 py-0.5 rounded-full border border-amber-500/20">
          &gt; 3s Delayed
        </span>
      </div>

      {/* Progress Box */}
      <div className="p-4 rounded-xl border border-zinc-800 bg-zinc-950/60 space-y-3">
        <div className="flex items-center justify-between text-xs">
          <span className="font-medium text-zinc-300 flex items-center gap-1.5">
            <Download className="w-3.5 h-3.5 text-indigo-500" />
            Downloading Physics Engine
          </span>
          <span className="font-mono text-indigo-400 font-bold">{percent}%</span>
        </div>

        {/* Track & Bar */}
        <div className="w-full bg-zinc-800 rounded-full h-2.5 overflow-hidden relative">
          <div
            className="bg-indigo-500 h-2.5 rounded-full transition-all duration-300 ease-out"
            style={{ width: `${percent}%` }}
          />
        </div>

        <div className="flex items-center justify-between text-[11px] text-zinc-400 font-mono">
          <span>
            {loadedMb} MB / {totalMb} MB
          </span>
          <span className="text-zinc-500">Physics Engine Core</span>
        </div>
      </div>

      <p className="text-[11px] text-zinc-400 mt-4 leading-normal">
        Initializing the experimental Build Evaluator in browser memory. Provided &ldquo;AS
        IS&rdquo; without warranties of airworthiness or safety.
      </p>
    </div>
  );
}
