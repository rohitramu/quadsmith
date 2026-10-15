import { useState, useMemo } from "react";
import { Search, X, ArrowRight, Sparkles } from "lucide-react";
import { useForge } from "../ForgeContext";

export function TemplateStep() {
  const {
    templateBuilds,
    selectedTemplateId,
    isTemplateConfirmed,
    applyTemplate,
    selectNoTemplate,
    handleNextStep,
    frames,
    motors,
    fcs,
  } = useForge();

  const [search, setSearch] = useState("");

  const filteredTemplates = useMemo(() => {
    if (!search.trim()) return templateBuilds;
    const q = search.toLowerCase();
    return templateBuilds.filter(
      (b) => b.name?.toLowerCase().includes(q) || b.description?.toLowerCase().includes(q),
    );
  }, [templateBuilds, search]);

  const isScratchSelected = isTemplateConfirmed && !selectedTemplateId;

  return (
    <div className="space-y-4">
      <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              Choose Starting Baseline
            </h3>
          </div>
          <span className="text-xs font-semibold text-blue-600 dark:text-blue-400">
            {selectedTemplateId
              ? `Template: ${templateBuilds.find((b) => (b.id || b.uuid) === selectedTemplateId)?.name || "Selected"}`
              : isScratchSelected
                ? "Starting from Scratch"
                : "Selection Required"}
          </span>
        </div>

        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          Select whether you want to build from a blank slate or begin from a proven existing
          quadcopter build. Confirming your baseline unlocks Step 1.
        </p>

        {/* Search Input */}
        <div className="relative">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Filter templates by build name, frame, style, or specs..."
            className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
          {search && (
            <button
              type="button"
              onClick={() => setSearch("")}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
            >
              <X size={13} />
            </button>
          )}
        </div>

        {/* Scrollable List with "No Template" at the top */}
        <div className="overflow-y-auto max-h-80 space-y-2.5 pr-1.5 focus:outline-none">
          {/* Dedicated "No Template / Start from Scratch" Card */}
          <div
            role="button"
            tabIndex={0}
            onClick={selectNoTemplate}
            className={`p-3.5 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3 ${
              isScratchSelected
                ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
            }`}
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <Sparkles size={14} className="text-blue-500" />
                <span className="text-xs sm:text-sm font-bold text-zinc-900 dark:text-zinc-100">
                  No Template (Start from Scratch)
                </span>
                <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                  Blank Slate
                </span>
              </div>
              <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-1">
                Hand-pick every component from Frame to GPS without any template pre-fill.
              </div>
            </div>
            <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
              <span
                className={`text-[10px] uppercase font-bold tracking-wide px-2.5 py-1 rounded ${
                  isScratchSelected
                    ? "bg-blue-600 text-white"
                    : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400"
                }`}
              >
                {isScratchSelected ? "Confirmed" : "Start Scratch"}
              </span>
              <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center">
                {isScratchSelected && <div className="w-2 h-2 rounded-full bg-blue-500" />}
              </div>
            </div>
          </div>

          {/* Existing Builds as Templates */}
          {filteredTemplates.map((template) => {
            const isSelected = selectedTemplateId === (template.id || template.uuid);
            const frameObj = frames.find((f) => f.uuid === template.frameUuid);
            const motorObj = motors.find((m) => m.uuid === template.motorUuid);
            const fcObj = fcs.find((f) => f.uuid === template.flightControllerUuid);
            const summaryParts = [frameObj?.name, motorObj?.name, fcObj?.name].filter(Boolean);

            return (
              <div
                key={template.id || template.uuid}
                role="button"
                tabIndex={0}
                onClick={() => applyTemplate(template)}
                className={`p-3.5 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3 ${
                  isSelected
                    ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                    : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                }`}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-xs sm:text-sm font-bold text-zinc-900 dark:text-zinc-100 truncate">
                      {template.name}
                    </span>
                    {frameObj?.geometry && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 shrink-0">
                        {frameObj.geometry}
                      </span>
                    )}
                  </div>
                  {summaryParts.length > 0 && (
                    <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-1 truncate">
                      {summaryParts.join(" • ")}
                    </div>
                  )}
                  {template.description && (
                    <div className="text-[10px] text-zinc-400 dark:text-zinc-500 mt-0.5 line-clamp-1">
                      {template.description}
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                  <span
                    className={`text-[10px] uppercase font-bold tracking-wide px-2.5 py-1 rounded ${
                      isSelected
                        ? "bg-blue-600 text-white"
                        : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400"
                    }`}
                  >
                    {isSelected ? "Template Active" : "Use Template"}
                  </span>
                  <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center">
                    {isSelected && <div className="w-2 h-2 rounded-full bg-blue-500" />}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Footer Navigation */}
      <div className="flex items-center justify-end">
        <button
          type="button"
          disabled={!isTemplateConfirmed}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>Next: Frame</span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
