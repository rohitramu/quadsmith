import { useState, useEffect, useRef } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { ConnectError, Code } from "@connectrpc/connect";
import { Moon, Sun, Loader2, ArrowLeft, Mail, AlertTriangle, ShieldAlert } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import logoDark from "../assets/quadsmith-logo-dark.svg";
import logoLight from "../assets/quadsmith-logo-light.svg";
import { OtpVerificationErrorSchema } from "../gen/quadsmith/auth_pb";

export function LoginPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { isAuthenticated, sendEmailOtp, verifyEmailOtp, verifyMagicLink } = useAuth();

  const redirectUrl = searchParams.get("redirect") || "/";
  const magicToken = searchParams.get("token");

  const [theme, setTheme] = useState(localStorage.getItem("theme") || "dark");
  const [step, setStep] = useState<"email" | "otp">(magicToken ? "otp" : "email");
  const [email, setEmail] = useState("");
  const [otpDigits, setOtpDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const [hintDigits, setHintDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const [isLoading, setIsLoading] = useState(false);
  const [magicLinkLoading, setMagicLinkLoading] = useState(!!magicToken);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [remainingAttempts, setRemainingAttempts] = useState<number | null>(null);
  const [isExhausted, setIsExhausted] = useState(false);
  const [isRateLimited, setIsRateLimited] = useState(false);
  const [cooldown, setCooldown] = useState(0);
  const [exhaustedCooldown, setExhaustedCooldown] = useState(0);

  const otpInputRefs = useRef<(HTMLInputElement | null)[]>([]);

  // Cooldown countdown timer
  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setInterval(() => {
      setCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [cooldown]);

  // Exhausted cooldown countdown timer (120s)
  useEffect(() => {
    if (exhaustedCooldown <= 0) return;
    const timer = setInterval(() => {
      setExhaustedCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [exhaustedCooldown]);

  useEffect(() => {
    if (theme === "dark") {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
    localStorage.setItem("theme", theme);
  }, [theme]);

  // If already logged in, redirect immediately
  useEffect(() => {
    if (isAuthenticated) {
      navigate(redirectUrl, { replace: true });
    }
  }, [isAuthenticated, navigate, redirectUrl]);

  // Magic link auto-verification
  useEffect(() => {
    const token = magicToken;
    if (token && !isAuthenticated) {
      async function handleMagicLink() {
        if (!token) return;
        setIsLoading(true);
        setErrorMsg(null);
        try {
          await verifyMagicLink(token);
          navigate(redirectUrl, { replace: true });
        } catch (err: unknown) {
          if (err instanceof Error) {
            setErrorMsg(err.message);
          } else {
            setErrorMsg("Failed to verify magic link. Please request a new one.");
          }
          setMagicLinkLoading(false);
        } finally {
          setIsLoading(false);
        }
      }
      handleMagicLink();
    }
  }, [magicToken, isAuthenticated, verifyMagicLink, navigate, redirectUrl]);

  const toggleTheme = () => setTheme(theme === "dark" ? "light" : "dark");

  const handleSendEmail = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    const trimmed = email.trim();
    if (!trimmed) {
      setErrorMsg("Please enter your email address");
      return;
    }

    setIsLoading(true);
    setErrorMsg(null);
    try {
      const res = await sendEmailOtp(trimmed);
      setCooldown(res.cooldownSeconds);
      setExhaustedCooldown(0);
      setStep("otp");
      setRemainingAttempts(null);
      setIsExhausted(false);
      setIsRateLimited(false);
      setHintDigits(["", "", "", "", "", ""]);
      setOtpDigits(["", "", "", "", "", ""]);
      setTimeout(() => otpInputRefs.current[0]?.focus(), 100);
    } catch (err: unknown) {
      if (err instanceof ConnectError) {
        if (
          err.code === Code.ResourceExhausted &&
          (err.rawMessage.toLowerCase().includes("too many") ||
            err.rawMessage.toLowerCase().includes("minutes"))
        ) {
          setIsRateLimited(true);
        }
        setErrorMsg(err.rawMessage);
      } else if (err instanceof Error) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg("Failed to send verification code. Please try again.");
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleOtpChange = (index: number, value: string) => {
    // Only accept numeric input
    const clean = value.replace(/\D/g, "");
    if (!clean && value !== "") return;

    const newDigits = [...otpDigits];
    if (clean.length > 1) {
      // Pasted multi-character string
      const chars = clean.slice(0, 6).split("");
      chars.forEach((c, i) => {
        newDigits[i] = c;
      });
      setOtpDigits(newDigits);
      const nextIndex = Math.min(chars.length, 5);
      otpInputRefs.current[nextIndex]?.focus();
      const code = newDigits.join("");
      if (code.length === 6 && !isLoading) {
        handleVerifyOtp(code);
      }
      return;
    }

    newDigits[index] = clean;
    setOtpDigits(newDigits);

    // Auto-advance
    if (clean && index < 5) {
      otpInputRefs.current[index + 1]?.focus();
    }

    const code = newDigits.join("");
    if (code.length === 6 && !isLoading) {
      handleVerifyOtp(code);
    }
  };

  const handleKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Backspace" && !otpDigits[index] && index > 0) {
      otpInputRefs.current[index - 1]?.focus();
    }
  };

  const handleVerifyOtp = async (explicitCode?: string) => {
    const code = explicitCode || otpDigits.join("");
    if (code.length < 6) {
      setErrorMsg("Please enter all 6 digits of the verification code");
      return;
    }
    if (isLoading) return;

    setIsLoading(true);
    setErrorMsg(null);
    try {
      await verifyEmailOtp(email, code);
      navigate(redirectUrl, { replace: true });
    } catch (err: unknown) {
      // Clear fields and show incorrect code as faded placeholder hints
      setHintDigits(code.split(""));
      setOtpDigits(["", "", "", "", "", ""]);
      setTimeout(() => otpInputRefs.current[0]?.focus(), 10);

      if (err instanceof ConnectError) {
        if (
          err.code === Code.ResourceExhausted &&
          (err.rawMessage.toLowerCase().includes("too many") ||
            err.rawMessage.toLowerCase().includes("minutes"))
        ) {
          setIsRateLimited(true);
          setErrorMsg(err.rawMessage);
          return;
        }
        const detail = err.findDetails(OtpVerificationErrorSchema)[0];
        if (detail) {
          setRemainingAttempts(detail.remainingAttempts);
          setIsExhausted(detail.isExhausted);
          if (detail.isExhausted) {
            setExhaustedCooldown(detail.cooldownSeconds || 120);
          }
          setErrorMsg(detail.message);
        } else {
          setErrorMsg(err.rawMessage);
        }
      } else if (err instanceof Error) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg("Verification failed. Please try again.");
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen w-full flex flex-col items-center justify-center p-4 bg-zinc-100 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-50 font-sans relative">
      {/* Theme Switcher Button */}
      <button
        onClick={toggleTheme}
        aria-label="Toggle theme"
        className="absolute top-4 right-4 p-2.5 rounded-full border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 shadow-2xs transition"
      >
        {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
      </button>

      {/* Main Card */}
      <div className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl p-8 space-y-6">
        {/* Header Logo */}
        <div className="text-center space-y-3">
          <div className="flex justify-center">
            <img
              src={theme === "dark" ? logoDark : logoLight}
              alt="Quadsmith"
              className="h-10 w-auto"
            />
          </div>
          <h1 className="text-xl font-bold tracking-tight">
            {step === "email" ? "Sign in to Quadsmith" : "Verify your email"}
          </h1>
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            {step === "email"
              ? "Design, evaluate, and share high-performance FPV drone builds."
              : `We sent a single-use login code to ${email}`}
          </p>
        </div>

        {/* Magic Link Loading Banner */}
        {magicLinkLoading && (
          <div className="p-4 rounded-xl border border-blue-500/20 bg-blue-500/10 text-blue-600 dark:text-blue-400 flex items-center gap-3 text-xs">
            <Loader2 size={18} className="animate-spin text-blue-500 shrink-0" />
            <span>Verifying your magic link token...</span>
          </div>
        )}

        {/* Dynamic Error / Attempt Warning Banner */}
        {errorMsg && (
          <div
            className={`p-3.5 rounded-xl border text-xs flex items-start gap-2.5 ${
              isExhausted || isRateLimited
                ? "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
                : remainingAttempts === 1
                  ? "border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300"
                  : remainingAttempts === 2
                    ? "border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-400"
                    : "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
            }`}
          >
            {isExhausted || isRateLimited ? (
              <ShieldAlert size={16} className="text-red-500 shrink-0 mt-0.5" />
            ) : remainingAttempts !== null ? (
              <AlertTriangle size={16} className="text-amber-500 shrink-0 mt-0.5" />
            ) : (
              <AlertTriangle size={16} className="text-red-500 shrink-0 mt-0.5" />
            )}
            <div className="space-y-0.5 flex-1">
              <p className="font-semibold">{errorMsg}</p>
              {remainingAttempts !== null && remainingAttempts > 0 && (
                <p className="text-[11px] opacity-90">
                  {remainingAttempts === 1
                    ? "Final attempt remaining before this code is invalidated."
                    : `${remainingAttempts} attempts remaining for this verification challenge.`}
                </p>
              )}
            </div>
          </div>
        )}

        {/* STEP 1: Email Form */}
        {step === "email" && (
          <form onSubmit={handleSendEmail} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">
                Email Address
              </label>
              <div className="relative">
                <Mail
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  size={16}
                />
                <input
                  type="email"
                  required
                  placeholder="alex@quadsmith.dev"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full pl-9 pr-3 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 text-sm text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={isLoading || !email.trim() || cooldown > 0}
              className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-sm flex items-center justify-center gap-2 ${
                cooldown > 0
                  ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                  : "text-white bg-blue-600 hover:bg-blue-500 disabled:opacity-50 cursor-pointer"
              }`}
            >
              {isLoading ? (
                <>
                  <Loader2 size={16} className="animate-spin" />
                  <span>Sending code...</span>
                </>
              ) : cooldown > 0 ? (
                <span>Send Verification Code ({cooldown}s)</span>
              ) : (
                <>
                  <span>Send Verification Code</span>
                  <span>&rarr;</span>
                </>
              )}
            </button>
          </form>
        )}

        {/* STEP 2: OTP Verification Form */}
        {step === "otp" && (
          <div className="space-y-5">
            {/* Dual-path instructions */}
            <div className="p-3.5 rounded-xl border border-blue-500/20 bg-blue-500/10 text-zinc-700 dark:text-zinc-300 text-xs space-y-2 text-center">
              <p className="text-[11px] text-zinc-600 dark:text-zinc-400">
                Enter the <strong>6-digit code</strong> sent to your email below.
              </p>
              <div className="flex items-center justify-center gap-2">
                <div className="h-px bg-blue-500/20 flex-1"></div>
                <span className="text-[10px] font-bold text-blue-600 dark:text-blue-400 uppercase tracking-wider">
                  OR
                </span>
                <div className="h-px bg-blue-500/20 flex-1"></div>
              </div>
              <p className="text-[11px] text-zinc-600 dark:text-zinc-400">
                Click the link in the email.
              </p>
            </div>

            {/* 6 Digit Inputs */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                  Verification Code
                </label>
                {remainingAttempts !== null &&
                  remainingAttempts > 0 &&
                  !isExhausted &&
                  !isRateLimited && (
                    <span
                      className={`text-[10px] font-mono font-bold ${
                        remainingAttempts === 1 ? "text-amber-500" : "text-rose-500"
                      }`}
                    >
                      {remainingAttempts} attempt{remainingAttempts === 1 ? "" : "s"} left
                    </span>
                  )}
              </div>

              <div className="flex items-center justify-between gap-2">
                {otpDigits.map((digit, index) => (
                  <input
                    key={index}
                    ref={(el) => {
                      otpInputRefs.current[index] = el;
                    }}
                    type="text"
                    inputMode="numeric"
                    maxLength={1}
                    placeholder={hintDigits[index] || ""}
                    value={digit}
                    disabled={isExhausted || isRateLimited || isLoading}
                    onChange={(e) => handleOtpChange(index, e.target.value)}
                    onKeyDown={(e) => handleKeyDown(index, e)}
                    className={`w-12 h-14 text-center font-mono text-2xl font-bold rounded-xl border bg-zinc-50 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-300 dark:placeholder:text-zinc-700 focus:outline-none transition ${
                      isExhausted || isRateLimited
                        ? "opacity-40 border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 text-zinc-400 dark:text-zinc-600 cursor-not-allowed"
                        : remainingAttempts !== null && remainingAttempts > 0
                          ? "border-2 border-rose-500 focus:border-rose-600"
                          : "border-zinc-300 dark:border-zinc-700 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
                    }`}
                  />
                ))}
              </div>
            </div>

            {/* Action Buttons & Status */}
            <div className="space-y-2.5">
              {isLoading && (
                <div className="flex items-center justify-center gap-2 py-1.5 text-xs font-medium text-blue-600 dark:text-blue-400">
                  <Loader2 size={16} className="animate-spin" />
                  <span>Verifying code...</span>
                </div>
              )}

              {isRateLimited ? (
                <button
                  type="button"
                  disabled
                  className="w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-xs flex items-center justify-center gap-2 text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                >
                  <ShieldAlert size={14} className="text-zinc-400 dark:text-zinc-500" />
                  <span>Temporarily Locked Out</span>
                </button>
              ) : isExhausted ? (
                <button
                  type="button"
                  onClick={() => handleSendEmail()}
                  disabled={exhaustedCooldown > 0 || isLoading}
                  className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-sm flex items-center justify-center gap-2 ${
                    exhaustedCooldown > 0
                      ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                      : "text-white bg-blue-600 hover:bg-blue-500 active:bg-blue-700 cursor-pointer"
                  }`}
                >
                  <span>
                    {exhaustedCooldown > 0
                      ? `Request New Verification Code (${exhaustedCooldown}s)`
                      : "Request New Verification Code"}
                  </span>
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => handleSendEmail()}
                  disabled={cooldown > 0 || isLoading}
                  className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-xs flex items-center justify-center gap-2 ${
                    cooldown > 0
                      ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                      : "text-zinc-800 dark:text-zinc-100 bg-zinc-100 dark:bg-zinc-800 hover:bg-zinc-200 dark:hover:bg-zinc-700 border border-zinc-300 dark:border-zinc-600 cursor-pointer"
                  }`}
                >
                  <span>{cooldown > 0 ? `Resend Email (${cooldown}s)` : "Resend Email"}</span>
                </button>
              )}

              {/* Back Option */}
              <div className="flex items-center text-xs pt-1">
                <button
                  type="button"
                  onClick={() => {
                    setStep("email");
                    setErrorMsg(null);
                    setRemainingAttempts(null);
                    setIsExhausted(false);
                    setIsRateLimited(false);
                    setExhaustedCooldown(0);
                    setHintDigits(["", "", "", "", "", ""]);
                    setOtpDigits(["", "", "", "", "", ""]);
                  }}
                  className="text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 flex items-center gap-1 transition cursor-pointer"
                >
                  <ArrowLeft size={13} />
                  <span>Back</span>
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
