import { Check, AlertTriangle, Lock } from "lucide-react";

export interface WizardStepBarProps {
  currentStep: number;
  unlockedSteps: number[];
  stepCompletion: Record<number, boolean>;
  stepProgressText?: Record<number, string>;
  stepConflicts?: Record<number, string>;
  stepBatteryConflicts?: Record<number, boolean>;
  onSelectStep: (step: number) => void;
}

export const FORGE_STEPS = [
  { step: 0, title: "Template", subtitle: "Baseline" },
  { step: 1, title: "Frame", subtitle: "Airframe" },
  { step: 2, title: "Flight Controller", subtitle: "Flight Logic" },
  { step: 3, title: "RX", subtitle: "Radio Link" },
  { step: 4, title: "RX Antennas", subtitle: "RC Signal" },
  { step: 5, title: "ESC", subtitle: "Speed Control" },
  { step: 6, title: "Motors", subtitle: "Propulsion" },
  { step: 7, title: "Propellers", subtitle: "Thrust" },
  { step: 8, title: "Camera", subtitle: "FPV Video" },
  { step: 9, title: "VTX", subtitle: "Transmitter" },
  { step: 10, title: "VTX Antennas", subtitle: "Video Signal" },
  { step: 11, title: "GPS", subtitle: "Navigation" },
  { step: 12, title: "Review & Save", subtitle: "Finalize" },
];

export function WizardStepBar({
  currentStep,
  unlockedSteps,
  stepCompletion,
  stepProgressText,
  stepConflicts = {},
  stepBatteryConflicts = {},
  onSelectStep,
}: WizardStepBarProps) {
  return (
    <nav aria-label="Build Steps" className="w-full">
      <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar py-1 px-0.5">
        {FORGE_STEPS.map((s) => {
          const isActive = currentStep === s.step;
          const isUnlocked = unlockedSteps.includes(s.step);
          const isComplete = !!stepCompletion[s.step];
          const hasConflict = Boolean(stepConflicts[s.step]);
          const hasBatteryConflict = Boolean(stepBatteryConflicts[s.step]);
          const progress = stepProgressText ? stepProgressText[s.step] : undefined;

          let borderClass = "border-zinc-200 dark:border-zinc-800";
          let bgClass = "bg-white dark:bg-zinc-900";
          let textClass = "text-zinc-600 dark:text-zinc-400";
          let cursorClass = "cursor-pointer hover:border-zinc-300 dark:hover:border-zinc-700";

          if (isActive) {
            borderClass = "border-blue-500 ring-2 ring-blue-500/20";
            bgClass = "bg-blue-50/60 dark:bg-blue-950/30";
            textClass = "text-blue-900 dark:text-blue-100";
            cursorClass = "cursor-default";
          } else if (hasConflict) {
            borderClass = "border-amber-500 ring-1 ring-amber-500/30";
            bgClass = "bg-amber-50/50 dark:bg-amber-950/20";
            textClass = "text-amber-900 dark:text-amber-200";
          } else if (hasBatteryConflict) {
            borderClass = "border-amber-400/80 ring-1 ring-amber-400/20";
            bgClass = "bg-amber-50/30 dark:bg-amber-950/15";
            textClass = "text-amber-800 dark:text-amber-300";
          } else if (!isUnlocked) {
            borderClass = "border-zinc-200/50 dark:border-zinc-800/50 opacity-60";
            bgClass = "bg-zinc-100/50 dark:bg-zinc-900/40";
            textClass = "text-zinc-400 dark:text-zinc-500";
            cursorClass = "cursor-not-allowed";
          } else if (isComplete) {
            borderClass = "border-emerald-500/40 hover:border-emerald-500";
            bgClass = "bg-white dark:bg-zinc-900";
            textClass = "text-zinc-800 dark:text-zinc-200";
          }

          return (
            <button
              key={s.step}
              type="button"
              disabled={!isUnlocked}
              onClick={() => {
                if (isUnlocked) onSelectStep(s.step);
              }}
              aria-label={`Step ${s.step}: ${s.title}`}
              className={`flex items-center gap-2 px-3 py-2 rounded-xl border text-xs font-medium whitespace-nowrap transition-all shrink-0 ${borderClass} ${bgClass} ${cursorClass}`}
            >
              <div className="flex items-center justify-center shrink-0">
                {hasConflict ? (
                  <AlertTriangle size={13} className="text-amber-500" />
                ) : hasBatteryConflict ? (
                  <AlertTriangle size={13} className="text-amber-500" />
                ) : !isUnlocked ? (
                  <Lock size={12} className="text-zinc-400 dark:text-zinc-500" />
                ) : isComplete ? (
                  <Check
                    size={13}
                    strokeWidth={2.5}
                    className="text-emerald-600 dark:text-emerald-400"
                  />
                ) : (
                  <span
                    className={`w-4 h-4 rounded-full flex items-center justify-center text-[10px] font-bold ${
                      isActive
                        ? "bg-blue-600 text-white"
                        : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400"
                    }`}
                  >
                    {s.step}
                  </span>
                )}
              </div>

              <div className="flex flex-col text-left">
                <span
                  className={`text-xs font-bold leading-tight ${isActive ? "text-blue-600 dark:text-blue-400" : textClass}`}
                >
                  {s.title}
                </span>
                <span className="text-[10px] text-zinc-400 dark:text-zinc-500 leading-tight">
                  {hasConflict
                    ? "Conflict"
                    : hasBatteryConflict
                      ? "Battery mismatch"
                      : !isUnlocked
                        ? progress || "Locked"
                        : progress || (isComplete ? "Done" : s.subtitle)}
                </span>
              </div>
            </button>
          );
        })}
      </div>
    </nav>
  );
}
