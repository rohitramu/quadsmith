import { useState, useMemo, useRef, useEffect } from "react";
import {
  Hammer,
  RotateCcw,
  Gauge,
  Search,
  AlertTriangle,
  AlertCircle,
  ChevronRight,
  Sparkles,
  ClipboardCheck,
} from "lucide-react";
import { useDocumentMeta } from "../hooks/useDocumentMeta";
import { BatteryCombobox } from "../components/BatteryCombobox";
import { BatteryPickerModal } from "../components/BatteryPickerModal";
import { CollectionIcon } from "../components/CollectionIcon";
import { getCollectionColor } from "../lib/collectionColors";
import { getTwrDescription, getTwrColor, getHoverThrottleColor } from "../lib/format";
import {
  ForgeProvider,
  useForge,
  rxHasIntegratedAntenna,
  getVtxAntennaCount,
  getRxAntennaCount,
  isRxAntenna,
  formatFrequencyBand,
  vtxHasIntegratedAntenna,
} from "../components/forge/ForgeContext";

// Step components
import { TemplateStep } from "../components/forge/steps/TemplateStep";
import { FrameStep } from "../components/forge/steps/FrameStep";
import { FlightControllerStep } from "../components/forge/steps/FlightControllerStep";
import { ReceiverStep } from "../components/forge/steps/ReceiverStep";
import { ReceiverAntennaStep } from "../components/forge/steps/ReceiverAntennaStep";
import { EscStep } from "../components/forge/steps/EscStep";
import { MotorStep } from "../components/forge/steps/MotorStep";
import { PropellerStep } from "../components/forge/steps/PropellerStep";
import { CameraStep } from "../components/forge/steps/CameraStep";
import { VtxStep } from "../components/forge/steps/VtxStep";
import { VtxAntennaStep } from "../components/forge/steps/VtxAntennaStep";
import { GpsStep } from "../components/forge/steps/GpsStep";
import { ReviewStep } from "../components/forge/steps/ReviewStep";

// Re-export utility functions
export {
  rxHasIntegratedAntenna,
  getVtxAntennaCount,
  getRxAntennaCount,
  isRxAntenna,
  formatFrequencyBand,
  vtxHasIntegratedAntenna,
};

