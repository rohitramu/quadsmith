import { useState, useEffect, useRef } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { ConnectError, Code } from "@connectrpc/connect";
import { Moon, Sun, Loader2, ArrowLeft, Mail, AlertTriangle, Lock } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import logoDark from "../assets/quadsmith-logo-dark.svg";
import logoLight from "../assets/quadsmith-logo-light.svg";
import { OtpVerificationErrorSchema } from "../gen/quadsmith/auth_pb";
import { getAvatarPalette, getInitial } from "../lib/avatarColor";

export function validateUsernameRules(val: string): string | null {
  const trimmed = val.trim().toLowerCase();
  if (trimmed.length < 3) {
    return "Usernames must be at least 3 characters long (3–20 characters).";
  }
  if (trimmed.length > 20) {
    return "Usernames cannot exceed 20 characters (3–20 characters max).";
  }
  if (!/^[a-z]/.test(trimmed)) {
    return "Username must start with a lowercase letter (a–z).";
  }
  if (!/[a-z0-9]$/.test(trimmed)) {
    return "Username must end with a letter or number (cannot end with an underscore).";
  }
  if (!/^[a-z0-9_]+$/.test(trimmed)) {
    return "Username can only contain lowercase letters, numbers, and underscores.";
  }
  if (trimmed.includes("__")) {
    return "Username cannot contain consecutive underscores (__).";
  }
  if (trimmed.includes("quadsmith") || trimmed.includes("quad_smith")) {
    return "Username cannot contain 'quadsmith' or 'quad_smith'.";
  }
  return null;
}

