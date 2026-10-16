import React, { useState, useRef, useEffect, useCallback } from "react";
import { Mail, Lock, AlertTriangle, Loader2, ArrowRight, ArrowLeft } from "lucide-react";
import { ConnectError, Code } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { useAuth } from "../context/AuthContext";
import {
  EmailOtpPurpose,
  LoginChallengeStatus,
  OtpVerificationErrorSchema,
  VerifyEmailOtpResponseSchema,
  OAuthProvider,
  OAuthPurpose,
} from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import type { VerifyEmailOtpResponse } from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import logoDark from "../assets/quadsmith-logo-dark.svg";
import logoLight from "../assets/quadsmith-logo-light.svg";
import { GoogleIcon, GitHubIcon, DiscordIcon } from "./OAuthIcons";

export interface EmailVerificationFormProps {
  /** The purpose of this OTP flow (LOGIN, ADD_EMAIL, EXPORT_DATA, DELETE_ACCOUNT) */
  purpose: EmailOtpPurpose;
  /** Initial email address */
  email?: string;
  /** Initial challenge ID if already created */
  initialChallengeId?: string | null;
  /** Whether to automatically send the OTP code upon mount (e.g. for GDPR Data Export) */
  autoSend?: boolean;
  /** Whether the user must first enter an email address before sending code (e.g. for Add Email) */
  requireEmailInput?: boolean;
  /** Callback invoked on successful verification */
  onSuccess: (res: VerifyEmailOtpResponse, email?: string) => void;
  /** Optional callback invoked on error */
  onError?: (err: Error) => void;
  /** Optional callback to cancel / close */
  onCancel?: () => void;
  /** Optional callback to return to email input step */
  onBackToEmail?: () => void;
  /** Optional custom title */
  title?: string;
  /** Optional custom OTP step title (defaults to "Verify your email") */
  otpTitle?: string;
  /** Optional custom subtitle */
  subtitle?: string;
  /** Optional custom OTP step subtitle */
  otpSubtitle?: string;
  /** Whether the challenge was externally denied (e.g. from cross-device polling) */
  isDenied?: boolean;
  /** External denial error message */
  denialError?: string | null;
  /** Callback to reset denial state when re-requesting code */
  onResetDenial?: () => void;
  /** Optional custom banner to replace the default dual-path instructions */
  customBanner?: React.ReactNode;
  /** Whether to show the Quadsmith logo in the header (default: true) */
  showLogo?: boolean;
  /** Custom label for email step submit button */
  submitButtonText?: string;
  /** Initial cooldown seconds (e.g. from prior send call) */
  initialCooldown?: number;
  /** Whether to show an explicit Verify button below inputs */
  showVerifyButton?: boolean;
  /** Whether to show an inline resend button at the bottom (default: false) */
  showResendButton?: boolean;
  /** Custom children to render above or below */
  extraContent?: React.ReactNode;
}

