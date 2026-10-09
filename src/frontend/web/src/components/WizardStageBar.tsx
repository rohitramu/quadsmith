import { Check } from "lucide-react";

export interface WizardStageBarProps {
  currentStage: number;
  unlockedStages: number[];
  stageCompletion: {
    1: boolean;
    2: boolean;
    3: boolean;
    4: boolean;
  };
  stageProgressText: {
    1: string;
    2: string;
    3: string;
    4: string;
  };
  onSelectStage: (stage: number) => void;
}

const STAGES = [
  {
    stage: 1,
    title: "Airframe & Propulsion",
    subtitle: "Frame, Motors, Props (Required)",
  },
  {
    stage: 2,
    title: "Flight Electronics",
    subtitle: "FC, ESC, RX, Antenna & GPS",
  },
  {
    stage: 3,
    title: "Video",
    subtitle: "VTX, Camera & Antenna",
  },
  {
    stage: 4,
    title: "Review & Save",
    subtitle: "BOM & Database Save",
  },
];

export function WizardStageBar({
  currentStage,
  unlockedStages,
  stageCompletion,
  stageProgressText,
  onSelectStage,
}: WizardStageBarProps) {
  return (
    <nav aria-label="Build Stages" className="w-full">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {STAGES.map((s) => {
          const isActive = currentStage === s.stage;
          const isUnlocked = unlockedStages.includes(s.stage);
          const isComplete = stageCompletion[s.stage as 1 | 2 | 3 | 4];
          const progress = stageProgressText[s.stage as 1 | 2 | 3 | 4];

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
              key={s.stage}
              type="button"
              disabled={!isUnlocked}
              onClick={() => {
                if (isUnlocked) onSelectStage(s.stage);
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
                  Stage {s.stage}
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