export function LoginPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const {
    isAuthenticated,
    sendEmailOtp,
    verifyEmailOtp,
    verifyMagicLink,
    completeAccountSetup,
    checkUsernameAvailable,
  } = useAuth();

  const redirectUrl = searchParams.get("redirect") || "/";
  const magicToken = searchParams.get("token");

  const [theme, setTheme] = useState(localStorage.getItem("theme") || "dark");
  const [step, setStep] = useState<"email" | "otp" | "setup">(magicToken ? "otp" : "email");
  const [email, setEmail] = useState("");
  const [otpDigits, setOtpDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const [hintDigits, setHintDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const [isLoading, setIsLoading] = useState(false);
  const [magicLinkLoading, setMagicLinkLoading] = useState(!!magicToken);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [remainingAttempts, setRemainingAttempts] = useState<number | null>(null);
  const [isExhausted, setIsExhausted] = useState(false);
  const [isRateLimited, setIsRateLimited] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [cooldown, setCooldown] = useState(0);
  const [exhaustedCooldown, setExhaustedCooldown] = useState(0);
  const [rateLimitCooldown, setRateLimitCooldown] = useState(0);

  // Account Setup State
  const [setupToken, setSetupToken] = useState("");
  const [username, setUsername] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [usernameStatus, setUsernameStatus] = useState<
    "idle" | "checking" | "available" | "unavailable" | "invalid"
  >("idle");
  const [usernameMessage, setUsernameMessage] = useState<string | null>(null);
  const [isCompletingSetup, setIsCompletingSetup] = useState(false);
  const [setupError, setSetupError] = useState<string | null>(null);

  const otpInputRefs = useRef<(HTMLInputElement | null)[]>([]);

  // Cooldown countdown timer
  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setInterval(() => {
      setCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [cooldown]);

  // Exhausted cooldown countdown timer (60s)
  useEffect(() => {
    if (exhaustedCooldown <= 0) return;
    const timer = setInterval(() => {
      setExhaustedCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [exhaustedCooldown]);

  // Rate limit cooldown countdown timer (60m0s -> 3600s)
  useEffect(() => {
    if (rateLimitCooldown <= 0) return;
    const timer = setInterval(() => {
      setRateLimitCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [rateLimitCooldown]);

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
          const res = await verifyMagicLink(token);
          if (res?.accountSetupRequired && res.setupToken) {
            setSetupToken(res.setupToken);
            setStep("setup");
          } else {
            setIsSuccess(true);
            navigate(redirectUrl, { replace: true });
          }
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

  // Debounced username availability check with client-side naming rules validation
  useEffect(() => {
    const trimmed = username.trim().toLowerCase();
    if (!trimmed) {
      setUsernameStatus("idle");
      setUsernameMessage(null);
      return;
    }

    const ruleError = validateUsernameRules(trimmed);
    if (ruleError) {
      setUsernameStatus("invalid");
      setUsernameMessage(ruleError);
      return;
    }

    setUsernameStatus("checking");
    const timeout = setTimeout(async () => {
      try {
        const res = await checkUsernameAvailable(trimmed);
        if (res.available) {
          setUsernameStatus("available");
          setUsernameMessage("Username is available!");
        } else {
          const msg = res.message || "";
          const lower = msg.toLowerCase();
          if (
            lower.includes("character") ||
            lower.includes("letter") ||
            lower.includes("underscore") ||
            lower.includes("quadsmith") ||
            lower.includes("invalid") ||
            lower.includes("naming")
          ) {
            setUsernameStatus("invalid");
            setUsernameMessage(msg);
          } else {
            setUsernameStatus("unavailable");
            setUsernameMessage(msg || "This username is already taken. Please choose another.");
          }
        }
      } catch {
        setUsernameStatus("unavailable");
        setUsernameMessage("This username is already taken or unavailable.");
      }
    }, 250);

    return () => clearTimeout(timeout);
  }, [username, checkUsernameAvailable]);

  const toggleTheme = () => setTheme(theme === "dark" ? "light" : "dark");

  const handleSendEmail = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    const trimmed = email.trim();
    if (!trimmed) {
      setErrorMsg("Please enter your email address");
      return;
    }

    const isResend = step === "otp" && !isExhausted && !isRateLimited;

    setIsLoading(true);
    if (!isResend) {
      setErrorMsg(null);
    }
    try {
      const res = await sendEmailOtp(trimmed);
      setCooldown(res.cooldownSeconds);
      setExhaustedCooldown(0);
      setRateLimitCooldown(0);
      setStep("otp");
      if (!isResend) {
        setRemainingAttempts(null);
        setIsExhausted(false);
        setIsRateLimited(false);
        setHintDigits(["", "", "", "", "", ""]);
        setOtpDigits(["", "", "", "", "", ""]);
      }
      setTimeout(() => otpInputRefs.current[0]?.focus(), 100);
    } catch (err: unknown) {
      if (err instanceof ConnectError) {
        if (
          err.code === Code.ResourceExhausted &&
          (err.rawMessage.toLowerCase().includes("too many") ||
            err.rawMessage.toLowerCase().includes("minutes"))
        ) {
          setIsRateLimited(true);
          const detail = err.findDetails(OtpVerificationErrorSchema)[0];
          let cd = detail?.cooldownSeconds;
          if (!cd) {
            const matchMin = err.rawMessage.match(/(\d+)\s*minutes?/i);
            const matchSec = err.rawMessage.match(/(\d+)\s*seconds?/i);
            cd = matchMin
              ? parseInt(matchMin[1], 10) * 60
              : matchSec
                ? parseInt(matchSec[1], 10)
                : 3600;
          }
          setRateLimitCooldown(cd);
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

  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault();
    const pasted = e.clipboardData.getData("text").replace(/\D/g, "").slice(0, 6);
    if (!pasted) return;
    const newDigits = [...otpDigits];
    pasted.split("").forEach((c, i) => {
      newDigits[i] = c;
    });
    setOtpDigits(newDigits);
    const nextIndex = Math.min(pasted.length, 5);
    otpInputRefs.current[nextIndex]?.focus();
    if (pasted.length === 6 && !isLoading) {
      handleVerifyOtp(pasted);
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
      const res = await verifyEmailOtp(email, code);
      if (res?.accountSetupRequired && res.setupToken) {
        setSetupToken(res.setupToken);
        setStep("setup");
      } else {
        setIsSuccess(true);
        navigate(redirectUrl, { replace: true });
      }
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
          const detail = err.findDetails(OtpVerificationErrorSchema)[0];
          let cd = detail?.cooldownSeconds;
          if (!cd) {
            const matchMin = err.rawMessage.match(/(\d+)\s*minutes?/i);
            const matchSec = err.rawMessage.match(/(\d+)\s*seconds?/i);
            cd = matchMin
              ? parseInt(matchMin[1], 10) * 60
              : matchSec
                ? parseInt(matchSec[1], 10)
                : 3600;
          }
          setRateLimitCooldown(cd);
          setErrorMsg(err.rawMessage);
          return;
        }
        const detail = err.findDetails(OtpVerificationErrorSchema)[0];
        if (detail) {
          setRemainingAttempts(detail.remainingAttempts);
          setIsExhausted(detail.isExhausted);
          if (detail.isExhausted) {
            setExhaustedCooldown(detail.cooldownSeconds || 60);
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

  const handleCompleteSetup = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (usernameStatus !== "available" || !setupToken || isCompletingSetup) return;

    setIsCompletingSetup(true);
    setSetupError(null);
    try {
      await completeAccountSetup(
        setupToken,
        username.trim().toLowerCase(),
        avatarUrl.trim() || undefined,
      );
      setIsSuccess(true);
      navigate(redirectUrl, { replace: true });
    } catch (err: unknown) {
      if (err instanceof ConnectError) {
        setSetupError(err.rawMessage);
      } else if (err instanceof Error) {
        setSetupError(err.message);
      } else {
        setSetupError("Failed to complete account setup. Please try again.");
      }
    } finally {
      setIsCompletingSetup(false);
    }
  };

  const handleResetToEmail = () => {
    setStep("email");
    setEmail("");
    setSetupToken("");
    setUsername("");
    setAvatarUrl("");
    setUsernameStatus("idle");
    setUsernameMessage(null);
    setSetupError(null);
    setErrorMsg(null);
    setRemainingAttempts(null);
    setIsExhausted(false);
    setIsRateLimited(false);
    setIsSuccess(false);
    setExhaustedCooldown(0);
    setRateLimitCooldown(0);
    setHintDigits(["", "", "", "", "", ""]);
    setOtpDigits(["", "", "", "", "", ""]);
  };

  const renderErrorBanner = () => {
    if (!errorMsg) return null;
    return (
      <div
        data-testid="otp-error-banner"
        className={`p-3.5 rounded-xl border text-xs flex items-start gap-2.5 ${
          isExhausted || isRateLimited
            ? "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
            : remainingAttempts === 2
              ? "border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300"
              : remainingAttempts === 1
                ? "border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-400"
                : "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
        }`}
      >
        {isExhausted || isRateLimited ? (
          <Lock size={16} className="text-red-500 shrink-0 mt-0.5" />
        ) : remainingAttempts === 2 ? (
          <AlertTriangle size={16} className="text-amber-500 shrink-0 mt-0.5" />
        ) : (
          <AlertTriangle size={16} className="text-rose-500 shrink-0 mt-0.5" />
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
    );
  };

  const renderVerifyEmailScreen = (isInactiveSlide = false) => (
    <div
      className={`w-full p-8 space-y-6 ${isInactiveSlide ? "opacity-30 pointer-events-none" : ""}`}
    >
      {/* Header */}
      <div className="text-center space-y-3">
        <div className="flex justify-center">
          <img
            src={theme === "dark" ? logoDark : logoLight}
            alt="Quadsmith"
            className="h-10 w-auto"
          />
        </div>
        <h1 className="text-xl font-bold tracking-tight">Verify your email</h1>
        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          We sent a single-use login code to {email}
        </p>
      </div>

      {/* Magic Link Loading Banner */}
      {magicLinkLoading && (
        <div className="p-4 rounded-xl border border-blue-500/20 bg-blue-500/10 text-blue-600 dark:text-blue-400 flex items-center gap-3 text-xs">
          <Loader2 size={18} className="animate-spin text-blue-500 shrink-0" />
          <span>Verifying your magic link token...</span>
        </div>
      )}

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
        <p className="text-[11px] text-zinc-600 dark:text-zinc-400">Click the link in the email.</p>
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
                  remainingAttempts === 1 ? "text-rose-500" : "text-amber-500"
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
              pattern="[0-9]*"
              autoComplete={index === 0 ? "one-time-code" : "off"}
              maxLength={1}
              placeholder={hintDigits[index] || ""}
              value={digit}
              disabled={isSuccess || isExhausted || isRateLimited || isLoading}
              onChange={(e) => handleOtpChange(index, e.target.value)}
              onKeyDown={(e) => handleKeyDown(index, e)}
              onPaste={handlePaste}
              className={`w-12 h-14 text-center font-mono text-2xl font-bold rounded-xl border bg-zinc-50 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-300 dark:placeholder:text-zinc-700 focus:outline-none transition ${
                isSuccess || isExhausted || isRateLimited
                  ? "opacity-40 border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 text-zinc-400 dark:text-zinc-600 cursor-not-allowed"
                  : "border-zinc-300 dark:border-zinc-700 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              }`}
            />
          ))}
        </div>
      </div>

      {/* Error / Attempt Warning Banner below verification code and above buttons */}
      {renderErrorBanner()}

      {/* Action Buttons & Status */}
      <div className="space-y-2.5">
        {isLoading && (
          <div className="flex items-center justify-center gap-2 py-1.5 text-xs font-medium text-blue-600 dark:text-blue-400">
            <Loader2 size={16} className="animate-spin" />
            <span>Verifying code...</span>
          </div>
        )}

        {(isExhausted || isRateLimited) && (
          <button
            type="button"
            onClick={() => handleSendEmail()}
            disabled={(isRateLimited ? rateLimitCooldown > 0 : exhaustedCooldown > 0) || isLoading}
            className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-sm flex items-center justify-center gap-2 ${
              (isRateLimited ? rateLimitCooldown > 0 : exhaustedCooldown > 0)
                ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                : "text-white bg-blue-600 hover:bg-blue-500 active:bg-blue-700 cursor-pointer"
            }`}
          >
            <span>
              {isRateLimited
                ? rateLimitCooldown > 0
                  ? `Request New Verification Code (${Math.floor(rateLimitCooldown / 60)}m${rateLimitCooldown % 60}s)`
                  : "Request New Verification Code"
                : exhaustedCooldown > 0
                  ? `Request New Verification Code (${exhaustedCooldown}s)`
                  : "Request New Verification Code"}
            </span>
          </button>
        )}

        {/* Back Option */}
        <div className="flex items-center text-xs pt-1">
          <button
            type="button"
            onClick={handleResetToEmail}
            className="text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 flex items-center gap-1 transition cursor-pointer"
          >
            <ArrowLeft size={13} />
            <span>Back</span>
          </button>
        </div>
      </div>
    </div>
  );

  const renderAccountSetupScreen = () => (
    <div className="w-full p-8 space-y-6">
      {/* Header */}
      <div className="text-center space-y-2">
        <div className="flex justify-center mb-1">
          {/* Avatar Preview mirroring Layout.tsx top-bar palette */}
          <div
            data-testid="avatar-preview"
            className={`w-16 h-16 rounded-full border flex items-center justify-center font-bold text-2xl shadow-2xs overflow-hidden relative transition-colors duration-150 ${
              avatarUrl
                ? "border-zinc-200 dark:border-zinc-800"
                : getAvatarPalette(username.trim().toLowerCase() || email).badgeClass
            }`}
          >
            {avatarUrl ? (
              <img
                src={avatarUrl}
                alt="Avatar Preview"
                className="w-full h-full object-cover"
                onError={(e) => {
                  (e.currentTarget as HTMLElement).style.display = "none";
                }}
              />
            ) : (
              <span>{getInitial(username || email)}</span>
            )}
          </div>
        </div>
        <h1 className="text-xl font-bold tracking-tight">Account Setup</h1>
        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          Your username is your public display identifier across Quadsmith.
        </p>
      </div>

      {/* Form Fields */}
      <div className="space-y-4">
        {/* Username Input */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <label
              htmlFor="usernameInput"
              className="text-xs font-semibold text-zinc-700 dark:text-zinc-300"
            >
              Username
            </label>
            {usernameStatus === "checking" ? (
              <span className="text-[11px] font-medium text-zinc-400 flex items-center gap-1">
                <Loader2 size={12} className="animate-spin text-blue-500" />
                Checking...
              </span>
            ) : usernameStatus === "available" ? (
              <span className="text-[11px] font-bold text-emerald-500">✓ Available</span>
            ) : usernameStatus === "unavailable" ? (
              <span className="text-[11px] font-semibold text-amber-500">Unavailable</span>
            ) : usernameStatus === "invalid" ? (
              <span className="text-[11px] font-semibold text-rose-500">Invalid</span>
            ) : (
              <span className="text-[11px] font-medium text-zinc-400">Required</span>
            )}
          </div>
          <div className="relative">
            <input
              type="text"
              id="usernameInput"
              data-testid="username-input"
              placeholder="e.g. rotor_pilot"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="off"
              spellCheck={false}
              aria-invalid={usernameStatus === "invalid" || usernameStatus === "unavailable"}
              aria-describedby="username-feedback"
              className={`w-full px-3.5 py-2.5 rounded-xl border text-xs font-mono text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 dark:placeholder:text-zinc-600 focus:outline-none transition ${
                usernameStatus === "available"
                  ? "border-emerald-500/80 bg-emerald-500/5 focus:ring-2 focus:ring-emerald-500"
                  : usernameStatus === "unavailable"
                    ? "border-amber-500 bg-amber-500/5 focus:ring-2 focus:ring-amber-500"
                    : usernameStatus === "invalid"
                      ? "border-rose-500 bg-rose-500/5 focus:ring-2 focus:ring-rose-500"
                      : "border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 focus:ring-2 focus:ring-blue-500"
              }`}
            />
          </div>
          <p
            id="username-feedback"
            data-testid="username-feedback"
            className={`text-[10px] ${
              usernameStatus === "available"
                ? "text-emerald-600 dark:text-emerald-400 font-medium"
                : usernameStatus === "unavailable"
                  ? "text-amber-600 dark:text-amber-400 font-medium"
                  : usernameStatus === "invalid"
                    ? "text-rose-600 dark:text-rose-400 font-medium"
                    : "text-zinc-500 dark:text-zinc-400"
            }`}
          >
            {usernameMessage ||
              "3–20 characters. Lowercase letters, numbers, and underscores only."}
          </p>
        </div>

        {/* Display Image URL (Optional) */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <label
              htmlFor="avatarUrlInput"
              className="text-xs font-semibold text-zinc-700 dark:text-zinc-300"
            >
              Display Image URL
            </label>
            <span className="text-[10px] text-zinc-400 font-normal">Optional</span>
          </div>
          <input
            type="url"
            id="avatarUrlInput"
            data-testid="avatar-url-input"
            placeholder="https://example.com/avatar.jpg"
            value={avatarUrl}
            onChange={(e) => setAvatarUrl(e.target.value)}
            className="w-full px-3.5 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 text-xs text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 dark:placeholder:text-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500 transition"
          />
          <p className="text-[10px] text-zinc-400 dark:text-zinc-500">
            Provide an image link, or leave blank to display your initial with adaptive category
            palette.
          </p>
        </div>
      </div>

      {/* Setup Error Feedback */}
      {setupError && (
        <div
          data-testid="setup-error-banner"
          className="p-3 rounded-xl text-xs bg-rose-500/10 border border-rose-500/30 text-rose-700 dark:text-rose-400 font-medium flex items-center gap-2"
        >
          <AlertTriangle size={15} className="shrink-0 text-rose-500" />
          <span>{setupError}</span>
        </div>
      )}

      {/* Action Buttons */}
      <div className="space-y-3 pt-1">
        <button
          type="button"
          data-testid="complete-setup-btn"
          disabled={usernameStatus !== "available" || isCompletingSetup}
          onClick={handleCompleteSetup}
          className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-sm flex items-center justify-center gap-2 ${
            usernameStatus !== "available" || isCompletingSetup
              ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
              : "text-white bg-blue-600 hover:bg-blue-500 active:bg-blue-700 cursor-pointer"
          }`}
        >
          {isCompletingSetup ? (
            <>
              <Loader2 size={16} className="animate-spin" />
              <span>Creating account...</span>
            </>
          ) : (
            <span>Complete Setup</span>
          )}
        </button>

        {/* Secondary Action: Use a different email */}
        <div className="text-center pt-1">
          <button
            type="button"
            onClick={handleResetToEmail}
            className="text-[11px] text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 transition cursor-pointer underline-offset-2 hover:underline"
          >
            Use a different email
          </button>
        </div>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen w-full flex flex-col items-center justify-center p-4 bg-zinc-100 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-50 font-sans relative">
      <style>{`
        @keyframes slideInTrack {
          from {
            transform: translateX(0%);
          }
          to {
            transform: translateX(-50%);
          }
        }
      `}</style>

      {/* Theme Switcher Button */}
      <button
        onClick={toggleTheme}
        aria-label="Toggle theme"
        className="absolute top-4 right-4 p-2.5 rounded-full border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 shadow-2xs transition"
      >
        {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
      </button>

      {/* STEP 1: Email Form Modal */}
      {step === "email" && (
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
            <h1 className="text-xl font-bold tracking-tight">Sign in to Quadsmith</h1>
            <p className="text-xs text-zinc-500 dark:text-zinc-400">
              Design, evaluate, and share high-performance FPV drone builds.
            </p>
          </div>

          <div className="space-y-4">
            {renderErrorBanner()}
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
          </div>
        </div>
      )}

      {/* STEP 2: Verify Email Modal */}
      {step === "otp" && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl overflow-hidden relative">
          {renderVerifyEmailScreen(false)}
        </div>
      )}

      {/* STEP 3: Sliding Transition into Account Setup */}
      {step === "setup" && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl overflow-hidden relative">
          <div
            data-testid="sliding-container"
            className="flex w-[200%] items-start"
            style={{
              animation: "slideInTrack 0.4s cubic-bezier(0.25, 1, 0.5, 1) forwards",
            }}
          >
            {/* Screen 1 slides out to the left */}
            <div className="w-1/2 shrink-0">{renderVerifyEmailScreen(true)}</div>
            {/* Screen 2 slides in from the right */}
            <div className="w-1/2 shrink-0">{renderAccountSetupScreen()}</div>
          </div>
        </div>
      )}
    </div>
  );
}
