import { AlertTriangle, Sparkles, ClipboardCheck } from "lucide-react";
import { CollectionIcon } from "./CollectionIcon";
import { getCollectionColor } from "../lib/collectionColors";

export interface WizardStepBarProps {
  currentStep: number;
  unlockedSteps: number[];
  stepCompletion: Record<number, boolean>;
  stepProgressText?: Record<number, string>;
  stepConflicts?: Record<number, string>;
  stepBatteryConflicts?: Record<number, boolean>;
  onSelectStep: (step: number) => void;
}

export interface ForgeStepDef {
  step: number;
  title: string;
  subtitle: string;
  collectionId: string | null;
}

export const FORGE_STEPS: ForgeStepDef[] = [
  { step: 0, title: "Template", subtitle: "Baseline", collectionId: null },
  { step: 1, title: "Frame", subtitle: "Airframe", collectionId: "frames" },
  {
    step: 2,
    title: "Flight Controller",
    subtitle: "Flight Logic",
    collectionId: "flight_controllers",
  },
  { step: 3, title: "RX", subtitle: "Radio Link", collectionId: "receivers" },
  { step: 4, title: "RX Antennas", subtitle: "RC Signal", collectionId: "antennas" },
  {
    step: 5,
    title: "ESC",
    subtitle: "Speed Control",
    collectionId: "electronic_speed_controllers",
  },
  { step: 6, title: "Motors", subtitle: "Propulsion", collectionId: "motors" },
  { step: 7, title: "Propellers", subtitle: "Thrust", collectionId: "propellers" },
  { step: 8, title: "Camera", subtitle: "FPV Video", collectionId: "cameras" },
  { step: 9, title: "VTX", subtitle: "Transmitter", collectionId: "video_transmitters" },
  { step: 10, title: "VTX Antennas", subtitle: "Video Signal", collectionId: "antennas" },
  { step: 11, title: "GPS", subtitle: "Navigation", collectionId: "gps_receivers" },
  { step: 12, title: "Review & Save", subtitle: "Finalize", collectionId: null },
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
                  <AlertTriangle size={14} className="text-amber-500 shrink-0" />
                ) : hasBatteryConflict ? (
                  <AlertTriangle size={14} className="text-amber-500 shrink-0" />
                ) : s.step === 0 ? (
                  <Sparkles
                    size={14}
                    className={
                      isComplete
                        ? "text-blue-500 shrink-0"
                        : "text-zinc-400 dark:text-zinc-500 shrink-0"
                    }
                  />
                ) : s.step === 12 ? (
                  <ClipboardCheck
                    size={14}
                    className={
                      isComplete
                        ? "text-emerald-500 shrink-0"
                        : "text-zinc-400 dark:text-zinc-500 shrink-0"
                    }
                  />
                ) : s.collectionId ? (
                  <CollectionIcon
                    collectionId={s.collectionId}
                    size={15}
                    color={isComplete ? getCollectionColor(s.collectionId).hex : undefined}
                    className={
                      isComplete ? "shrink-0" : "text-zinc-400 dark:text-zinc-500 shrink-0"
                    }
                  />
                ) : null}
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
