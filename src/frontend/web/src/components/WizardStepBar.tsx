import { Check } from "lucide-react";

export interface WizardStepBarProps {
  currentStep: number;
  unlockedSteps: number[];
  stepCompletion: {
    0: boolean;
    1: boolean;
    2: boolean;
    3: boolean;
    4: boolean;
  };
  stepProgressText: {
    0: string;
    1: string;
    2: string;
    3: string;
    4: string;
  };
  onSelectStep: (step: number) => void;
}

const STEPS = [
  {
    step: 0,
    title: "Template Selection",
    subtitle: "Scratch or Template",
  },
  {
    step: 1,
    title: "Airframe & Propulsion",
    subtitle: "Frame, Motors, Props",
  },
  {
    step: 2,
    title: "Flight Electronics",
    subtitle: "FC, ESC, RX, GPS",
  },
  {
    step: 3,
    title: "Video",
    subtitle: "VTX, Camera & Antenna",
  },
  {
    step: 4,
    title: "Review & Save",
    subtitle: "BOM & Database Save",
  },
];

export function WizardStepBar({
  currentStep,
  unlockedSteps,
  stepCompletion,
  stepProgressText,
  onSelectStep,
}: WizardStepBarProps) {
  return (
    <nav aria-label="Build Steps" className="w-full">
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-2.5">
        {STEPS.map((s) => {
          const isActive = currentStep === s.step;
          const isUnlocked = unlockedSteps.includes(s.step);
          const isComplete = stepCompletion[s.step as 0 | 1 | 2 | 3 | 4];
          const progress = stepProgressText[s.step as 0 | 1 | 2 | 3 | 4];

          let borderClass = "border-zinc-200 dark:border-zinc-800";
          let bgClass = "bg-white dark:bg-zinc-900";
          let textClass = "text-zinc-600 dark:text-zinc-400";
          let cursorClass = "cursor-pointer hover:border-zinc-300 dark:hover:border-zinc-700";

          if (isActive) {
            borderClass = "border-blue-500 ring-2 ring-blue-500/20";
            bgClass = "bg-blue-50/50 dark:bg-blue-950/20";
            textClass = "text-zinc-900 dark:text-zinc-50";
            cursorClass = "cursor-default";
          } else if (!isUnlocked) {
            borderClass = "border-zinc-200/50 dark:border-zinc-800/50 opacity-60";
            bgClass = "bg-zinc-100/50 dark:bg-zinc-900/40";
            textClass = "text-zinc-400 dark:text-zinc-500";
            cursorClass = "cursor-not-allowed";
          } else if (isComplete) {
            borderClass = "border-emerald-500/40 hover:border-emerald-500";
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
              className={`text-left p-3.5 rounded-xl border transition-all relative ${borderClass} ${bgClass} ${cursorClass}`}
            >
              <div className="flex items-center justify-between mb-1">
                <span
                  className={`text-[11px] font-bold uppercase tracking-wider ${
                    isActive
                      ? "text-blue-600 dark:text-blue-400"
                      : isComplete
                        ? "text-emerald-600 dark:text-emerald-400"
                        : "text-zinc-400 dark:text-zinc-500"
                  }`}
                >
                  Step {s.step}
                </span>

                {isComplete ? (
                  <span className="flex items-center gap-1 text-[11px] font-bold text-emerald-600 dark:text-emerald-400">
                    <Check size={12} strokeWidth={3} />
                    <span>Done</span>
                  </span>
                ) : !isUnlocked ? null : (
                  <span
                    className={`text-[11px] font-bold ${
                      isActive ? "text-amber-500 dark:text-amber-400" : "text-zinc-400"
                    }`}
                  >
                    {progress}
                  </span>
                )}
              </div>

              <div
                className={`text-sm font-bold truncate ${
                  isActive ? "text-zinc-900 dark:text-white" : textClass
                }`}
              >
                {s.title}
              </div>
              <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 truncate">
                {s.subtitle}
              </div>
            </button>
          );
        })}
      </div>
    </nav>
  );
}
