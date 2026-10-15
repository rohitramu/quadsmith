import { useState, useEffect, useRef } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { ConnectError, Code } from "@connectrpc/connect";
import { Moon, Sun, Loader2, ArrowLeft, Mail, AlertTriangle, Lock, Check } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import logoDark from "../assets/quadsmith-logo-dark.svg";
import logoLight from "../assets/quadsmith-logo-light.svg";
import {
  OtpVerificationErrorSchema,
  LoginChallengeStatus,
} from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import type { GetLoginChallengeDetailsResponse } from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import { UsernameAvailabilityStatus } from "@quadsmith/shared/gen/quadsmith/api/user_pb";
import { getAvatarPalette, getInitial } from "@quadsmith/shared";

export function LoginPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const {
    isAuthenticated,
    sendEmailOtp,
    verifyEmailOtp,
    getLoginChallengeDetails,
    approveLoginChallenge,
    denyLoginChallenge,
    getLoginChallengeStatus,
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
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [remainingAttempts, setRemainingAttempts] = useState<number | null>(null);
  const [isExhausted, setIsExhausted] = useState(false);
  const [isRateLimited, setIsRateLimited] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [cooldown, setCooldown] = useState(0);
  const [exhaustedCooldown, setExhaustedCooldown] = useState(0);
  const [rateLimitCooldown, setRateLimitCooldown] = useState(0);

  // Device B: Magic Link Approval State
  const [challengeDetails, setChallengeDetails] = useState<GetLoginChallengeDetailsResponse | null>(
    null,
  );
  const [approvalState, setApprovalState] = useState<
    "loading" | "prompt" | "approving" | "approved" | "denying" | "denied" | "error"
  >(magicToken ? "loading" : "prompt");
  const [approvalError, setApprovalError] = useState<string | null>(null);

  // Device A: Denial State from Polling
  const [isDenied, setIsDenied] = useState(false);

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
  const emailInputRef = useRef<HTMLInputElement>(null);

  // Cooldown countdown timer
  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setInterval(() => {
      setCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [cooldown]);

  // Exhausted cooldown countdown timer (15s)
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

  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "theme" && (e.newValue === "light" || e.newValue === "dark")) {
        setTheme(e.newValue);
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  // If already logged in, redirect immediately
  useEffect(() => {
    if (isAuthenticated) {
      navigate(redirectUrl, { replace: true });
    }
  }, [isAuthenticated, navigate, redirectUrl]);

  // When sign-in page opens or returns to email step, focus email input
  useEffect(() => {
    if (step === "email" && !magicToken) {
      emailInputRef.current?.focus();
    }
  }, [step, magicToken]);

  // Device B: Load challenge details upon opening magic link
  useEffect(() => {
    const token = magicToken;
    if (!token) return;
    const validToken: string = token;
    let active = true;
    async function loadDetails() {
      try {
        const details = await getLoginChallengeDetails(validToken);
        if (!active) return;
        setChallengeDetails(details);
        if (details.status === LoginChallengeStatus.APPROVED) {
          setApprovalState("approved");
        } else if (details.status === LoginChallengeStatus.DENIED) {
          setApprovalState("denied");
        } else if (details.status === LoginChallengeStatus.EXPIRED) {
          setApprovalState("error");
          setApprovalError(
            "This login challenge is invalid or has expired. Please request a new code.",
          );
        } else {
          setApprovalState("prompt");
        }
      } catch (err: unknown) {
        if (!active) return;
        setApprovalState("error");
        if (err instanceof Error) {
          setApprovalError(err.message);
        } else {
          setApprovalError("Failed to load login request. It may be invalid or expired.");
        }
      }
    }
    loadDetails();
    return () => {
      active = false;
    };
  }, [magicToken, getLoginChallengeDetails]);

  // Device A: Polling for approval status while on OTP step
  useEffect(() => {
    if (
      step !== "otp" ||
      isAuthenticated ||
      magicToken ||
      isDenied ||
      isExhausted ||
      isRateLimited
    ) {
      return;
    }

    const interval = setInterval(async () => {
      try {
        const res = await getLoginChallengeStatus();
        if (res.status === LoginChallengeStatus.APPROVED) {
          if (res.authResponse?.accountSetupRequired && res.authResponse.setupToken) {
            setSetupToken(res.authResponse.setupToken);
            setStep("setup");
          } else {
            setIsSuccess(true);
            navigate(redirectUrl, { replace: true });
          }
        } else if (res.status === LoginChallengeStatus.DENIED) {
          setIsDenied(true);
          setOtpDigits(["", "", "", "", "", ""]);
          setHintDigits(["", "", "", "", "", ""]);
          setErrorMsg("Login request was denied.");
        }
      } catch {
        // Ignore transient polling network errors
      }
    }, 2000);

    return () => clearInterval(interval);
  }, [
    step,
    isAuthenticated,
    magicToken,
    isDenied,
    isExhausted,
    isRateLimited,
    getLoginChallengeStatus,
    navigate,
    redirectUrl,
  ]);

  const handleApprove = async () => {
    if (!magicToken || approvalState !== "prompt") return;
    setApprovalState("approving");
    try {
      await approveLoginChallenge(magicToken);
      setApprovalState("approved");
    } catch (err: unknown) {
      setApprovalState("error");
      if (err instanceof Error) {
        setApprovalError(err.message);
      } else {
        setApprovalError("Failed to approve login. Please try again.");
      }
    }
  };

  const handleDeny = async () => {
    if (!magicToken || approvalState !== "prompt") return;
    setApprovalState("denying");
    try {
      await denyLoginChallenge(magicToken);
      setApprovalState("denied");
    } catch (err: unknown) {
      setApprovalState("error");
      if (err instanceof Error) {
        setApprovalError(err.message);
      } else {
        setApprovalError("Failed to deny login request.");
      }
    }
  };

  // Debounced server-side username availability check
  // The server decides whether the username is available, unavailable, or invalid.
  useEffect(() => {
    const trimmed = username.trim().toLowerCase();
    if (!trimmed) {
      setUsernameStatus("idle");
      setUsernameMessage(null);
      return;
    }

    setUsernameStatus("checking");
    const timeout = setTimeout(async () => {
      try {
        const res = await checkUsernameAvailable(trimmed);
        if (
          res.status === UsernameAvailabilityStatus.AVAILABLE ||
          (res.status === undefined && res.available)
        ) {
          setUsernameStatus("available");
          setUsernameMessage(res.message || "Username is available!");
        } else if (res.status === UsernameAvailabilityStatus.INVALID) {
          setUsernameStatus("invalid");
          setUsernameMessage(res.message || "Username does not meet the naming rules.");
        } else {
          setUsernameStatus("unavailable");
          setUsernameMessage(
            res.message || "This username is already taken. Please choose another.",
          );
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
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(trimmed)) {
      setErrorMsg("Please enter a valid email address");
      return;
    }

    const isResend = step === "otp" && !isExhausted && !isRateLimited && !isDenied;

    setIsLoading(true);
    if (!isResend) {
      setErrorMsg(null);
    }
    try {
      const res = await sendEmailOtp(trimmed);
      setCooldown(res.cooldownSeconds);
      setExhaustedCooldown(0);
      setRateLimitCooldown(0);
      setIsDenied(false);
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
            setExhaustedCooldown(detail.cooldownSeconds || 15);
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
    setIsDenied(false);
    setRemainingAttempts(null);
    setIsExhausted(false);
    setIsRateLimited(false);
    setIsSuccess(false);
    setExhaustedCooldown(0);
    setRateLimitCooldown(0);
    setHintDigits(["", "", "", "", "", ""]);
    setOtpDigits(["", "", "", "", "", ""]);
    setTimeout(() => emailInputRef.current?.focus(), 50);
  };

  const renderErrorBanner = () => {
    if (!errorMsg) return null;
    return (
      <div
        data-testid="otp-error-banner"
        className={`p-3.5 rounded-xl border text-xs flex items-start gap-2.5 ${
          isExhausted || isRateLimited || isDenied
            ? "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
            : remainingAttempts === 2
              ? "border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300"
              : remainingAttempts === 1
                ? "border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-400"
                : "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
        }`}
      >
        {isExhausted || isRateLimited || isDenied ? (
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

      {/* Cross-Device Waiting Pulse Banner */}
      {!isDenied && !isExhausted && !isRateLimited && (
        <div
          data-testid="waiting-for-approval-banner"
          className="p-3.5 rounded-xl border border-blue-500/20 bg-blue-500/10 text-blue-600 dark:text-blue-400 flex items-center gap-3 text-xs text-left"
        >
          <div className="relative flex items-center justify-center w-4 h-4 shrink-0">
            <span className="absolute w-3.5 h-3.5 rounded-full bg-blue-400/40 animate-ping"></span>
            <span className="w-2 h-2 rounded-full bg-blue-500"></span>
          </div>
          <div className="space-y-0.5 flex-1">
            <p className="font-semibold text-zinc-900 dark:text-zinc-100">
              Waiting for magic link approval on your other device...
            </p>
            <p className="text-[11px] text-zinc-600 dark:text-zinc-400">
              Click the magic link in your email on your phone to instantly log in here, or enter
              the 6-digit code below.
            </p>
          </div>
        </div>
      )}

      {/* 6 Digit Inputs */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
            Verification Code
          </label>
          {remainingAttempts !== null &&
            remainingAttempts > 0 &&
            !isExhausted &&
            !isRateLimited &&
            !isDenied && (
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
              disabled={isSuccess || isExhausted || isRateLimited || isDenied || isLoading}
              onChange={(e) => handleOtpChange(index, e.target.value)}
              onKeyDown={(e) => handleKeyDown(index, e)}
              onPaste={handlePaste}
              className={`w-12 h-14 text-center font-mono text-2xl font-bold rounded-xl border bg-zinc-50 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-300 dark:placeholder:text-zinc-700 focus:outline-none transition ${
                isSuccess || isExhausted || isRateLimited || isDenied
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

        {(isExhausted || isRateLimited || isDenied) &&
          (() => {
            const activeCooldown = isRateLimited
              ? rateLimitCooldown
              : isExhausted
                ? exhaustedCooldown
                : cooldown;
            const isReqDisabled = activeCooldown > 0 || isLoading;
            return (
              <button
                type="button"
                onClick={() => handleSendEmail()}
                disabled={isReqDisabled}
                className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-sm flex items-center justify-center gap-2 ${
                  isReqDisabled
                    ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                    : "text-white bg-blue-600 hover:bg-blue-500 active:bg-blue-700 cursor-pointer"
                }`}
              >
                <span>
                  {isRateLimited && rateLimitCooldown > 0
                    ? `Request New Verification Code (${Math.floor(rateLimitCooldown / 60)}m${rateLimitCooldown % 60}s)`
                    : activeCooldown > 0
                      ? `Request New Verification Code (${activeCooldown}s)`
                      : "Request New Verification Code"}
                </span>
              </button>
            );
          })()}

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

  const renderDeviceBApprovalScreen = () => (
    <div className="w-full p-8 space-y-6">
      {/* Header */}
      <div className="text-center space-y-3">
        <div className="flex justify-center">
          <img
            src={theme === "dark" ? logoDark : logoLight}
            alt="Quadsmith"
            className="h-10 w-auto"
          />
        </div>
        <h1 className="text-xl font-bold tracking-tight">Approve Login Request</h1>
        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          A device is requesting to sign in to your Quadsmith account.
        </p>
      </div>

      {approvalState === "loading" && (
        <div
          data-testid="approval-loading"
          className="p-8 flex flex-col items-center justify-center gap-3 text-xs text-zinc-500"
        >
          <Loader2 size={24} className="animate-spin text-blue-500" />
          <span>Loading request details...</span>
        </div>
      )}

      {approvalState === "error" && (
        <div
          data-testid="approval-error-card"
          className="p-4 rounded-xl border border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-400 text-xs space-y-2"
        >
          <div className="flex items-center gap-2 font-semibold">
            <AlertTriangle size={16} />
            <span>Invalid or Expired Link</span>
          </div>
          <p className="text-[11px] opacity-90">
            {approvalError || "This login challenge is invalid or has expired."}
          </p>
        </div>
      )}

      {approvalState === "approved" && (
        <div
          data-testid="approval-success-card"
          className="p-6 rounded-2xl border border-emerald-500/30 bg-emerald-500/10 text-center space-y-3"
        >
          <div className="w-12 h-12 rounded-full bg-emerald-500/20 text-emerald-500 flex items-center justify-center mx-auto">
            <Check size={24} />
          </div>
          <h2 className="text-lg font-bold text-emerald-700 dark:text-emerald-400">
            Login Approved!
          </h2>
          <p className="text-xs text-zinc-600 dark:text-zinc-400">
            Your other device
            {challengeDetails?.deviceHint ? ` (${challengeDetails.deviceHint})` : ""} has been
            logged in.
          </p>
          <div className="pt-2 text-[11px] text-zinc-500 dark:text-zinc-400 font-medium">
            ✓ You can safely close this window now.
          </div>
        </div>
      )}

      {approvalState === "denied" && (
        <div
          data-testid="approval-denied-card"
          className="p-6 rounded-2xl border border-rose-500/30 bg-rose-500/10 text-center space-y-3"
        >
          <div className="w-12 h-12 rounded-full bg-rose-500/20 text-rose-500 flex items-center justify-center mx-auto">
            <Lock size={24} />
          </div>
          <h2 className="text-lg font-bold text-rose-700 dark:text-rose-400">
            Login Request Denied
          </h2>
          <p className="text-xs text-zinc-600 dark:text-zinc-400">
            The sign-in request has been canceled.
          </p>
          <div className="pt-2 text-[11px] text-zinc-500 dark:text-zinc-400 font-medium">
            ✓ You can safely close this window now.
          </div>
        </div>
      )}

      {(approvalState === "prompt" ||
        approvalState === "approving" ||
        approvalState === "denying") &&
        challengeDetails && (
          <div className="space-y-6">
            {/* Metadata Card */}
            <div className="p-4 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/60 space-y-3 text-xs">
              <div className="flex items-center justify-between pb-2 border-b border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500 dark:text-zinc-400">Account</span>
                <span className="font-mono font-medium text-zinc-900 dark:text-zinc-100">
                  {challengeDetails.email}
                </span>
              </div>
              <div className="flex items-center justify-between pb-2 border-b border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500 dark:text-zinc-400">Requesting Device</span>
                <span className="font-medium text-zinc-900 dark:text-zinc-100">
                  {challengeDetails.deviceHint || "Web Browser"}
                </span>
              </div>
              <div className="flex items-center justify-between pb-2 border-b border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500 dark:text-zinc-400">IP & Location</span>
                <span className="font-mono text-[11px] text-zinc-900 dark:text-zinc-100">
                  {challengeDetails.ipAddress}
                  {challengeDetails.locationHint ? ` (${challengeDetails.locationHint})` : ""}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-zinc-500 dark:text-zinc-400">Expires</span>
                <span className="text-zinc-700 dark:text-zinc-300">Within 15 minutes</span>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="space-y-2.5">
              <button
                type="button"
                data-testid="approve-login-btn"
                disabled={approvalState !== "prompt"}
                onClick={handleApprove}
                className="w-full py-3 px-4 rounded-xl font-semibold text-xs text-white bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 disabled:opacity-60 transition shadow-sm flex items-center justify-center gap-2 cursor-pointer"
              >
                {approvalState === "approving" ? (
                  <>
                    <Loader2 size={16} className="animate-spin" />
                    <span>Approving...</span>
                  </>
                ) : (
                  <>
                    <Check size={16} />
                    <span>Approve Login</span>
                  </>
                )}
              </button>

              <button
                type="button"
                data-testid="deny-login-btn"
                disabled={approvalState !== "prompt"}
                onClick={handleDeny}
                className="w-full py-2.5 px-4 rounded-xl font-medium text-xs text-rose-600 dark:text-rose-400 bg-zinc-100 dark:bg-zinc-900 hover:bg-zinc-200 dark:hover:bg-zinc-800 border border-zinc-200 dark:border-zinc-800 disabled:opacity-60 transition flex items-center justify-center gap-1.5 cursor-pointer"
              >
                {approvalState === "denying" ? (
                  <>
                    <Loader2 size={14} className="animate-spin" />
                    <span>Denying...</span>
                  </>
                ) : (
                  <span>Deny Request</span>
                )}
              </button>
            </div>
          </div>
        )}
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

      {/* DEVICE B: Magic Link Approval Screen */}
      {magicToken && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl overflow-hidden relative">
          {renderDeviceBApprovalScreen()}
        </div>
      )}

      {/* DEVICE A: STEP 1: Email Form Modal */}
      {!magicToken && step === "email" && (
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
            <form noValidate onSubmit={handleSendEmail} className="space-y-4">
              <div>
                <label
                  htmlFor="emailInput"
                  className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1.5"
                >
                  Email Address
                </label>
                <div className="relative">
                  <Mail
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                    size={16}
                  />
                  <input
                    id="emailInput"
                    ref={emailInputRef}
                    autoFocus
                    type="email"
                    required
                    placeholder="alex@quadsmith.dev"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full pl-9 pr-3 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 text-sm text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>

              {(() => {
                const isSendDisabled =
                  isLoading ||
                  !email.trim() ||
                  cooldown > 0 ||
                  (isRateLimited && rateLimitCooldown > 0);

                return (
                  <button
                    type="submit"
                    disabled={isSendDisabled}
                    className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-sm flex items-center justify-center gap-2 ${
                      isSendDisabled
                        ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                        : "text-white bg-blue-600 hover:bg-blue-500 active:bg-blue-700 cursor-pointer"
                    }`}
                  >
                    {isLoading ? (
                      <>
                        <Loader2 size={16} className="animate-spin" />
                        <span>Sending code...</span>
                      </>
                    ) : isRateLimited && rateLimitCooldown > 0 ? (
                      <span>
                        Send Verification Code ({Math.floor(rateLimitCooldown / 60)}m
                        {rateLimitCooldown % 60}s)
                      </span>
                    ) : cooldown > 0 ? (
                      <span>Send Verification Code ({cooldown}s)</span>
                    ) : !email.trim() ? (
                      <span>Send Verification Code</span>
                    ) : (
                      <>
                        <span>Send Verification Code</span>
                        <span>&rarr;</span>
                      </>
                    )}
                  </button>
                );
              })()}
            </form>
          </div>
        </div>
      )}

      {/* DEVICE A: STEP 2: Verify Email Modal */}
      {!magicToken && step === "otp" && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl overflow-hidden relative">
          {renderVerifyEmailScreen(false)}
        </div>
      )}

      {/* DEVICE A: STEP 3: Sliding Transition into Account Setup */}
      {!magicToken && step === "setup" && (
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
