import { useMemo } from "react";
import { Link } from "react-router-dom";
import {
  ArrowLeft,
  Save,
  AlertCircle,
  Loader2,
  ShieldCheck,
  ShieldAlert,
  Lock,
  Globe,
} from "lucide-react";
import { useForge } from "../ForgeContext";
import { CollectionIcon } from "@quadsmith/shared";
import { getCollectionColor } from "@quadsmith/shared";

export function ReviewStep() {
  const {
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
    vtxAntCount,
    selectedGps,
    activeBattery,
    noneSelections,
    dryWeightG,
    buildName,
    setBuildName,
    buildDesc,
    setBuildDesc,
    isPrivate,
    setIsPrivate,
    isSaving,
    saveError,
    editingBuildId,
    handleSaveBuild,
    handlePrevStep,
  } = useForge();

  const activeRx = useIntegratedRx ? integratedRx : selectedRx;
  const activeEsc = useIntegratedEsc ? integratedEsc : selectedEsc;
  const activeVtx = useIntegratedVtx ? integratedVtx : selectedVtx;

  const bomItems = useMemo(() => {
    return [
      {
        id: "frame",
        collectionId: "frames",
        label: "Airframe",
        name: selectedFrame?.name || "None",
        manufacturer: selectedFrame?.manufacturer,
        qty: 1,
        weightG: selectedFrame?.weightG || 0,
        omitted: false,
      },
      {
        id: "fc",
        collectionId: "flight_controllers",
        label: "Flight Controller",
        name: selectedFc?.name || "None",
        manufacturer: selectedFc?.manufacturer,
        qty: 1,
        weightG: selectedFc?.weightG || 0,
        omitted: false,
      },
      {
        id: "rx",
        collectionId: "receivers",
        label: "Radio Receiver",
        name: isIntegrated(useIntegratedRx)
          ? `${integratedRx?.name} (AIO Integrated)`
          : activeRx?.name || "None",
        manufacturer: activeRx?.manufacturer,
        qty: 1,
        weightG: useIntegratedRx ? 0 : activeRx?.weightG || 0,
        omitted: false,
      },
      {
        id: "rx_antenna",
        collectionId: "antennas",
        label: "RX Antenna",
        name: isRxAntennaIntegrated
          ? "Integrated Ceramic Antenna"
          : selectedRxAnt
            ? `${selectedRxAnt.name}`
            : "None",
        manufacturer: isRxAntennaIntegrated ? undefined : selectedRxAnt?.manufacturer,
        qty: isRxAntennaIntegrated ? 0 : rxAntCount,
        weightG: isRxAntennaIntegrated ? 0 : (selectedRxAnt?.weightG || 0) * rxAntCount,
        omitted: isRxAntennaIntegrated,
      },
      {
        id: "esc",
        collectionId: "electronic_speed_controllers",
        label: "Speed Controller",
        name: useIntegratedEsc
          ? `${integratedEsc?.name} (AIO Integrated)`
          : activeEsc?.name || "None",
        manufacturer: activeEsc?.manufacturer,
        qty: 1,
        weightG: useIntegratedEsc ? 0 : activeEsc?.weightG || 0,
        omitted: false,
      },
      {
        id: "motor",
        collectionId: "motors",
        label: "Motors",
        name: selectedMotor?.name || "None",
        manufacturer: selectedMotor?.manufacturer,
        qty: motorCount,
        weightG: (selectedMotor?.weightG || 0) * motorCount,
        omitted: false,
      },
      {
        id: "propeller",
        collectionId: "propellers",
        label: "Propellers",
        name: selectedProp?.name || "None",
        manufacturer: selectedProp?.manufacturer,
        qty: motorCount,
        weightG: (selectedProp?.weightG || 0) * motorCount,
        omitted: false,
      },
      {
        id: "camera",
        collectionId: "cameras",
        label: "FPV Camera",
        name: noneSelections.camera ? "None (Omitted)" : selectedCam?.name || "None",
        manufacturer: selectedCam?.manufacturer,
        qty: noneSelections.camera ? 0 : 1,
        weightG: noneSelections.camera ? 0 : selectedCam?.weightG || 0,
        omitted: noneSelections.camera,
      },
      {
        id: "vtx",
        collectionId: "video_transmitters",
        label: "Video Transmitter",
        name: noneSelections.vtx
          ? "None (Omitted)"
          : useIntegratedVtx
            ? `${integratedVtx?.name} (AIO Integrated)`
            : activeVtx?.name || "None",
        manufacturer: activeVtx?.manufacturer,
        qty: noneSelections.vtx ? 0 : 1,
        weightG: noneSelections.vtx || useIntegratedVtx ? 0 : activeVtx?.weightG || 0,
        omitted: noneSelections.vtx,
      },
      {
        id: "vtx_antenna",
        collectionId: "antennas",
        label: "VTX Antenna",
        name:
          noneSelections.vtx || noneSelections.vtxAntenna
            ? "None (Omitted)"
            : selectedVtxAnt
              ? `${selectedVtxAnt.name}`
              : "None",
        manufacturer: selectedVtxAnt?.manufacturer,
        qty: noneSelections.vtx || noneSelections.vtxAntenna ? 0 : vtxAntCount,
        weightG:
          noneSelections.vtx || noneSelections.vtxAntenna
            ? 0
            : (selectedVtxAnt?.weightG || 0) * vtxAntCount,
        omitted: noneSelections.vtx || noneSelections.vtxAntenna,
      },
      {
        id: "gps",
        collectionId: "gps_receivers",
        label: "GPS Module",
        name: noneSelections.gps ? "None (Omitted)" : selectedGps?.name || "None",
        manufacturer: selectedGps?.manufacturer,
        qty: noneSelections.gps ? 0 : 1,
        weightG: noneSelections.gps ? 0 : selectedGps?.weightG || 0,
        omitted: noneSelections.gps,
      },
      {
        id: "battery",
        collectionId: "batteries",
        label: "Test Battery",
        name: activeBattery ? activeBattery.name : "No compatible batteries",
        manufacturer: activeBattery?.manufacturer,
        qty: activeBattery ? 1 : 0,
        weightG: activeBattery?.weightG || 0,
        omitted: !activeBattery,
      },
    ];
  }, [
    selectedFrame,
    selectedFc,
    activeRx,
    useIntegratedRx,
    integratedRx,
    selectedRxAnt,
    isRxAntennaIntegrated,
    rxAntCount,
    activeEsc,
    useIntegratedEsc,
    integratedEsc,
    selectedMotor,
    motorCount,
    selectedProp,
    selectedCam,
    activeVtx,
    useIntegratedVtx,
    integratedVtx,
    selectedVtxAnt,
    vtxAntCount,
    selectedGps,
    activeBattery,
    noneSelections,
  ]);

  function isIntegrated(val: boolean) {
    return val;
  }

  return (
    <div className="space-y-4">
      {/* Save Error Banner */}
      {saveError && (
        <div className="p-3.5 rounded-xl border border-red-500/30 bg-red-500/10 text-red-300 flex items-start gap-3 text-xs">
          <AlertCircle size={16} className="text-red-500 mt-0.5 shrink-0" />
          <div>
            <span className="font-bold">Error Saving Build</span>
            <p className="opacity-90 mt-0.5">{saveError}</p>
          </div>
        </div>
      )}

      {/* Bill of Materials Card */}
      <div className="p-4 rounded-2xl border border-zinc-800 bg-zinc-900 space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <ShieldCheck size={18} className="text-emerald-500" />
            <h3 className="font-bold text-sm text-zinc-100">Bill of Materials (BOM)</h3>
          </div>
          <div className="text-xs font-mono font-bold text-zinc-400">
            Dry Weight: <span className="text-blue-400">{dryWeightG.toFixed(1)}g</span>
          </div>
        </div>

        <div className="divide-y divide-zinc-800 overflow-hidden rounded-xl border border-zinc-800 text-xs">
          {bomItems.map((item) => {
            const isSelected =
              !item.omitted &&
              item.name !== "None" &&
              item.qty > 0 &&
              !item.name.startsWith("None");
            const colorDef = getCollectionColor(item.collectionId);

            return (
              <div
                key={item.id}
                className={`p-2.5 flex items-center justify-between gap-2 transition-colors ${
                  !isSelected ? "opacity-60 bg-zinc-950/40" : "bg-zinc-900"
                }`}
              >
                <div className="flex items-center gap-2.5 min-w-0">
                  <CollectionIcon
                    collectionId={item.collectionId}
                    size={14}
                    color={isSelected ? colorDef.hex : undefined}
                    className={isSelected ? "" : "text-zinc-500"}
                  />
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <span className="font-bold text-zinc-100 truncate">{item.name}</span>
                      {item.qty > 1 && (
                        <span className="text-[10px] px-1.5 py-0.2 rounded font-mono font-bold bg-zinc-800 text-zinc-300">
                          x{item.qty}
                        </span>
                      )}
                    </div>
                    <span className="text-[10px] text-zinc-500">
                      {item.label} {item.manufacturer ? `• by ${item.manufacturer}` : ""}
                    </span>
                  </div>
                </div>

                <div className="text-right shrink-0">
                  <span className="font-mono text-zinc-300 font-semibold text-xs">
                    {item.weightG > 0 ? `${item.weightG.toFixed(1)}g` : "--"}
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Build Details Form */}
      <div className="p-4 rounded-2xl border border-zinc-800 bg-zinc-900 space-y-3">
        <h3 className="font-bold text-sm text-zinc-100">Build Metadata &amp; Visibility</h3>

        <div className="space-y-3 text-xs">
          <div>
            <label htmlFor="build-name" className="block font-semibold text-zinc-300 mb-1">
              Build Name
            </label>
            <input
              id="build-name"
              type="text"
              value={buildName}
              onChange={(e) => setBuildName(e.target.value)}
              placeholder="e.g., Apex 5 Light Freestyle"
              className="w-full bg-zinc-950 border border-zinc-800 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-blue-500 font-medium"
            />
          </div>

          <div>
            <label className="block font-semibold text-zinc-300 mb-1">Description</label>
            <textarea
              rows={2}
              value={buildDesc}
              onChange={(e) => setBuildDesc(e.target.value)}
              placeholder="Brief description of build purpose, PID tune, or custom 3D prints..."
              className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-3 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <div className="flex items-center justify-between p-3 rounded-xl bg-zinc-950 border border-zinc-800">
            <div className="flex items-center gap-2.5">
              {isPrivate ? (
                <Lock size={15} className="text-zinc-500" />
              ) : (
                <Globe size={15} className="text-blue-500" />
              )}
              <div>
                <span className="font-bold text-zinc-200">
                  {isPrivate ? "Private Build" : "Public Build"}
                </span>
                <p className="text-[11px] text-zinc-400">
                  {isPrivate
                    ? "Only you can view and edit this build configuration."
                    : "Visible in the community showcase and public search."}
                </p>
              </div>
            </div>

            <button
              type="button"
              role="switch"
              aria-checked={!isPrivate}
              aria-label="Build Visibility"
              onClick={() => setIsPrivate(!isPrivate)}
              className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-600 focus:ring-offset-2 ${
                !isPrivate ? "bg-blue-600" : "bg-zinc-700"
              }`}
            >
              <span className="sr-only">Build Visibility</span>
              <span
                aria-hidden="true"
                className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out ${
                  !isPrivate ? "translate-x-5" : "translate-x-0"
                }`}
              />
            </button>
          </div>
        </div>
      </div>

      {/* Hardware & Flight Safety Disclaimer */}
      <div
        data-testid="review-safety-disclaimer"
        className="p-3.5 rounded-2xl border border-amber-500/20 bg-amber-500/5 flex items-start gap-3 text-xs text-zinc-300"
      >
        <ShieldAlert size={16} className="text-amber-400 shrink-0 mt-0.5" />
        <p className="leading-relaxed">
          <strong className="text-amber-300">Hardware &amp; Flight Safety Notice:</strong>{" "}
          Compatibility checks and{" "}
          <span className="text-amber-400 font-semibold">Experimental Build Evaluator</span> metrics
          are theoretical estimates. Before assembly or flight, always verify manufacturer
          datasheets, wiring pinouts, electrical ratings, and propeller clearance, and test first
          power-up with a current-limiting smoke stopper. Review our{" "}
          <Link
            to="/terms#disclaimers"
            className="text-amber-400 hover:text-amber-300 underline underline-offset-2 font-medium"
          >
            Safety Disclaimers
          </Link>{" "}
          and{" "}
          <Link
            to="/terms"
            className="text-amber-400 hover:text-amber-300 underline underline-offset-2 font-medium"
          >
            Terms of Service
          </Link>
          .
        </p>
      </div>

      {/* Navigation Footer */}
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={handlePrevStep}
          className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold text-zinc-400 hover:bg-zinc-800 transition-colors"
        >
          <ArrowLeft size={14} />
          <span>Back: GPS</span>
        </button>

        <button
          type="button"
          disabled={isSaving || !selectedFrame || !selectedFc || !selectedMotor || !selectedProp}
          onClick={handleSaveBuild}
          className="flex items-center gap-2 px-6 py-2.5 rounded-xl font-bold text-xs bg-emerald-600 hover:bg-emerald-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-md cursor-pointer"
        >
          {isSaving ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
          <span>
            {isSaving ? "Saving Build..." : editingBuildId ? "Save Changes" : "Save & Create Build"}
          </span>
        </button>
      </div>
    </div>
  );
}
