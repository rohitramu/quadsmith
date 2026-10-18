import React, { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { ShieldAlert, AlertTriangle, Zap, Wind, Radio, Layers, X, Scale } from "lucide-react";

export interface SafetyDisclaimersModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function SafetyDisclaimersCards() {
  return (
    <div className="space-y-4">
      {/* Critical Warning Callout */}
      <div
        role="note"
        aria-label="Critical Safety Warning"
        className="p-4 rounded-xl border border-amber-500/30 bg-amber-950/25 text-amber-200 flex items-start gap-3.5"
      >
        <AlertTriangle className="text-amber-400 shrink-0 mt-0.5" size={20} />
        <div className="space-y-1 text-xs sm:text-sm leading-relaxed text-pretty">
          <p className="font-bold text-amber-100">
            Important Hardware, Electrical, and Flight Safety Notice
          </p>
          <p className="text-amber-200/90">
            The Experimental Build Evaluator, compatibility checks, and component catalog are
            provided strictly <strong>&ldquo;AS IS&rdquo;</strong> without warranties of
            airworthiness or safety. Improper wiring, voltage mismatches, or mechanical failures can
            cause fires, severe personal injury, or property damage. Always verify every component
            against official manufacturer datasheets, test on the bench before applying full battery
            power, and comply with all local rules and regulations.
          </p>
        </div>
      </div>

      {/* 4 Detailed Disclaimer Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="p-4 sm:p-5 rounded-xl border border-zinc-800 bg-zinc-950/70 space-y-2">
          <div className="flex items-center gap-2 text-sm font-bold text-zinc-100">
            <Zap size={16} className="text-amber-400 shrink-0" />
            <h3>Electrical &amp; LiPo Battery Safety</h3>
          </div>
          <p className="text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
            Lithium Polymer (LiPo/LiHV) and Lithium-Ion batteries carry severe risks of fire,
            explosion, and thermal runaway if short-circuited, overcharged, punctured, or operated
            beyond their current ratings. Never rely solely on software compatibility checks for
            wiring. Always inspect board silkscreens, verify pinouts and polarity with a digital
            multimeter, and use a current-limiting smoke stopper when powering a build for the first
            time.
          </p>
        </div>

        <div className="p-4 sm:p-5 rounded-xl border border-zinc-800 bg-zinc-950/70 space-y-2">
          <div className="flex items-center gap-2 text-sm font-bold text-zinc-100">
            <Wind size={16} className="text-rose-400 shrink-0" />
            <h3>Mechanical &amp; Flight Safety</h3>
          </div>
          <p className="text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
            High-RPM brushless motors and propellers can inflict severe lacerations and eye
            injuries. Always remove all propellers before connecting a battery on the bench for
            flight controller configuration, ESC flashing, or motor direction testing. Verify that
            motor mounting screws do not contact or short stator windings, confirm frame clearances,
            and validate receiver failsafe behavior prior to every flight.
          </p>
        </div>

        <div className="p-4 sm:p-5 rounded-xl border border-zinc-800 bg-zinc-950/70 space-y-2">
          <div className="flex items-center gap-2 text-sm font-bold text-zinc-100">
            <Radio size={16} className="text-blue-400 shrink-0" />
            <h3>Local Rules &amp; Regulations</h3>
          </div>
          <p className="text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
            Operating multirotor aircraft and radio or video transmitters is subject to local laws,
            airspace restrictions, and radio frequency regulations. Catalog listings of transmit
            power levels or frequency bands do not authorize their use in your area. You are solely
            responsible for understanding and complying with all applicable local rules and
            regulations before powering on transmitters or flying.
          </p>
        </div>

        <div className="p-4 sm:p-5 rounded-xl border border-zinc-800 bg-zinc-950/70 space-y-2">
          <div className="flex items-center gap-2 text-sm font-bold text-zinc-100">
            <Layers size={16} className="text-emerald-400 shrink-0" />
            <h3>Catalog Data &amp; Hardware Revisions</h3>
          </div>
          <p className="text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
            Component specifications, dimensions, weights, and compatibility rules are compiled from
            manufacturer documentation and community contributions on a best-effort basis.
            Manufacturers frequently revise PCB layouts, connector pinouts, and mounting tolerances
            between production batches without notice. Always treat the manufacturer&rsquo;s
            official manual for your exact hardware revision as the primary source of truth.
          </p>
        </div>
      </div>
    </div>
  );
}

export function SafetyDisclaimersModal({ isOpen, onClose }: SafetyDisclaimersModalProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (isOpen) {
      if (!dialog.open) {
        if (typeof dialog.showModal === "function") {
          dialog.showModal();
        } else {
          dialog.setAttribute("open", "");
        }
      }
    } else if (dialog.open) {
      if (typeof dialog.close === "function") {
        dialog.close();
      } else {
        dialog.removeAttribute("open");
      }
    }
  }, [isOpen]);

  const handleCancel = (e: React.SyntheticEvent) => {
    e.preventDefault();
    onClose();
  };

  if (!isOpen) return null;

  return (
    <dialog
      ref={dialogRef}
      closedby="any"
      aria-labelledby="safety-disclaimers-modal-title"
      data-testid="safety-disclaimers-modal"
      onCancel={handleCancel}
      onClick={(e) => {
        const dialog = dialogRef.current;
        if (!dialog || e.target !== dialog) return;
        const rect = dialog.getBoundingClientRect();
        if (rect.width === 0 && rect.height === 0) {
          onClose();
          return;
        }
        const isInDialog =
          rect.top <= e.clientY &&
          e.clientY <= rect.top + rect.height &&
          rect.left <= e.clientX &&
          e.clientX <= rect.left + rect.width;
        if (!isInDialog) {
          onClose();
        }
      }}
      className="backdrop:bg-black/60 backdrop:backdrop-blur-xs fixed inset-0 m-auto max-w-3xl w-[92vw] max-h-[85vh] p-0 rounded-2xl bg-zinc-900 border border-zinc-800 shadow-2xl overflow-hidden flex flex-col z-50 text-zinc-100"
    >
      {/* Modal Header */}
      <div className="p-4 sm:p-5 border-b border-zinc-800 flex items-center justify-between gap-3 bg-zinc-900">
        <div className="flex items-center gap-2.5 min-w-0">
          <div className="p-2 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-400 shrink-0">
            <ShieldAlert size={18} />
          </div>
          <div className="min-w-0">
            <h2
              id="safety-disclaimers-modal-title"
              className="text-base sm:text-lg font-bold text-zinc-100 truncate"
            >
              Safety Disclaimers
            </h2>
            <p className="text-xs text-zinc-400 truncate">
              Hardware, electrical, mechanical, and local regulatory responsibilities
            </p>
          </div>
        </div>

        <button
          type="button"
          onClick={onClose}
          aria-label="Close safety disclaimers"
          data-testid="close-safety-disclaimers-modal"
          className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer shrink-0"
        >
          <X size={20} />
        </button>
      </div>

      {/* Modal Content */}
      <div className="p-4 sm:p-6 overflow-y-auto space-y-4 flex-1">
        <SafetyDisclaimersCards />
      </div>

      {/* Modal Footer */}
      <div className="p-4 sm:px-6 border-t border-zinc-800 bg-zinc-950/60 flex flex-wrap items-center justify-between gap-3">
        <Link
          to="/terms"
          onClick={onClose}
          className="inline-flex items-center gap-1.5 text-xs font-medium text-zinc-400 hover:text-zinc-200 underline underline-offset-2 transition-colors"
        >
          <Scale size={13} className="shrink-0" />
          <span>View Full Terms of Service</span>
        </Link>

        <button
          type="button"
          onClick={onClose}
          data-testid="acknowledge-safety-disclaimers-btn"
          className="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-colors cursor-pointer"
        >
          Close
        </button>
      </div>
    </dialog>
  );
}

export interface SafetyDisclaimersButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  children?: React.ReactNode;
}

export function SafetyDisclaimersButton({
  children = "Safety Disclaimers",
  className = "text-amber-400/90 hover:text-amber-300 underline underline-offset-2 font-medium transition-colors cursor-pointer",
  onClick,
  ...rest
}: SafetyDisclaimersButtonProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button
        type="button"
        onClick={(e) => {
          onClick?.(e);
          setIsOpen(true);
        }}
        className={className}
        {...rest}
      >
        {children}
      </button>
      <SafetyDisclaimersModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  );
}