export function EmailVerificationForm({
  purpose,
  email: initialEmail = "",
  initialChallengeId = null,
  initialCooldown = 0,
  autoSend = false,
  requireEmailInput = false,
  onSuccess,
  onError,
  onCancel,
  onBackToEmail,
  title,
  otpTitle,
  subtitle,
  otpSubtitle,
  isDenied = false,
  denialError = null,
  onResetDenial,
  customBanner,
  showLogo = true,
  submitButtonText,
  showVerifyButton = false,
  showResendButton = false,
  extraContent,
}: EmailVerificationFormProps) {
  const { sendEmailOtp, verifyEmailOtp, getLoginChallengeStatus, startOAuth } = useAuth();

  const handleOAuthLink = async (provider: OAuthProvider) => {
    try {
      const res = await startOAuth(provider, OAuthPurpose.OAUTH_PURPOSE_ADD_EMAIL, "/account");
      window.location.href = res.authorizationUrl;
    } catch (err: unknown) {
      if (err instanceof Error) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg("Failed to start OAuth linking.");
      }
    }
  };

  const [step, setStep] = useState<"email" | "otp">(requireEmailInput ? "email" : "otp");
  const [email, setEmail] = useState(initialEmail);
  const [challengeId, setChallengeId] = useState<string | null>(initialChallengeId);

  const [isSending, setIsSending] = useState(false);
  const [isVerifying, setIsVerifying] = useState(false);
  const [cooldown, setCooldown] = useState(initialCooldown);
  const [exhaustedCooldown, setExhaustedCooldown] = useState(0);
  const [rateLimitCooldown, setRateLimitCooldown] = useState(0);

  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [remainingAttempts, setRemainingAttempts] = useState<number | null>(null);
  const [isExhausted, setIsExhausted] = useState(false);
  const [isRateLimited, setIsRateLimited] = useState(false);

  const [otpDigits, setOtpDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const [hintDigits, setHintDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

  useEffect(() => {
    if (step === "otp") {
      setTimeout(() => {
        inputRefs.current[0]?.focus();
      }, 50);
    }
  }, [step]);

  const currentTheme =
    (typeof localStorage !== "undefined" && localStorage.getItem("theme")) ||
    (typeof document !== "undefined" && document.documentElement.classList.contains("dark")
      ? "dark"
      : "light") ||
    "dark";

  // Active countdown timer
  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setInterval(() => {
      setCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [cooldown]);

  // Exhaustion timer
  useEffect(() => {
    if (exhaustedCooldown <= 0) return;
    const timer = setInterval(() => {
      setExhaustedCooldown((prev) => {
        if (prev <= 1) {
          setIsExhausted(false);
          setErrorMsg(null);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [exhaustedCooldown]);

  // Rate limit timer
  useEffect(() => {
    if (rateLimitCooldown <= 0) return;
    const timer = setInterval(() => {
      setRateLimitCooldown((prev) => {
        if (prev <= 1) {
          setIsRateLimited(false);
          setErrorMsg(null);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [rateLimitCooldown]);

  // Clear inputs and attempts on denial
  useEffect(() => {
    if (isDenied) {
      setOtpDigits(["", "", "", "", "", ""]);
      setHintDigits(["", "", "", "", "", ""]);
      setRemainingAttempts(null);
    }
  }, [isDenied]);

  // Poll challenge approval status while on OTP step (cross-device approval)
  useEffect(() => {
    if (step !== "otp" || !challengeId || isExhausted || isRateLimited || isVerifying || isDenied) {
      return;
    }

    const interval = setInterval(
      async () => {
        try {
          const res = await getLoginChallengeStatus(challengeId);
          if (res.status === LoginChallengeStatus.APPROVED) {
            clearInterval(interval);
            if (res.verifyResponse) {
              onSuccess(res.verifyResponse, email);
            } else if (res.authResponse) {
              const synthetic = create(VerifyEmailOtpResponseSchema, {
                success: true,
                message: "Login successful",
                purpose: EmailOtpPurpose.LOGIN,
                email,
                accessToken: res.authResponse.accessToken,
                refreshToken: res.authResponse.refreshToken,
                user: res.authResponse.user,
                profile: res.authResponse.profile,
                accountSetupRequired: res.authResponse.accountSetupRequired,
                setupToken: res.authResponse.setupToken,
                accountStatus: res.authResponse.accountStatus,
                activeBan: res.authResponse.activeBan,
                scheduledPurgeAt: res.authResponse.scheduledPurgeAt,
              });
              onSuccess(synthetic, email);
            }
          } else if (res.status === LoginChallengeStatus.DENIED) {
            clearInterval(interval);
            setErrorMsg(denialError || "Verification request was denied on your other device.");
          }
        } catch {
          // Ignore transient polling network errors
        }
      },
      (typeof window !== "undefined" && (window as any).__VITEST__) ||
        import.meta.env.MODE === "test"
        ? 50
        : 2000,
    );

    return () => clearInterval(interval);
  }, [
    step,
    challengeId,
    isExhausted,
    isRateLimited,
    isVerifying,
    isDenied,
    denialError,
    getLoginChallengeStatus,
    onSuccess,
    email,
  ]);

  // Send OTP
  const handleSendOtp = useCallback(
    async (targetEmail?: string) => {
      const emailToSend = (targetEmail !== undefined ? targetEmail : email).trim();
      if (purpose === EmailOtpPurpose.ADD_EMAIL || purpose === EmailOtpPurpose.LOGIN) {
        if (!emailToSend) {
          setErrorMsg("Please enter your email address");
          return;
        }
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(emailToSend)) {
          setErrorMsg("Please enter a valid email address");
          return;
        }
      }

      setIsSending(true);
      setErrorMsg(null);

      try {
        const res = await sendEmailOtp(emailToSend || undefined, purpose);
        if (res.challengeId) {
          setChallengeId(res.challengeId);
        }
        if (res.email) {
          setEmail(res.email);
        }
        setOtpDigits(["", "", "", "", "", ""]);
        setHintDigits(["", "", "", "", "", ""]);
        setCooldown(res.cooldownSeconds ?? 20);
        setExhaustedCooldown(0);
        setRateLimitCooldown(0);
        setIsExhausted(false);
        setIsRateLimited(false);
        setRemainingAttempts(null);
        setStep("otp");
        setIsSending(false);
        if (onResetDenial) {
          onResetDenial();
        }
        setTimeout(() => inputRefs.current[0]?.focus(), 100);
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
        if (onError && err instanceof Error) {
          onError(err);
        }
      } finally {
        setIsSending(false);
      }
    },
    [email, purpose, sendEmailOtp, onError, onResetDenial],
  );

  // Auto-send on mount if requested
  const hasAutoSent = useRef(false);
  useEffect(() => {
    if (autoSend && !hasAutoSent.current) {
      hasAutoSent.current = true;
      handleSendOtp();
    }
  }, [autoSend, handleSendOtp]);

  // Verify OTP
  const handleVerifyOtp = useCallback(
    async (explicitCode?: string) => {
      const code = explicitCode || otpDigits.join("");
      if (code.length < 6) {
        setErrorMsg("Please enter all 6 digits of the verification code");
        return;
      }
      if (isVerifying) return;

      setIsVerifying(true);
      setErrorMsg(null);

      try {
        const res = await verifyEmailOtp({
          email: email || undefined,
          code,
          challengeId,
          purpose,
        });
        onSuccess(res, email);
      } catch (err: unknown) {
        // Clear inputs and show digits as faded placeholder hints
        setHintDigits(code.split(""));
        setOtpDigits(["", "", "", "", "", ""]);
        setTimeout(() => inputRefs.current[0]?.focus(), 10);

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
          } else {
            const detail = err.findDetails(OtpVerificationErrorSchema)[0];
            if (detail) {
              setErrorMsg(detail.message || err.rawMessage);
              setRemainingAttempts(detail.remainingAttempts);
              if (detail.isExhausted) {
                setIsExhausted(true);
                setExhaustedCooldown(detail.cooldownSeconds || 15);
              }
              if (detail.cooldownSeconds > 0 && !detail.isExhausted) {
                setRateLimitCooldown(detail.cooldownSeconds);
                setIsRateLimited(true);
              }
            } else {
              setErrorMsg(err.rawMessage);
            }
          }
        } else if (err instanceof Error) {
          setErrorMsg(err.message);
        } else {
          setErrorMsg("Verification failed");
        }
        if (onError && err instanceof Error) {
          onError(err);
        }
      } finally {
        setIsVerifying(false);
      }
    },
    [challengeId, email, isVerifying, onError, onSuccess, otpDigits, purpose, verifyEmailOtp],
  );

  const handleOtpChange = (index: number, val: string) => {
    const clean = val.replace(/\D/g, "");
    if (!clean && val !== "") return;

    const newDigits = [...otpDigits];

    if (clean.length > 1) {
      // Pasted multi-character string
      const chars = clean.slice(0, 6).split("");
      chars.forEach((c, i) => {
        newDigits[i] = c;
      });
      setOtpDigits(newDigits);
      const nextIndex = Math.min(chars.length, 5);
      inputRefs.current[nextIndex]?.focus();
      const code = newDigits.join("");
      if (code.length === 6 && !isVerifying) {
        handleVerifyOtp(code);
      }
      return;
    }

    newDigits[index] = clean;
    setOtpDigits(newDigits);

    // Auto-advance
    if (clean && index < 5) {
      inputRefs.current[index + 1]?.focus();
    }

    const code = newDigits.join("");
    if (code.length === 6 && !isVerifying) {
      handleVerifyOtp(code);
    }
  };

  const handleKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Backspace" && !otpDigits[index] && index > 0) {
      inputRefs.current[index - 1]?.focus();
    } else if (e.key === "ArrowLeft" && index > 0) {
      inputRefs.current[index - 1]?.focus();
    } else if (e.key === "ArrowRight" && index < 5) {
      inputRefs.current[index + 1]?.focus();
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
    inputRefs.current[nextIndex]?.focus();
    if (pasted.length === 6 && !isVerifying) {
      handleVerifyOtp(pasted);
    }
  };

  // Render Step 1: Email Input
  if (step === "email") {
    const isSendDisabled =
      isSending || !email.trim() || cooldown > 0 || (isRateLimited && rateLimitCooldown > 0);

    return (
      <div className="space-y-6">
        {showLogo && (
          <div className="text-center space-y-3">
            <div className="flex justify-center">
              <img
                src={currentTheme === "dark" ? logoDark : logoLight}
                alt="Quadsmith"
                className="h-10 w-auto"
              />
            </div>
            <h1 className="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
              {title || "Add Email Address"}
            </h1>
            <p className="text-xs text-zinc-500 dark:text-zinc-400">
              {subtitle || "Associate a new email address with your Quadsmith account."}
            </p>
          </div>
        )}

        {errorMsg && (
          <div
            data-testid="otp-error-banner"
            className="p-3.5 rounded-xl border border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400 text-xs flex items-start gap-2.5"
          >
            {isRateLimited ? (
              <Lock size={16} className="text-red-500 shrink-0 mt-0.5" />
            ) : (
              <AlertTriangle size={16} className="text-red-500 shrink-0 mt-0.5" />
            )}
            <div className="space-y-0.5 flex-1">
              <p className="font-semibold">{errorMsg}</p>
            </div>
          </div>
        )}

        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleSendOtp();
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">
              Email Address
            </label>
            <div className="relative">
              <Mail className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" size={16} />
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
            disabled={isSendDisabled}
            className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-sm flex items-center justify-center gap-2 ${
              isSendDisabled
                ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                : "text-white bg-blue-600 hover:bg-blue-500 active:bg-blue-700 cursor-pointer"
            }`}
          >
            {isSending ? (
              <>
                <Loader2 size={16} className="animate-spin" />
                <span>Sending code...</span>
              </>
            ) : isRateLimited && rateLimitCooldown > 0 ? (
              <span>
                Send Verification Code ({Math.floor(rateLimitCooldown / 60)}m{" "}
                {rateLimitCooldown % 60}s)
              </span>
            ) : cooldown > 0 ? (
              <span>Send Verification Code ({cooldown}s)</span>
            ) : (
              <>
                <span>{submitButtonText || "Send Verification Code"}</span>
                <ArrowRight size={14} />
              </>
            )}
          </button>
        </form>

        {purpose === EmailOtpPurpose.ADD_EMAIL && (
          <div className="space-y-4 pt-1">
            <div className="relative flex items-center py-1">
              <div className="flex-grow border-t border-zinc-200 dark:border-zinc-800"></div>
              <span className="flex-shrink mx-3 text-[10px] font-semibold uppercase tracking-wider text-zinc-400 dark:text-zinc-500">
                Or link via OAuth
              </span>
              <div className="flex-grow border-t border-zinc-200 dark:border-zinc-800"></div>
            </div>

            <div className="grid grid-cols-3 gap-2">
              <button
                type="button"
                onClick={() => handleOAuthLink(OAuthProvider.OAUTH_PROVIDER_GOOGLE)}
                className="py-2 px-3 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700 text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer text-zinc-800 dark:text-zinc-200 shadow-2xs transition"
              >
                <GoogleIcon className="w-3.5 h-3.5 shrink-0" />
                <span>Google</span>
              </button>
              <button
                type="button"
                onClick={() => handleOAuthLink(OAuthProvider.OAUTH_PROVIDER_GITHUB)}
                className="py-2 px-3 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700 text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer text-zinc-800 dark:text-zinc-200 shadow-2xs transition"
              >
                <GitHubIcon className="w-3.5 h-3.5 shrink-0 fill-current" />
                <span>GitHub</span>
              </button>
              <button
                type="button"
                onClick={() => handleOAuthLink(OAuthProvider.OAUTH_PROVIDER_DISCORD)}
                className="py-2 px-3 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700 text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer text-zinc-800 dark:text-zinc-200 shadow-2xs transition"
              >
                <DiscordIcon className="w-3.5 h-3.5 shrink-0" />
                <span>Discord</span>
              </button>
            </div>
          </div>
        )}

        {extraContent}
      </div>
    );
  }

  // Initial sending state for auto-send (e.g. Download My Data)
  if (autoSend && isSending && !challengeId && !errorMsg) {
    return (
      <div className="text-center py-8 space-y-4">
        <Loader2 size={32} className="animate-spin text-blue-500 mx-auto" />
        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          Sending verification challenge to {email || "your primary email"}...
        </p>
      </div>
    );
  }

  // Step 2: OTP Entry Screen
  return (
    <div className="space-y-6">
      {/* Header */}
      {showLogo && (
        <div className="text-center space-y-3">
          <div className="flex justify-center">
            <img
              src={currentTheme === "dark" ? logoDark : logoLight}
              alt="Quadsmith"
              className="h-10 w-auto"
            />
          </div>
          <h1 className="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
            {otpTitle || "Verify your email"}
          </h1>
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            {otpSubtitle ||
              (requireEmailInput
                ? `We sent a single-use verification code to ${email || "your email"}`
                : subtitle || `We sent a single-use verification code to ${email || "your email"}`)}
          </p>
        </div>
      )}

      {/* Instructions / Banner */}
      {customBanner !== undefined ? (
        customBanner
      ) : (
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
                inputRefs.current[index] = el;
              }}
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              autoComplete={index === 0 ? "one-time-code" : "off"}
              maxLength={1}
              placeholder={isDenied ? "" : hintDigits[index] || ""}
              value={digit}
              disabled={isExhausted || isRateLimited || isDenied || isVerifying}
              onChange={(e) => handleOtpChange(index, e.target.value)}
              onKeyDown={(e) => handleKeyDown(index, e)}
              onPaste={handlePaste}
              className={`w-12 h-14 text-center font-mono text-2xl font-bold rounded-xl border bg-zinc-50 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-300 dark:placeholder:text-zinc-700 focus:outline-none transition ${
                isExhausted || isRateLimited || isDenied
                  ? "opacity-40 border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 text-zinc-400 dark:text-zinc-600 cursor-not-allowed"
                  : "border-zinc-300 dark:border-zinc-700 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              }`}
            />
          ))}
        </div>
      </div>

      {/* Error / Attempt Warning Banner */}
      {(isDenied ? denialError || "Login request was denied." : errorMsg) && (
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
            <p className="font-semibold">
              {isDenied ? denialError || "Login request was denied." : errorMsg}
            </p>
            {remainingAttempts !== null && remainingAttempts > 0 && !isDenied && (
              <p className="text-[11px] opacity-90">
                {remainingAttempts === 1
                  ? "Final attempt remaining before this code is invalidated."
                  : `${remainingAttempts} attempts remaining for this verification challenge.`}
              </p>
            )}
          </div>
        </div>
      )}

      {/* Action Buttons & Status */}
      <div className="space-y-2.5">
        {isVerifying && (
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
            const isReqDisabled = activeCooldown > 0 || isSending;
            return (
              <button
                type="button"
                onClick={() => handleSendOtp()}
                disabled={isReqDisabled}
                className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-sm flex items-center justify-center gap-2 ${
                  isReqDisabled
                    ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                    : "text-white bg-blue-600 hover:bg-blue-500 active:bg-blue-700 cursor-pointer"
                }`}
              >
                {isSending ? (
                  <>
                    <Loader2 size={16} className="animate-spin" />
                    <span>Requesting new code...</span>
                  </>
                ) : isRateLimited && rateLimitCooldown > 0 ? (
                  <span>
                    Request New Verification Code ({Math.floor(rateLimitCooldown / 60)}m
                    {rateLimitCooldown % 60}s)
                  </span>
                ) : activeCooldown > 0 ? (
                  <span>Request New Verification Code ({activeCooldown}s)</span>
                ) : (
                  <span>Request New Verification Code</span>
                )}
              </button>
            );
          })()}

        {showVerifyButton && (
          <button
            type="button"
            disabled={
              isVerifying ||
              isExhausted ||
              isRateLimited ||
              isDenied ||
              otpDigits.join("").length !== 6
            }
            onClick={() => handleVerifyOtp()}
            className="w-full py-2.5 rounded-xl font-semibold text-xs text-white bg-blue-600 hover:bg-blue-500 active:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer transition shadow-sm flex items-center justify-center gap-2"
          >
            {isVerifying ? (
              <>
                <Loader2 size={16} className="animate-spin" />
                <span>Verifying...</span>
              </>
            ) : (
              <span>Verify & Continue</span>
            )}
          </button>
        )}

        <div className="flex items-center justify-between pt-2">
          {requireEmailInput || onBackToEmail ? (
            <button
              type="button"
              onClick={() => {
                setErrorMsg(null);
                setRemainingAttempts(null);
                setIsExhausted(false);
                setIsRateLimited(false);
                if (onBackToEmail) {
                  onBackToEmail();
                } else {
                  setStep("email");
                }
              }}
              className="text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 flex items-center gap-1 text-xs transition cursor-pointer"
            >
              <ArrowLeft size={13} />
              <span>Back</span>
            </button>
          ) : onCancel ? (
            <button
              type="button"
              onClick={onCancel}
              className="px-4 py-2 rounded-xl text-xs font-medium text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 transition cursor-pointer"
            >
              Cancel
            </button>
          ) : (
            <div />
          )}

          {showResendButton && !isExhausted && !isRateLimited && !isDenied && (
            <div className="text-right">
              {cooldown > 0 ? (
                <span className="text-xs text-zinc-400 dark:text-zinc-500">
                  Resend code in {cooldown}s
                </span>
              ) : (
                <button
                  type="button"
                  onClick={() => handleSendOtp()}
                  disabled={isSending}
                  className="text-xs font-semibold text-blue-600 dark:text-blue-400 hover:underline cursor-pointer disabled:opacity-50"
                >
                  {isSending ? "Sending..." : "Resend code"}
                </button>
              )}
            </div>
          )}
        </div>
      </div>

      {extraContent}
    </div>
  );
}