function BuildWizardContent() {
  useDocumentMeta({ title: "The Forge | Quadsmith" });

  const {
    activeStep,
    setActiveStep,
    unlockedSteps,
    stepCompletion,
    stepConflicts,
    stepBatteryConflicts,
    resetBuild,
    templateBuilds,
    selectedTemplateId,
    isTemplateConfirmed,

    // Selections
    selectedFrame,
    selectedFc,
    selectedRx,
    useIntegratedRx,
    integratedRx,
    selectedRxAnt,
    isRxAntennaIntegrated,
    rxAntCount,
    selectedEsc,
    useIntegratedEsc,
    integratedEsc,
    selectedMotor,
    motorCount,
    selectedProp,
    selectedCam,
    selectedVtx,
    useIntegratedVtx,
    integratedVtx,
    selectedVtxAnt,
    isVtxAntennaIntegrated,
    vtxAntCount,
    selectedGps,
    activeBattery,
    noneSelections,

    // Evaluator
    electricalLimits,
    evaluatedMetrics,
    dryWeightG,
    allUpWeightG,
    payloadWeightG,
    setPayloadWeightG,
    selectedBatteryId,
    setSelectedBatteryId,
    compatibleBatteries,
  } = useForge();

  const [isBatteryModalOpen, setIsBatteryModalOpen] = useState(false);
  const [payloadInput, setPayloadInput] = useState(payloadWeightG.toString());

  const stepRowRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const stepsContainerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const rowEl = stepRowRefs.current[activeStep];
    const containerEl = stepsContainerRef.current;
    if (rowEl && containerEl) {
      const containerRect = containerEl.getBoundingClientRect();
      const rowRect = rowEl.getBoundingClientRect();
      if (rowRect.top < containerRect.top) {
        containerEl.scrollTo({
          top: containerEl.scrollTop - (containerRect.top - rowRect.top + 8),
          behavior: "smooth",
        });
      } else if (rowRect.bottom > containerRect.bottom) {
        containerEl.scrollTo({
          top: containerEl.scrollTop + (rowRect.bottom - containerRect.bottom + 8),
          behavior: "smooth",
        });
      }
    }
  }, [activeStep]);

  const handlePayloadInput = (val: string) => {
    setPayloadInput(val);
    const parsed = parseFloat(val);
    if (!isNaN(parsed) && parsed >= 0) {
      setPayloadWeightG(parsed);
    } else if (val === "") {
      setPayloadWeightG(0);
    }
  };

  const adjustPayload = (delta: number) => {
    const next = Math.max(0, payloadWeightG + delta);
    setPayloadWeightG(next);
    setPayloadInput(next.toString());
  };

  const setPresetPayload = (val: number) => {
    setPayloadWeightG(val);
    setPayloadInput(val.toString());
  };

  // Count equipped parts out of 11 components
  const equippedCount = [
    selectedFrame,
    selectedFc,
    useIntegratedRx ? integratedRx : selectedRx,
    isRxAntennaIntegrated ? { name: "Integrated SMD" } : selectedRxAnt,
    useIntegratedEsc ? integratedEsc : selectedEsc,
    selectedMotor,
    selectedProp,
    selectedCam,
    useIntegratedVtx ? integratedVtx : selectedVtx,
    isVtxAntennaIntegrated ? { name: "Integrated VTX Ant" } : selectedVtxAnt,
    selectedGps,
  ].filter(Boolean).length;

  const selectedTemplate = templateBuilds?.find((t) => t.id === selectedTemplateId);

  const allForgeStepsList = [
    {
      step: 0,
      title: "Template",
      label: "Template",
      collectionId: null,
      item: selectedTemplate
        ? { name: selectedTemplate.name }
        : isTemplateConfirmed
          ? { name: "Start from Scratch" }
          : null,
      weightG: 0,
      omitted: false,
      count: 1,
    },
    {
      step: 1,
      title: "Frame",
      label: "Frame",
      collectionId: "frames",
      item: selectedFrame,
      weightG: selectedFrame?.weightG || 0,
      omitted: false,
      count: 1,
    },
    {
      step: 2,
      title: "Flight Controller",
      label: "Flight Controller",
      collectionId: "flight_controllers",
      item: selectedFc,
      weightG: selectedFc?.weightG || 0,
      omitted: false,
      count: 1,
    },
    {
      step: 3,
      title: "RX",
      label: "Radio Receiver",
      collectionId: "receivers",
      item: useIntegratedRx ? { name: `${integratedRx?.name} (AIO)` } : selectedRx,
      weightG: useIntegratedRx ? 0 : selectedRx?.weightG || 0,
      omitted: false,
      count: 1,
    },
    {
      step: 4,
      title: "RX Antennas",
      label: "RX Antenna",
      collectionId: "antennas",
      item: isRxAntennaIntegrated ? { name: "Integrated SMD" } : selectedRxAnt,
      weightG: isRxAntennaIntegrated ? 0 : (selectedRxAnt?.weightG || 0) * rxAntCount,
      omitted: isRxAntennaIntegrated,
      count: rxAntCount,
    },
    {
      step: 5,
      title: "ESC",
      label: "Speed Controller",
      collectionId: "electronic_speed_controllers",
      item: useIntegratedEsc ? { name: `${integratedEsc?.name} (AIO)` } : selectedEsc,
      weightG: useIntegratedEsc ? 0 : selectedEsc?.weightG || 0,
      omitted: noneSelections.esc,
      count: 1,
    },
    {
      step: 6,
      title: "Motors",
      label: "Motors",
      collectionId: "motors",
      item: selectedMotor,
      weightG: (selectedMotor?.weightG || 0) * motorCount,
      omitted: false,
      count: motorCount,
    },
    {
      step: 7,
      title: "Propellers",
      label: "Propellers",
      collectionId: "propellers",
      item: selectedProp,
      weightG: (selectedProp?.weightG || 0) * motorCount,
      omitted: false,
      count: motorCount,
    },
    {
      step: 8,
      title: "Camera",
      label: "FPV Camera",
      collectionId: "cameras",
      item: selectedCam,
      weightG: noneSelections.camera ? 0 : selectedCam?.weightG || 0,
      omitted: noneSelections.camera,
      count: 1,
    },
    {
      step: 9,
      title: "VTX",
      label: "Video Transmitter",
      collectionId: "video_transmitters",
      item: useIntegratedVtx ? { name: `${integratedVtx?.name} (AIO)` } : selectedVtx,
      weightG: noneSelections.vtx || useIntegratedVtx ? 0 : selectedVtx?.weightG || 0,
      omitted: noneSelections.vtx,
      count: 1,
    },
    {
      step: 10,
      title: "VTX Antennas",
      label: "VTX Antenna",
      collectionId: "antennas",
      item: isVtxAntennaIntegrated ? { name: "Integrated VTX Ant" } : selectedVtxAnt,
      weightG:
        noneSelections.vtx || noneSelections.vtxAntenna || isVtxAntennaIntegrated
          ? 0
          : (selectedVtxAnt?.weightG || 0) * vtxAntCount,
      omitted: noneSelections.vtx || noneSelections.vtxAntenna,
      count: vtxAntCount,
    },
    {
      step: 11,
      title: "GPS",
      label: "GPS Module",
      collectionId: "gps_receivers",
      item: selectedGps,
      weightG: noneSelections.gps ? 0 : selectedGps?.weightG || 0,
      omitted: noneSelections.gps,
      count: 1,
    },
    {
      step: 12,
      title: "Review & Save",
      label: "Review & Save",
      collectionId: null,
      item: stepCompletion[12] ? { name: "Ready to Save" } : null,
      weightG: 0,
      omitted: false,
      count: 1,
    },
  ];

  const stepProgressText: Record<number, string> = useMemo(() => {
    const texts: Record<number, string> = {};
    if (noneSelections.camera) {
      texts[9] = "Locked (Omitted)";
      texts[10] = "Locked (Omitted)";
    } else if (noneSelections.vtx) {
      texts[9] = "None (Omitted)";
      texts[10] = "Locked (Omitted)";
    }
    if (noneSelections.gps) {
      texts[11] = "None (Omitted)";
    }
    return texts;
  }, [noneSelections]);

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 py-6 space-y-6">
      {/* Header Banner */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-200 dark:border-zinc-800 pb-5">
        <div>
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-xl bg-orange-500/10 border border-orange-500/20 flex items-center justify-center">
              <Hammer className="text-[#FF6D00]" size={20} />
            </div>
            <h1 className="text-2xl font-black tracking-tight text-zinc-900 dark:text-zinc-100">
              The Forge
            </h1>
          </div>
          <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
            Component configurator with strict sequential validation and real-time physics
            simulation.
          </p>
        </div>

        <button
          type="button"
          onClick={resetBuild}
          className="flex items-center gap-1.5 self-start sm:self-center px-3 py-1.5 rounded-xl border border-red-200 dark:border-red-900/40 bg-red-50/50 dark:bg-red-950/20 text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/30 text-xs font-bold transition-colors cursor-pointer"
        >
          <RotateCcw size={13} />
          <span>Reset Build</span>
        </button>
      </div>

      {/* Main 2-Column Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Column (8 cols): Active Step Component */}
        <div className="lg:col-span-8">
          {activeStep === 0 && <TemplateStep />}
          {activeStep === 1 && <FrameStep />}
          {activeStep === 2 && <FlightControllerStep />}
          {activeStep === 3 && <ReceiverStep />}
          {activeStep === 4 && <ReceiverAntennaStep />}
          {activeStep === 5 && <EscStep />}
          {activeStep === 6 && <MotorStep />}
          {activeStep === 7 && <PropellerStep />}
          {activeStep === 8 && <CameraStep />}
          {activeStep === 9 && <VtxStep />}
          {activeStep === 10 && <VtxAntennaStep />}
          {activeStep === 11 && <GpsStep />}
          {activeStep === 12 && <ReviewStep />}
        </div>

        {/* Right Column (4 cols): Persistent Steps & Parts Summary & Live Evaluator */}
        <div className="lg:col-span-4 space-y-4 sticky top-6">
          {/* Selected Parts / Build Steps Card */}
          <div className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-4 space-y-3 shadow-sm">
            <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-800 pb-2">
              <div className="flex items-center gap-2">
                <Hammer className="text-[#FF6D00] shrink-0" size={15} />
                <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 uppercase tracking-wider">
                  Selected Parts
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold text-zinc-600 dark:text-zinc-400 font-mono">
                  Dry:{" "}
                  <span className="text-blue-600 dark:text-blue-400">{dryWeightG.toFixed(1)}g</span>
                </span>
                <span className="text-zinc-300 dark:text-zinc-700">•</span>
                <span className="text-xs font-semibold text-blue-600 dark:text-blue-400">
                  {equippedCount} / 11 Parts
                </span>
              </div>
            </div>

            <nav aria-label="Build Steps" className="w-full">
              <div
                ref={stepsContainerRef}
                className="space-y-1.5 max-h-72 overflow-y-auto pr-1 text-xs custom-scroll"
              >
                {allForgeStepsList.map((part) => {
                  const isSelected = !!part.item;
                  const isCurrent = activeStep === part.step;
                  const isUnlocked = unlockedSteps.includes(part.step);
                  const isComplete = !!stepCompletion[part.step];
                  const hasConflict = !!stepConflicts[part.step];
                  const hasBatteryConflict = !!stepBatteryConflicts[part.step];
                  const progress = stepProgressText ? stepProgressText[part.step] : undefined;
                  const colorDef = part.collectionId ? getCollectionColor(part.collectionId) : null;

                  let borderClass = "border-zinc-100 dark:border-zinc-800/80";
                  let bgClass = "hover:bg-zinc-50 dark:hover:bg-zinc-800/40 cursor-pointer";

                  if (isCurrent) {
                    borderClass =
                      "border-blue-500 bg-blue-50/70 dark:bg-blue-950/40 ring-1 ring-blue-500/40 shadow-xs";
                    bgClass = "cursor-default";
                  } else if (hasConflict) {
                    borderClass =
                      "border-amber-500 ring-1 ring-amber-500/30 bg-amber-50/50 dark:bg-amber-950/20";
                  } else if (hasBatteryConflict) {
                    borderClass =
                      "border-amber-400/80 ring-1 ring-amber-400/20 bg-amber-50/30 dark:bg-amber-950/15";
                  } else if (!isUnlocked) {
                    borderClass = "border-zinc-100 dark:border-zinc-800/40 opacity-40";
                    bgClass = "cursor-not-allowed";
                  }

                  return (
                    <button
                      key={part.step}
                      type="button"
                      ref={(el) => {
                        stepRowRefs.current[part.step] = el;
                      }}
                      role="button"
                      disabled={!isUnlocked}
                      onClick={() => {
                        if (isUnlocked) setActiveStep(part.step);
                      }}
                      aria-label={`Step ${part.step}: ${part.title}`}
                      className={`w-full p-2 rounded-xl border flex items-center justify-between gap-2 transition-all text-left ${borderClass} ${bgClass}`}
                    >
                      <div className="flex items-center gap-2 min-w-0">
                        <div className="w-5 h-5 rounded-md flex items-center justify-center shrink-0">
                          {hasConflict || hasBatteryConflict ? (
                            <AlertTriangle size={14} className="text-amber-500 shrink-0" />
                          ) : part.step === 0 ? (
                            <Sparkles
                              size={14}
                              className={
                                isComplete || isTemplateConfirmed
                                  ? "text-blue-500 shrink-0"
                                  : "text-zinc-400 dark:text-zinc-500 shrink-0"
                              }
                            />
                          ) : part.step === 12 ? (
                            <ClipboardCheck
                              size={14}
                              className={
                                isComplete
                                  ? "text-emerald-500 shrink-0"
                                  : "text-zinc-400 dark:text-zinc-500 shrink-0"
                              }
                            />
                          ) : part.collectionId && colorDef ? (
                            <CollectionIcon
                              collectionId={part.collectionId}
                              size={14}
                              color={isSelected && !part.omitted ? colorDef.hex : undefined}
                              className={
                                isSelected && !part.omitted
                                  ? "shrink-0"
                                  : "text-zinc-400 dark:text-zinc-500 shrink-0"
                              }
                            />
                          ) : null}
                        </div>
                        <div className="min-w-0">
                          <div className="font-semibold text-zinc-800 dark:text-zinc-200 text-[11px] truncate flex items-center gap-1">
                            <span className="text-zinc-400 font-normal">{part.step}.</span>
                            <span>{part.label}</span>
                          </div>
                          <div className="text-[10px] text-zinc-400 truncate">
                            {hasConflict
                              ? "Conflict"
                              : hasBatteryConflict
                                ? "Battery Conflict"
                                : progress ||
                                  (part.item
                                    ? `${part.item.name || "Selected"}${part.count > 1 ? ` (x${part.count})` : ""}`
                                    : !isUnlocked
                                      ? "Locked (Omitted)"
                                      : part.omitted
                                        ? "None (Omitted)"
                                        : part.step === 0
                                          ? "Baseline"
                                          : "Not selected")}
                          </div>
                        </div>
                      </div>

                      <div className="text-right shrink-0 flex items-center gap-1.5">
                        {isCurrent && (
                          <span className="text-[9px] px-1.5 py-0.2 rounded font-bold bg-blue-600 text-white">
                            Active
                          </span>
                        )}
                        {part.weightG > 0 ? (
                          <span className="font-mono text-zinc-600 dark:text-zinc-400 text-[10px]">
                            {part.weightG.toFixed(1)}g
                          </span>
                        ) : part.step === 0 ? (
                          <span className="text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">
                            {isComplete ? "Done" : "--"}
                          </span>
                        ) : part.step === 12 ? (
                          <span className="text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">
                            Save
                          </span>
                        ) : (
                          <span className="font-mono text-zinc-400 text-[10px]">--</span>
                        )}
                        <ChevronRight size={12} className="text-zinc-400 shrink-0" />
                      </div>
                    </button>
                  );
                })}
              </div>
            </nav>
          </div>

          {/* Live Evaluator Card */}
          <div className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-4 space-y-4 shadow-sm">
            <div className="border-b border-zinc-200 dark:border-zinc-800 pb-2.5">
              <div className="flex items-center gap-2">
                <Gauge className="text-blue-600 dark:text-blue-400 shrink-0" size={16} />
                <h3 className="text-xs font-bold uppercase tracking-wider text-zinc-800 dark:text-zinc-200">
                  Build Evaluator
                </h3>
              </div>
              <p className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-1 leading-snug">
                Calculated flight metrics are theoretical estimates based on analytical physics
                models.
              </p>
            </div>

            {/* Test Battery Selection with Browse Button */}
            <div className="space-y-1.5 text-xs">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  <label className="block font-semibold text-zinc-700 dark:text-zinc-300">
                    Test Battery (Runtime Parameter)
                  </label>
                  {!activeBattery && (
                    <span className="text-[9px] px-1.5 py-0.2 rounded font-bold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                      Required
                    </span>
                  )}
                </div>
                {electricalLimits?.connector ? (
                  <span className="text-[10px] text-blue-600 dark:text-blue-400 font-mono font-medium">
                    {electricalLimits.connector}
                  </span>
                ) : null}
              </div>

              <div className="flex items-center gap-1.5">
                <BatteryCombobox
                  selectedBatteryId={selectedBatteryId}
                  onSelectBattery={(id) => setSelectedBatteryId(id)}
                  batteries={compatibleBatteries}
                  ariaLabel="Select test battery"
                  className="flex-1 min-w-0"
                />

                <button
                  type="button"
                  onClick={() => setIsBatteryModalOpen(true)}
                  title="Browse all compatible batteries"
                  aria-label="Browse all compatible batteries"
                  className="px-2.5 py-1.5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 transition-colors text-xs font-medium flex items-center gap-1 shrink-0 cursor-pointer shadow-xs"
                >
                  <Search size={12} className="text-zinc-400" />
                  <span>Browse</span>
                </button>
              </div>

              {activeBattery ? (
                <div className="text-[10px] text-zinc-500 dark:text-zinc-400 mt-1 flex items-center gap-2">
                  <span>
                    {activeBattery.cellCountS}S ({activeBattery.minVoltage}V -{" "}
                    {activeBattery.maxVoltage}V)
                  </span>
                  <span>• {activeBattery.capacityMah}mAh</span>
                  <span>• {activeBattery.weightG}g</span>
                </div>
              ) : (
                <div className="p-2 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-800 dark:text-amber-300 text-[11px] flex items-start gap-2 mt-1">
                  <AlertCircle size={14} className="text-amber-500 mt-0.5 shrink-0" />
                  <p className="leading-tight">
                    <span className="font-bold">Evaluation Paused:</span> Select a test battery to
                    calculate thrust-to-weight, hover throttle, and flight times.
                  </p>
                </div>
              )}
            </div>

            {/* Runtime Payload Simulation */}
            <div className="bg-zinc-50 dark:bg-zinc-950/60 p-3 rounded-xl border border-zinc-200 dark:border-zinc-800 space-y-2 text-xs">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-zinc-700 dark:text-zinc-300">
                  Payload Simulation
                </span>
                <span className="font-mono font-bold text-blue-600 dark:text-blue-400">
                  +{payloadWeightG.toFixed(2)}g
                </span>
              </div>
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={() => adjustPayload(-10)}
                  disabled={payloadWeightG <= 0}
                  aria-label="Remove 10g"
                  className="w-7 h-7 flex items-center justify-center rounded-lg border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-100 dark:hover:bg-zinc-700 disabled:opacity-30 disabled:cursor-not-allowed text-zinc-700 dark:text-zinc-200 transition-colors cursor-pointer shrink-0 font-bold"
                >
                  -
                </button>
                <div className="relative flex-1 flex items-center">
                  <input
                    type="text"
                    inputMode="decimal"
                    value={payloadInput}
                    onChange={(e) => handlePayloadInput(e.target.value)}
                    placeholder="0"
                    className="w-full bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-700 rounded-lg px-2.5 py-1 pr-6 text-xs font-mono text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-1 focus:ring-blue-500 text-center"
                  />
                  <span className="absolute right-2 text-xs text-zinc-400 font-mono pointer-events-none select-none">
                    g
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => adjustPayload(10)}
                  aria-label="Add 10g"
                  className="w-7 h-7 flex items-center justify-center rounded-lg border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-100 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-200 transition-colors cursor-pointer shrink-0 font-bold"
                >
                  +
                </button>
              </div>
              <div className="flex gap-1 text-[10px]">
                <button
                  type="button"
                  onClick={() => setPresetPayload(0)}
                  className={`px-2 py-0.5 rounded cursor-pointer transition-colors font-medium ${
                    payloadWeightG === 0
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                  }`}
                >
                  Bare (0g)
                </button>
                <button
                  type="button"
                  onClick={() => setPresetPayload(16)}
                  className={`px-2 py-0.5 rounded cursor-pointer transition-colors font-medium ${
                    payloadWeightG === 16
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                  }`}
                >
                  Thumb (+16g)
                </button>
                <button
                  type="button"
                  onClick={() => setPresetPayload(133)}
                  className={`px-2 py-0.5 rounded cursor-pointer transition-colors font-medium ${
                    payloadWeightG === 133
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                  }`}
                >
                  GoPro (+133g)
                </button>
              </div>
            </div>

            {/* TWR Hero Metric */}
            <div className="text-center p-3 rounded-xl bg-zinc-50 dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800">
              <div className="text-[11px] uppercase font-semibold text-zinc-500">
                Estimated Thrust-to-Weight
              </div>
              <div
                className={`text-3xl font-extrabold mt-1 ${
                  evaluatedMetrics
                    ? getTwrColor(evaluatedMetrics.thrustToWeightRatio).text
                    : "text-zinc-400"
                }`}
              >
                {evaluatedMetrics
                  ? `${evaluatedMetrics.thrustToWeightRatio.toFixed(2)} : 1`
                  : "-- : 1"}
              </div>
              <span
                className={`inline-block mt-1 px-2.5 py-0.5 rounded-full text-[11px] font-bold border transition-colors ${
                  evaluatedMetrics
                    ? getTwrColor(evaluatedMetrics.thrustToWeightRatio).badge
                    : !activeBattery
                      ? "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20"
                      : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border-transparent"
                }`}
              >
                {evaluatedMetrics
                  ? getTwrDescription(evaluatedMetrics.thrustToWeightRatio)
                  : !activeBattery
                    ? "Select Battery to Calculate"
                    : "Select Propulsion Parts"}
              </span>
            </div>

            {/* Telemetry Metrics Grid */}
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div className="p-2.5 rounded-xl bg-zinc-50 dark:bg-zinc-950/40 border border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500">Dry Weight</span>
                <div className="text-sm font-bold text-zinc-900 dark:text-zinc-100 font-mono mt-0.5">
                  {dryWeightG.toFixed(2)}g
                </div>
              </div>
              <div className="p-2.5 rounded-xl bg-zinc-50 dark:bg-zinc-950/40 border border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500">All-Up Weight</span>
                <div className="text-sm font-bold text-zinc-900 dark:text-zinc-100 font-mono mt-0.5">
                  {allUpWeightG.toFixed(2)}g
                </div>
              </div>
              <div className="p-2.5 rounded-xl bg-zinc-50 dark:bg-zinc-950/40 border border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500">Hover Throttle</span>
                <div
                  className={`text-sm font-bold font-mono mt-0.5 ${
                    evaluatedMetrics
                      ? getHoverThrottleColor(evaluatedMetrics.hoverThrottlePercent).text
                      : "text-zinc-400"
                  }`}
                >
                  {evaluatedMetrics
                    ? `${evaluatedMetrics.hoverThrottlePercent.toFixed(1)}%`
                    : "--%"}
                </div>
              </div>
              <div className="p-2.5 rounded-xl bg-zinc-50 dark:bg-zinc-950/40 border border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500">Est. Flight Time</span>
                <div className="text-sm font-bold text-zinc-900 dark:text-zinc-100 font-mono mt-0.5">
                  {evaluatedMetrics
                    ? `${evaluatedMetrics.flightTimeMinutes.toFixed(1)} min`
                    : "-- min"}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Battery Picker Modal */}
      <BatteryPickerModal
        isOpen={isBatteryModalOpen}
        onClose={() => setIsBatteryModalOpen(false)}
        selectedBatteryId={selectedBatteryId}
        onSelectBattery={(id) => setSelectedBatteryId(id)}
        limits={electricalLimits}
        compatibleBatteries={compatibleBatteries}
      />
    </div>
  );
}

export function BuildWizardPage() {
  return (
    <ForgeProvider>
      <BuildWizardContent />
    </ForgeProvider>
  );
}
