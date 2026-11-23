import { Link } from "react-router-dom";
import { ShieldAlert, FlaskConical, Scale, Lock, ChevronRight, CheckCircle2 } from "lucide-react";
import { useDocumentMeta } from "@quadsmith/shared";
import logo from "../assets/quadsmith-logo.svg";
import { SafetyDisclaimersCards } from "../components/SafetyDisclaimersModal";

export interface TermsOfServicePageProps {
  standalone?: boolean;
}

export function TermsOfServicePage({ standalone = false }: TermsOfServicePageProps) {
  useDocumentMeta({
    title: "Terms of Service & Disclaimers — Quadsmith",
    description:
      "Quadsmith Terms of Service, FPV drone hardware and LiPo electrical safety disclaimers, local compliance responsibilities, and Experimental Build Evaluator notice.",
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
          <span>Legal &amp; Safety Documentation</span>
        </div>

        <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight text-zinc-50 text-balance">
          Terms of Service &amp; Disclaimers
        </h1>

        <p className="text-sm sm:text-base text-zinc-400 leading-relaxed text-pretty">
          Quadsmith is a reference catalog, compatibility checker, and experimental build evaluator
          for FPV multirotor builders. Please review these safety disclaimers and terms carefully
          before assembling, powering, or flying any aircraft, and always comply with local rules
          and regulations.
        </p>
      </header>

      {/* ============================================================ */}
      {/* SECTION 1: SAFETY DISCLAIMERS                                */}
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
            <span>Safety Disclaimers</span>
          </h2>
          <p className="text-xs sm:text-sm text-zinc-400 mt-1">
            Critical hardware, electrical, mechanical, and local compliance responsibilities for
            multirotor builders and pilots.
          </p>
        </div>

        <SafetyDisclaimersCards />
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
            Why the Build Evaluator and compatibility checks are designated as Experimental.
          </p>
        </div>

        <div className="space-y-4 text-xs sm:text-sm text-zinc-300 leading-relaxed text-pretty">
          <p>
            The Quadsmith <strong>Build Evaluator</strong> and compatibility checks are{" "}
            <strong className="text-amber-400">Experimental</strong> features provided strictly{" "}
            <strong>&ldquo;AS IS&rdquo;</strong> and <strong>&ldquo;AS AVAILABLE&rdquo;</strong>{" "}
            without warranties of airworthiness, compatibility, accuracy, or safety.
          </p>

          <ul className="space-y-2.5 bg-zinc-950/70 p-5 rounded-xl border border-zinc-800">
            <li className="flex items-start gap-2.5">
              <CheckCircle2 size={16} className="text-amber-400 shrink-0 mt-0.5" />
              <span>
                <strong>Provided &ldquo;AS IS&rdquo; Without Warranty:</strong> The Build Evaluator
                is an experimental tool for general informational comparison only and makes no
                representations or warranties regarding flight performance, airworthiness, or
                operational safety.
              </span>
            </li>
            <li className="flex items-start gap-2.5">
              <CheckCircle2 size={16} className="text-amber-400 shrink-0 mt-0.5" />
              <span>
                <strong>Advisory Compatibility Checks:</strong> Compatibility checks evaluate
                selected specifications on a best-effort basis only. Passing compatibility checks
                does not guarantee physical fitment, electrical compatibility, or safe operation.
              </span>
            </li>
            <li className="flex items-start gap-2.5">
              <CheckCircle2 size={16} className="text-amber-400 shrink-0 mt-0.5" />
              <span>
                <strong>Independent Verification Required:</strong> Always verify physical
                clearances, wiring pinouts, and voltage ratings against official manufacturer
                datasheets, and test your aircraft safely before flight.
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
        className="p-6 sm:p-8 rounded-2xl border border-zinc-800 bg-zinc-900/40 space-y-4 scroll-mt-20"
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
            General terms governing your use of Quadsmith.
          </p>
        </div>

        <div className="space-y-3 text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
          <p>
            By using Quadsmith, you agree to these terms and agree to comply with all applicable
            laws. Quadsmith and everything provided through it are offered strictly{" "}
            <strong className="text-zinc-200">&ldquo;AS IS&rdquo;</strong> and{" "}
            <strong className="text-zinc-200">&ldquo;AS AVAILABLE&rdquo;</strong> without warranties
            of any kind, express or implied.
          </p>

          <p>
            You use Quadsmith entirely at your own risk. To the fullest extent permitted by law,
            Quadsmith is not liable for any damages, losses, or claims arising from your use of or
            inability to use the service, and you agree to indemnify and hold Quadsmith harmless
            from any claims arising from your use, your content, or your violation of these terms.
          </p>

          <p>
            You are responsible for your account and anything you submit, and grant Quadsmith a
            license to use and display that content in connection with the service. Quadsmith may
            modify or discontinue the service, remove content, restrict access, or update these
            terms at any time without notice or liability.
          </p>
        </div>
      </section>

      {/* ============================================================ */}
      {/* SECTION 4: DATA & PRIVACY                                    */}
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
            How Quadsmith handles information.
          </p>
        </div>

        <div className="space-y-3 text-xs sm:text-sm text-zinc-400 leading-relaxed text-pretty">
          <p>
            By using Quadsmith, you acknowledge that Quadsmith and its third-party service providers
            may collect and process information you provide as well as technical and usage data to
            operate, secure, and improve the service and comply with the law.
          </p>
          <p>Quadsmith takes reasonable measures to protect your information.</p>
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
