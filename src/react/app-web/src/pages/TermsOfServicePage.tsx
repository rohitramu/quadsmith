import { Link } from "react-router-dom";
import {
  ShieldAlert,
  FlaskConical,
  Zap,
  Wind,
  Radio,
  Layers,
  Scale,
  Lock,
  ChevronRight,
  AlertTriangle,
  CheckCircle2,
} from "lucide-react";
import { useDocumentMeta } from "@quadsmith/shared";
import logo from "../assets/quadsmith-logo.svg";

export interface TermsOfServicePageProps {
  standalone?: boolean;
}

export function TermsOfServicePage({ standalone = false }: TermsOfServicePageProps) {
  useDocumentMeta({
    title: "Terms of Service & Disclaimers — Quadsmith",
    description:
      "Quadsmith Terms of Service, FPV drone hardware and LiPo electrical safety disclaimers, and Experimental Build Evaluator documentation.",
    image: "/og-default.png",
  });

  const content = (
    <div className="max-w-4xl mx-auto space-y-8">
      {/* Breadcrumb Navigation */}
      <nav
        aria-label="Breadcrumb"
        className="text-sm text-zinc-400 flex items-center gap-1.5 flex-wrap"
      >
        {standalone ? (
          <Link to="/login" className="hover:text-zinc-100 hover:underline transition-colors">
            Sign In
          </Link>
        ) : (
          <Link to="/" className="hover:text-zinc-100 hover:underline transition-colors">
            Home
          </Link>
        )}
        <ChevronRight size={14} className="text-zinc-500 shrink-0" />
        <span className="text-zinc-100 font-medium">Terms of Service &amp; Disclaimers</span>
      </nav>

      {/* Page Header */}
      <header className="p-6 sm:p-8 rounded-2xl border border-zinc-800 bg-gradient-to-br from-zinc-900/80 to-zinc-950 shadow-xs space-y-4">
        <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-blue-500/10 text-blue-400 border border-blue-500/20">
          <Scale size={13} className="shrink-0" />
          <span>Legal, Safety &amp; Engineering Documentation</span>
        </div>

        <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight text-zinc-50 text-balance">
          Terms of Service &amp; Disclaimers
        </h1>

        <p className="text-sm sm:text-base text-zinc-400 leading-relaxed text-pretty">
          Quadsmith is an engineering reference catalog, compatibility checker, and experimental
          flight performance simulator for FPV multirotor builders. Please review these safety
          disclaimers and terms carefully before assembling, powering, or flying any aircraft.
        </p>

        {/* Quick Section Navigation */}
        <div className="pt-2 flex flex-wrap items-center gap-2 text-xs">
          <a
            href="#disclaimers"
            className="px-3 py-1.5 rounded-xl border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-200 font-medium transition-colors"
          >
            Safety &amp; Engineering Disclaimers
          </a>
          <a
            href="#experimental-evaluator"
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl border border-amber-500/30 bg-amber-500/10 hover:bg-amber-500/15 text-amber-300 font-medium transition-colors"
          >
            <FlaskConical size={12} className="shrink-0" />
            <span>Experimental Build Evaluator</span>
          </a>
          <a
            href="#terms-of-service"
            className="px-3 py-1.5 rounded-xl border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-200 font-medium transition-colors"
          >
            Terms of Service
          </a>
          <a
            href="#privacy"
            className="px-3 py-1.5 rounded-xl border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-200 font-medium transition-colors"
          >
            Data &amp; Privacy
          </a>
        </div>
      </header>

      {/* ============================================================ */}
      {/* SECTION 1: SAFETY & ENGINEERING DISCLAIMERS                  */}
      {/* ============================================================ */}
      <section
        id="disclaimers"
        aria-labelledby="disclaimers-heading"
        className="p-6 sm:p-8 rounded-2xl border border-zinc-800 bg-zinc-900/40 space-y-6 scroll-mt-20"
      >
        <div className="border-b border-zinc-800 pb-4">
          <h2
            id="disclaimers-heading"
            className="text-xl sm:text-2xl font-bold text-zinc-100 flex items-center gap-2.5 text-balance"
          >
            <ShieldAlert className="text-amber-400 shrink-0" size={22} />
            <span>Safety &amp; Engineering Disclaimers</span>
          </h2>
          <p className="text-xs sm:text-sm text-zinc-400 mt-1">
            Critical hardware, electrical, mechanical, and regulatory warnings for multirotor
            builders and pilots.
          </p>
        </div>

        {/* Critical Warning Callout */}
        <div
          role="note"
          aria-label="Critical Safety Warning"
          className="p-4 sm:p-5 rounded-xl border border-amber-500/30 bg-amber-950/25 text-amber-200 flex items-start gap-3.5"
        >
          <AlertTriangle className="text-amber-400 shrink-0 mt-0.5" size={20} />
          <div className="space-y-1 text-xs sm:text-sm leading-relaxed text-pretty">
            <p className="font-bold text-amber-100">
              Important Hardware, Electrical, and Flight Safety Notice
            </p>
            <p className="text-amber-200/90">
              FPV drones combine high-discharge lithium batteries, high-current brushless power
              systems, rapidly spinning propellers, and radio frequency transmitters. Improper
              wiring, voltage mismatches, or mechanical failures can cause fires, severe personal
              injury, or property damage. Always verify every component against official
              manufacturer datasheets and test on the bench before applying full battery power.
            </p>
          </div>
        </div>

        {/* 4 Detailed Disclaimer Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-5 rounded-xl border border-zinc-800 bg-zinc-950/70 space-y-2.5">
            <div className="flex items-center gap-2 text-sm font-bold text-zinc-100">
              <Zap size={16} className="text-amber-400 shrink-0" />
              <h3>Electrical &amp; LiPo Battery Safety</h3>
            </div>
            <p className="text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
              Lithium Polymer (LiPo/LiHV) and Lithium-Ion batteries carry severe risks of fire,
              explosion, and thermal runaway if short-circuited, overcharged, punctured, or operated
              beyond their continuous or burst current ratings. Never rely solely on software
              compatibility checks for wiring. Always inspect board silkscreens, verify pinouts and
              polarity with a digital multimeter, and use a current-limiting smoke stopper when
              powering a build for the first time.
            </p>
          </div>

          <div className="p-5 rounded-xl border border-zinc-800 bg-zinc-950/70 space-y-2.5">
            <div className="flex items-center gap-2 text-sm font-bold text-zinc-100">
              <Wind size={16} className="text-rose-400 shrink-0" />
              <h3>Mechanical &amp; Flight Safety</h3>
            </div>
            <p className="text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
              High-RPM brushless motors and propellers can inflict severe lacerations and eye
              injuries. Always remove all propellers before connecting a battery on the bench for
              flight controller configuration, ESC flashing, or motor direction testing. Verify that
              motor mounting screws do not contact or short stator windings, confirm frame
              clearances, and validate receiver failsafe behavior prior to every flight.
            </p>
          </div>

          <div className="p-5 rounded-xl border border-zinc-800 bg-zinc-950/70 space-y-2.5">
            <div className="flex items-center gap-2 text-sm font-bold text-zinc-100">
              <Radio size={16} className="text-blue-400 shrink-0" />
              <h3>RF Spectrum &amp; Aviation Compliance</h3>
            </div>
            <p className="text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
              Operating Unmanned Aircraft Systems (UAS) and radio/video transmitters (VTX and RC
              control links) is governed by national and local aviation and telecommunications
              authorities (such as FAA Part 107 / Recreational Flyer rules, Remote ID, EASA, and
              FCC/CE amateur radio regulations). Catalog listings of transmit power levels or
              frequency bands do not constitute legal authorization to operate those transmitters in
              your jurisdiction.
            </p>
          </div>

          <div className="p-5 rounded-xl border border-zinc-800 bg-zinc-950/70 space-y-2.5">
            <div className="flex items-center gap-2 text-sm font-bold text-zinc-100">
              <Layers size={16} className="text-emerald-400 shrink-0" />
              <h3>Catalog Data &amp; Hardware Revisions</h3>
            </div>
            <p className="text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
              Component specifications, dimensions, weights, and compatibility rules are compiled
              from manufacturer documentation and community contributions on a best-effort basis.
              Manufacturers frequently revise PCB layouts, JST connector pinouts, and mounting
              tolerances between production batches without notice. Always treat the manufacturer’s
              official manual for your exact hardware revision as the primary source of truth.
            </p>
          </div>
        </div>
      </section>

      {/* ============================================================ */}
      {/* SECTION 2: EXPERIMENTAL BUILD EVALUATOR                      */}
      {/* ============================================================ */}
      <section
        id="experimental-evaluator"
        aria-labelledby="experimental-evaluator-heading"
        className="p-6 sm:p-8 rounded-2xl border border-zinc-800 bg-zinc-900/40 space-y-5 scroll-mt-20"
      >
        <div className="border-b border-zinc-800 pb-4">
          <div className="flex items-center gap-2.5 flex-wrap">
            <FlaskConical className="text-amber-400 shrink-0" size={22} />
            <h2
              id="experimental-evaluator-heading"
              className="text-xl sm:text-2xl font-bold text-zinc-100 text-balance"
            >
              Experimental Build Evaluator &amp; Compatibility Engine
            </h2>
            <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-amber-500/10 text-amber-400 border border-amber-500/20">
              <FlaskConical size={11} className="shrink-0" />
              <span>Experimental</span>
            </span>
          </div>
          <p className="text-xs sm:text-sm text-zinc-400 mt-1">
            Why Build Evaluator telemetry and compatibility checks are designated as Experimental.
          </p>
        </div>

        <div className="space-y-4 text-xs sm:text-sm text-zinc-300 leading-relaxed text-pretty">
          <p>
            The Quadsmith <strong>Build Evaluator</strong> (featured on Build Profile pages, Build
            Feed cards, and live inside The Forge configurator) is an{" "}
            <strong className="text-amber-400">Experimental</strong> analytical physics engine. All
            computed telemetry—including All-Up Weight (AUW), Thrust-to-Weight Ratio (TWR), Hover
            Throttle percentage, Maximum Acceleration, Top Speed, and Estimated Flight Time—are{" "}
            <strong>theoretical estimates</strong> rather than guaranteed empirical measurements.
          </p>

          <ul className="space-y-2.5 bg-zinc-950/70 p-5 rounded-xl border border-zinc-800">
            <li className="flex items-start gap-2.5">
              <CheckCircle2 size={16} className="text-amber-400 shrink-0 mt-0.5" />
              <span>
                <strong>Idealized Aerodynamic &amp; Electrical Assumptions:</strong> Calculations
                assume standard sea-level air density (1.225 kg/m³), momentum-theory propeller
                thrust and power coefficients, simplified internal battery resistance voltage sag,
                and uniform motor electromechanical efficiency.
              </span>
            </li>
            <li className="flex items-start gap-2.5">
              <CheckCircle2 size={16} className="text-amber-400 shrink-0 mt-0.5" />
              <span>
                <strong>Real-World Performance Deviations:</strong> Actual flight performance and
                current draw vary significantly based on propeller blade airfoil geometry, stator
                magnetic saturation, ESC PWM frequency and timing, PID and filter tuning, ambient
                temperature, altitude, wind, and battery health.
              </span>
            </li>
            <li className="flex items-start gap-2.5">
              <CheckCircle2 size={16} className="text-amber-400 shrink-0 mt-0.5" />
              <span>
                <strong>Advisory Compatibility Checks:</strong> The Compatibility Checker evaluates
                modeled mounting patterns, propeller clearance, voltage ranges, and RF protocols. It
                cannot detect every 3D physical interference, standoff collision, capacitor
                clearance issue, or custom wiring requirement.
              </span>
            </li>
          </ul>
        </div>
      </section>

      {/* ============================================================ */}
      {/* SECTION 3: TERMS OF SERVICE                                  */}
      {/* ============================================================ */}
      <section
        id="terms-of-service"
        aria-labelledby="tos-heading"
        className="p-6 sm:p-8 rounded-2xl border border-zinc-800 bg-zinc-900/40 space-y-6 scroll-mt-20"
      >
        <div className="border-b border-zinc-800 pb-4">
          <h2
            id="tos-heading"
            className="text-xl sm:text-2xl font-bold text-zinc-100 flex items-center gap-2.5 text-balance"
          >
            <Scale className="text-blue-400 shrink-0" size={22} />
            <span>Terms of Service</span>
          </h2>
          <p className="text-xs sm:text-sm text-zinc-400 mt-1">
            Conditions governing access to the Quadsmith web application, APIs, and CLI tools.
          </p>
        </div>

        <div className="space-y-5 text-xs sm:text-sm text-zinc-300 leading-relaxed text-pretty">
          <div className="space-y-1.5">
            <h3 className="text-sm sm:text-base font-bold text-zinc-100">1. Acceptance of Terms</h3>
            <p className="text-zinc-400">
              By accessing or using Quadsmith—including browsing the component catalog, configuring
              builds in The Forge, using the Experimental Build Evaluator, signing into an account,
              or interacting with the Quadsmith API or CLI—you agree to be bound by these Terms of
              Service and all Safety &amp; Engineering Disclaimers on this page.
            </p>
          </div>

          <div className="space-y-1.5">
            <h3 className="text-sm sm:text-base font-bold text-zinc-100">
              2. &ldquo;AS IS&rdquo; &amp; &ldquo;AS AVAILABLE&rdquo; Disclaimer of Warranties
            </h3>
            <p className="text-zinc-400">
              Quadsmith and all catalog data, compatibility diagnostics, and experimental physics
              evaluations are provided strictly on an <strong>&ldquo;AS IS&rdquo;</strong> and{" "}
              <strong>&ldquo;AS AVAILABLE&rdquo;</strong> basis, without warranties of any kind,
              either express, implied, or statutory—including, without limitation, implied
              warranties of merchantability, fitness for a particular purpose, safety, accuracy, or
              non-infringement.
            </p>
          </div>

          <div className="space-y-1.5">
            <h3 className="text-sm sm:text-base font-bold text-zinc-100">
              3. Limitation of Liability &amp; Assumption of Risk
            </h3>
            <p className="text-zinc-400">
              You expressly understand and agree that you assume all risks associated with
              selecting, wiring, soldering, powering, and flying multirotor hardware. To the fullest
              extent permitted by law, Quadsmith, its operators, maintainers, and contributors shall
              not be liable for any direct, indirect, incidental, special, consequential, or
              exemplary damages—including damaged or destroyed electronics, battery fires, lost
              aircraft (&ldquo;flyaways&rdquo;), personal injury, property damage, or regulatory
              fines—arising from your use of or reliance on this platform.
            </p>
          </div>

          <div className="space-y-1.5">
            <h3 className="text-sm sm:text-base font-bold text-zinc-100">
              4. User Accounts &amp; Community Contributions
            </h3>
            <p className="text-zinc-400">
              Quadsmith uses passwordless authentication via verified email one-time passcodes
              (OTP), magic links, and OAuth providers. You are responsible for maintaining the
              security of your linked email accounts and active sessions. By submitting public
              builds, custom catalog components, collections, reviews, Q&amp;A discussions, or
              reference links, you represent that you have the right to share that content and grant
              Quadsmith a non-exclusive license to display and distribute it across the platform.
            </p>
          </div>

          <div className="space-y-1.5">
            <h3 className="text-sm sm:text-base font-bold text-zinc-100">
              5. Acceptable Use &amp; Platform Moderation
            </h3>
            <p className="text-zinc-400">
              You agree not to submit intentionally inaccurate hardware safety specifications,
              abusive or unlawful content, spam, or automated scraping payloads that degrade service
              availability. Platform administrators reserve the right to moderate content and issue
              temporary or permanent account restrictions for violations of community standards.
            </p>
          </div>
        </div>
      </section>

      {/* ============================================================ */}
      {/* SECTION 4: DATA & PRIVACY SUMMARY                            */}
      {/* ============================================================ */}
      <section
        id="privacy"
        aria-labelledby="privacy-heading"
        className="p-6 sm:p-8 rounded-2xl border border-zinc-800 bg-zinc-900/40 space-y-4 scroll-mt-20"
      >
        <div className="border-b border-zinc-800 pb-4">
          <h2
            id="privacy-heading"
            className="text-xl sm:text-2xl font-bold text-zinc-100 flex items-center gap-2.5 text-balance"
          >
            <Lock className="text-emerald-400 shrink-0" size={22} />
            <span>Data &amp; Privacy</span>
          </h2>
          <p className="text-xs sm:text-sm text-zinc-400 mt-1">
            How Quadsmith handles account credentials, sessions, data portability, and deletion.
          </p>
        </div>

        <div className="space-y-3 text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
          <p>
            Quadsmith collects only the data required to operate your account and community
            contributions: your verified email address(es), linked OAuth identifiers, active session
            metadata, public profile handle and bio, and the builds, components, collections, and
            discussions you author.
          </p>
          <p>
            Authenticated users can inspect and revoke active sessions, download a complete
            machine-readable <code>.zip</code> archive of their personal data (GDPR Article 15 &amp;
            20 Data Export), or initiate account deletion with a 30-day restoration grace period at
            any time from{" "}
            <Link
              to="/account"
              className="text-blue-400 hover:text-blue-300 underline underline-offset-2 font-medium"
            >
              Account Settings
            </Link>
            . When an account is permanently purged after the 30-day grace period, personal email
            records and authentication credentials are erased while public communal contributions
            remain preserved under anonymized attribution.
          </p>
        </div>
      </section>
    </div>
  );

  if (!standalone) {
    return content;
  }

  return (
    <div className="min-h-screen w-full min-w-[320px] flex flex-col bg-zinc-950 text-zinc-50 font-sans">
      <header className="sticky top-0 z-40 flex items-center justify-between h-14 sm:h-16 px-4 sm:px-6 border-b border-zinc-800 bg-zinc-900">
        <Link to="/login" className="flex items-center hover:opacity-90 transition-opacity">
          <img src={logo} alt="Quadsmith" className="h-9 sm:h-11 w-auto" />
        </Link>
        <Link
          to="/login"
          className="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-colors"
        >
          Sign In
        </Link>
      </header>
      <main className="flex-1 p-4 sm:p-6 lg:p-8">{content}</main>
    </div>
  );
}
