import { useState, useEffect, useRef } from "react";
import { Link, useSearchParams, useNavigate } from "react-router-dom";
import { ConnectError, Code } from "@connectrpc/connect";
import { Loader2, Mail, AlertTriangle, Lock, Check } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import logo from "../assets/quadsmith-logo.svg";
import {
  OtpVerificationErrorSchema,
  LoginChallengeStatus,
  EmailOtpPurpose,
  OAuthProvider,
  OAuthPurpose,
} from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import { UsernameAvailabilityStatus } from "@quadsmith/shared/gen/quadsmith/api/user_pb";
import { getAvatarPalette, getInitial } from "@quadsmith/shared";
import { EmailVerificationForm } from "../components/EmailVerificationForm";
import { ChallengeApprovalScreen } from "../components/ChallengeApprovalScreen";
import { GoogleIcon, GitHubIcon, DiscordIcon } from "../components/OAuthIcons";
import { SafetyDisclaimersButton } from "../components/SafetyDisclaimersModal";

export function LoginPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const {
    isAuthenticated,
    sendEmailOtp,
    getLoginChallengeStatus,
    completeAccountSetup,
    checkUsernameAvailable,
    startOAuth,
    completeOAuth,
  } = useAuth();

  const redirectUrl = searchParams.get("redirect") || "/";
  const magicToken = searchParams.get("token");
  const oauthCode = searchParams.get("code");
  const oauthState = searchParams.get("state");

  const [step, setStep] = useState<"email" | "otp" | "setup">(magicToken ? "otp" : "email");
  const [isOAuthProcessing, setIsOAuthProcessing] = useState(Boolean(oauthCode && oauthState));
  const [isOAuthLoading, setIsOAuthLoading] = useState(false);
  const [cliLoginApproved, setCliLoginApproved] = useState(false);
  const [isCliOAuthFlow, setIsCliOAuthFlow] = useState(false);
  const [email, setEmail] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [remainingAttempts, setRemainingAttempts] = useState<number | null>(null);
  const [isExhausted, setIsExhausted] = useState(false);
  const [isRateLimited, setIsRateLimited] = useState(false);
  const [cooldown, setCooldown] = useState(0);
  const [exhaustedCooldown, setExhaustedCooldown] = useState(0);
  const [rateLimitCooldown, setRateLimitCooldown] = useState(0);

  // Device A: Denial State from Polling
  const [isDenied, setIsDenied] = useState(false);

  // Account Setup State
  const [setupToken, setSetupToken] = useState("");
  const [username, setUsername] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [avatarLoadError, setAvatarLoadError] = useState(false);
  const [usernameStatus, setUsernameStatus] = useState<
    "idle" | "checking" | "available" | "unavailable" | "invalid"
  >("idle");
  const [usernameMessage, setUsernameMessage] = useState<string | null>(null);
  const [isCompletingSetup, setIsCompletingSetup] = useState(false);
  const [setupError, setSetupError] = useState<string | null>(null);

  const otpInputRefs = useRef<(HTMLInputElement | null)[]>([]);
  const emailInputRef = useRef<HTMLInputElement>(null);
  const processedOAuthRef = useRef<string | null>(null);

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

  // If already logged in, redirect immediately (unless viewing an approval magic link or completing OAuth)
  useEffect(() => {
    if (
      isAuthenticated &&
      !magicToken &&
      !oauthCode &&
      !oauthState &&
      !isOAuthProcessing &&
      !cliLoginApproved &&
      !isCliOAuthFlow
    ) {
      navigate(redirectUrl, { replace: true });
    }
  }, [
    isAuthenticated,
    magicToken,
    oauthCode,
    oauthState,
    isOAuthProcessing,
    cliLoginApproved,
    isCliOAuthFlow,
    navigate,
    redirectUrl,
  ]);

  // When sign-in page opens or returns to email step, focus email input
  useEffect(() => {
    if (step === "email" && !magicToken) {
      emailInputRef.current?.focus();
    }
  }, [step, magicToken]);

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
            navigate(redirectUrl, { replace: true });
          }
        } else if (res.status === LoginChallengeStatus.DENIED) {
          setIsDenied(true);
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

  // OAuth Callback Handler
  useEffect(() => {
    if (!oauthCode || !oauthState) return;
    const oauthKey = `${oauthCode}:${oauthState}`;
    if (processedOAuthRef.current === oauthKey) return;
    processedOAuthRef.current = oauthKey;

    const clearOAuthParams = () => {
      const next = new URLSearchParams(searchParams);
      next.delete("code");
      next.delete("state");
      setSearchParams(next, { replace: true });
    };

    const processOAuth = async () => {
      setIsOAuthProcessing(true);
      setIsOAuthLoading(true);
      setErrorMsg(null);
      try {
        const res = await completeOAuth(oauthCode, oauthState);

        if (res.purpose === OAuthPurpose.OAUTH_PURPOSE_CLI_LOGIN) {
          setIsCliOAuthFlow(true);
        }

        const setupRequired = res.authResponse?.accountSetupRequired;
        const sToken = res.authResponse?.setupToken;
        if (setupRequired && sToken) {
          setSetupToken(sToken);
          if (res.primaryEmail) {
            setEmail(res.primaryEmail);
          }
          if (res.suggestedAvatarUrl) {
            setAvatarLoadError(false);
            setAvatarUrl(res.suggestedAvatarUrl);
          }
          setStep("setup");
          setIsOAuthProcessing(false);
          clearOAuthParams();
        } else if (res.purpose === OAuthPurpose.OAUTH_PURPOSE_CLI_LOGIN) {
          setCliLoginApproved(true);
          setIsOAuthProcessing(false);
          clearOAuthParams();
        } else if (res.mergeRequired && res.mergeTicket) {
          sessionStorage.setItem(
            "qs_pending_oauth_merge",
            JSON.stringify({
              email: res.primaryEmail || res.otherEmails?.[0] || "",
              otherUsername: res.otherUserUsername || "user",
              otherBuildCount: res.otherBuildCount || 0,
              otherBuildNames: res.otherBuildNames || [],
              otherEmails: res.otherEmails || [],
              otherOauthProviders: res.otherOauthProviders || [],
              mergeTicket: res.mergeTicket,
            }),
          );
          navigate("/account", { replace: true });
        } else {
          const postLoginRedirect = sessionStorage.getItem("qs_oauth_redirect") || redirectUrl;
          sessionStorage.removeItem("qs_oauth_redirect");
          navigate(postLoginRedirect, { replace: true });
        }
      } catch (err: unknown) {
        setIsOAuthProcessing(false);
        clearOAuthParams();
        if (err instanceof ConnectError) {
          setErrorMsg(err.rawMessage);
        } else if (err instanceof Error) {
          setErrorMsg(err.message);
        } else {
          setErrorMsg("Failed to complete OAuth authentication. Please try again.");
        }
      } finally {
        setIsOAuthLoading(false);
      }
    };

    processOAuth();
  }, [oauthCode, oauthState, completeOAuth, navigate, redirectUrl, searchParams, setSearchParams]);

  const handleOAuthLogin = async (provider: OAuthProvider) => {
    setIsOAuthLoading(true);
    setErrorMsg(null);
    try {
      if (redirectUrl && redirectUrl !== "/") {
        sessionStorage.setItem("qs_oauth_redirect", redirectUrl);
      } else {
        sessionStorage.removeItem("qs_oauth_redirect");
      }
      const res = await startOAuth(provider, OAuthPurpose.OAUTH_PURPOSE_LOGIN, "/login");
      window.location.href = res.authorizationUrl;
    } catch (err: unknown) {
      if (err instanceof ConnectError) {
        setErrorMsg(err.rawMessage);
      } else if (err instanceof Error) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg("Failed to start OAuth login. Please try again.");
      }
      setIsOAuthLoading(false);
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
      if (isCliOAuthFlow || searchParams.get("challenge_id")) {
        setCliLoginApproved(true);
      } else {
        const postLoginRedirect = sessionStorage.getItem("qs_oauth_redirect") || redirectUrl;
        sessionStorage.removeItem("qs_oauth_redirect");
        navigate(postLoginRedirect, { replace: true });
      }
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
    setExhaustedCooldown(0);
    setRateLimitCooldown(0);
    setTimeout(() => emailInputRef.current?.focus(), 50);
  };

  const renderErrorBanner = () => {
    if (!errorMsg) return null;
    return (
      <div
        data-testid="otp-error-banner"
        className={`p-3.5 rounded-xl border text-xs flex items-start gap-2.5 ${
          isExhausted || isRateLimited || isDenied
            ? "border-red-500/30 bg-red-500/10 text-red-400"
            : remainingAttempts === 2
              ? "border-amber-500/30 bg-amber-500/10 text-amber-300"
              : remainingAttempts === 1
                ? "border-rose-500/30 bg-rose-500/10 text-rose-400"
                : "border-red-500/30 bg-red-500/10 text-red-400"
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
      <EmailVerificationForm
        purpose={EmailOtpPurpose.LOGIN}
        email={email}
        initialCooldown={cooldown}
        subtitle={`We sent a single-use login code to ${email}`}
        isDenied={isDenied}
        denialError={isDenied ? "Login request was denied." : null}
        onResetDenial={() => {
          setIsDenied(false);
          setErrorMsg(null);
        }}
        onSuccess={(res) => {
          if (res.accountSetupRequired && res.setupToken) {
            setSetupToken(res.setupToken);
            setStep("setup");
          } else {
            navigate(redirectUrl, { replace: true });
          }
        }}
        onBackToEmail={handleResetToEmail}
        customBanner={
          !isDenied ? (
            <div
              data-testid="waiting-for-approval-banner"
              className="p-3.5 rounded-xl border border-blue-500/20 bg-blue-500/10 text-blue-400 flex items-center gap-3 text-xs text-left"
            >
              <div className="relative flex items-center justify-center w-4 h-4 shrink-0">
                <span className="absolute w-3.5 h-3.5 rounded-full bg-blue-400/40 animate-ping"></span>
                <span className="w-2 h-2 rounded-full bg-blue-500"></span>
              </div>
              <div className="space-y-0.5 flex-1">
                <p className="font-semibold text-zinc-100">
                  Waiting for magic link approval on your other device...
                </p>
                <p className="text-[11px] text-zinc-400">
                  Click the magic link in your email on your phone to instantly log in here, or
                  enter the 6-digit code below.
                </p>
              </div>
            </div>
          ) : undefined
        }
      />
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
              avatarUrl && !avatarLoadError
                ? "border-zinc-800"
                : getAvatarPalette(username.trim().toLowerCase() || email).badgeClass
            }`}
          >
            {avatarUrl && !avatarLoadError ? (
              <img
                src={avatarUrl}
                alt="Avatar Preview"
                className="w-full h-full object-cover"
                onError={() => {
                  setAvatarLoadError(true);
                }}
              />
            ) : (
              <span>{getInitial(username || email)}</span>
            )}
          </div>
        </div>
        <h1 className="text-xl font-bold tracking-tight">Account Setup</h1>
        <p className="text-xs text-zinc-400">
          Your username is your public display identifier across Quadsmith.
        </p>
      </div>

      {/* Form Fields */}
      <div className="space-y-4">
        {/* Username Input */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <label htmlFor="usernameInput" className="text-xs font-semibold text-zinc-300">
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
              className={`w-full px-3.5 py-2.5 rounded-xl border text-xs font-mono text-zinc-100 placeholder:text-zinc-600 focus:outline-none transition ${
                usernameStatus === "available"
                  ? "border-emerald-500/80 bg-emerald-500/5 focus:ring-2 focus:ring-emerald-500"
                  : usernameStatus === "unavailable"
                    ? "border-amber-500 bg-amber-500/5 focus:ring-2 focus:ring-amber-500"
                    : usernameStatus === "invalid"
                      ? "border-rose-500 bg-rose-500/5 focus:ring-2 focus:ring-rose-500"
                      : "border-zinc-700 bg-zinc-950 focus:ring-2 focus:ring-blue-500"
              }`}
            />
          </div>
          <p
            id="username-feedback"
            data-testid="username-feedback"
            className={`text-[10px] ${
              usernameStatus === "available"
                ? "text-emerald-400 font-medium"
                : usernameStatus === "unavailable"
                  ? "text-amber-400 font-medium"
                  : usernameStatus === "invalid"
                    ? "text-rose-400 font-medium"
                    : "text-zinc-400"
            }`}
          >
            {usernameMessage ||
              "3–20 characters. Lowercase letters, numbers, and underscores only."}
          </p>
        </div>

        {/* Display Image URL (Optional) */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <label htmlFor="avatarUrlInput" className="text-xs font-semibold text-zinc-300">
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
            onChange={(e) => {
              setAvatarLoadError(false);
              setAvatarUrl(e.target.value);
            }}
            className="w-full px-3.5 py-2.5 rounded-xl border border-zinc-700 bg-zinc-950 text-xs text-zinc-100 placeholder:text-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500 transition"
          />
          <p className="text-[10px] text-zinc-500">
            Provide an image link, or leave blank to display your initial with adaptive category
            palette.
          </p>
        </div>
      </div>

      {/* Setup Error Feedback */}
      {setupError && (
        <div
          data-testid="setup-error-banner"
          className="p-3 rounded-xl text-xs bg-rose-500/10 border border-rose-500/30 text-rose-400 font-medium flex items-center gap-2"
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
              ? "text-zinc-500 bg-zinc-800 border border-zinc-700 cursor-not-allowed opacity-60"
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
            className="text-[11px] text-zinc-400 hover:text-zinc-300 transition cursor-pointer underline-offset-2 hover:underline"
          >
            Use a different email
          </button>
        </div>

        <p
          data-testid="login-legal-notice"
          className="text-[11px] text-zinc-500 text-center pt-2 border-t border-zinc-800/80 leading-relaxed"
        >
          By completing account setup, you agree to our{" "}
          <Link
            to="/terms"
            className="text-zinc-400 hover:text-zinc-200 underline underline-offset-2 transition-colors"
          >
            Terms of Service
          </Link>{" "}
          and acknowledge our{" "}
          <SafetyDisclaimersButton className="text-zinc-400 hover:text-zinc-200 underline underline-offset-2 transition-colors cursor-pointer">
            Safety Disclaimers
          </SafetyDisclaimersButton>
          .
        </p>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen w-full flex flex-col items-center justify-center p-4 bg-zinc-950 text-zinc-50 font-sans relative">
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

      {/* CLI Login Approved Screen */}
      {cliLoginApproved && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-800 bg-zinc-900 shadow-xl p-8 space-y-6 text-center">
          <div className="w-12 h-12 rounded-full bg-emerald-500/10 text-emerald-500 flex items-center justify-center mx-auto border border-emerald-500/20">
            <Check size={24} />
          </div>
          <div className="space-y-2">
            <h1 className="text-xl font-bold tracking-tight">CLI Login Approved</h1>
            <p className="text-xs text-zinc-400">
              You are now signed in to Quadsmith CLI on your terminal. You can safely close this
              window.
            </p>
          </div>
        </div>
      )}

      {/* OAuth In-Progress Screen */}
      {isOAuthProcessing && !cliLoginApproved && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-800 bg-zinc-900 shadow-xl p-8 space-y-6 text-center">
          <div className="flex justify-center">
            <Loader2 size={32} className="animate-spin text-blue-500" />
          </div>
          <div className="space-y-1">
            <h2 className="text-base font-semibold">Completing authentication...</h2>
            <p className="text-xs text-zinc-400">Exchanging credentials with provider...</p>
          </div>
        </div>
      )}

      {/* DEVICE B: Magic Link Approval Screen */}
      {magicToken && !cliLoginApproved && !isOAuthProcessing && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-800 bg-zinc-900 shadow-xl overflow-hidden relative">
          <ChallengeApprovalScreen token={magicToken} />
        </div>
      )}

      {/* DEVICE A: STEP 1: Email Form Modal */}
      {!magicToken && !cliLoginApproved && !isOAuthProcessing && step === "email" && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-800 bg-zinc-900 shadow-xl p-8 space-y-6">
          {/* Header Logo */}
          <div className="text-center space-y-3">
            <div className="flex justify-center">
              <img src={logo} alt="Quadsmith" className="h-10 w-auto" />
            </div>
            <h1 className="text-xl font-bold tracking-tight">Sign in to Quadsmith</h1>
            <p className="text-xs text-zinc-400">
              Design, evaluate, and share high-performance FPV drone builds.
            </p>
          </div>

          <div className="space-y-4">
            {renderErrorBanner()}
            <form noValidate onSubmit={handleSendEmail} className="space-y-4">
              <div>
                <label
                  htmlFor="emailInput"
                  className="block text-xs font-medium text-zinc-300 mb-1.5"
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
                    className="w-full pl-9 pr-3 py-2.5 rounded-xl border border-zinc-700 bg-zinc-950 text-sm text-zinc-100 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
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
                        ? "text-zinc-500 bg-zinc-800 border border-zinc-700 cursor-not-allowed opacity-60"
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

            {/* OAuth Continue With Buttons */}
            <div className="relative flex items-center py-1">
              <div className="flex-grow border-t border-zinc-800"></div>
              <span className="flex-shrink mx-3 text-[10px] font-semibold uppercase tracking-wider text-zinc-500">
                Or continue with
              </span>
              <div className="flex-grow border-t border-zinc-800"></div>
            </div>

            <div className="space-y-2.5">
              <button
                type="button"
                disabled={isOAuthLoading}
                onClick={() => handleOAuthLogin(OAuthProvider.OAUTH_PROVIDER_GOOGLE)}
                className="w-full py-2.5 px-4 rounded-xl border border-zinc-700 bg-zinc-800/90 hover:bg-zinc-800 text-zinc-100 font-semibold text-xs transition flex items-center justify-center gap-2.5 shadow-2xs cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <GoogleIcon className="w-4 h-4 shrink-0" />
                <span>Continue with Google</span>
              </button>

              <button
                type="button"
                disabled={isOAuthLoading}
                onClick={() => handleOAuthLogin(OAuthProvider.OAUTH_PROVIDER_GITHUB)}
                className="w-full py-2.5 px-4 rounded-xl border border-zinc-700 bg-zinc-800/90 hover:bg-zinc-800 text-zinc-100 font-semibold text-xs transition flex items-center justify-center gap-2.5 shadow-2xs cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <GitHubIcon className="w-4 h-4 shrink-0 fill-current" />
                <span>Continue with GitHub</span>
              </button>

              <button
                type="button"
                disabled={isOAuthLoading}
                onClick={() => handleOAuthLogin(OAuthProvider.OAUTH_PROVIDER_DISCORD)}
                className="w-full py-2.5 px-4 rounded-xl border border-zinc-700 bg-zinc-800/90 hover:bg-zinc-800 text-zinc-100 font-semibold text-xs transition flex items-center justify-center gap-2.5 shadow-2xs cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <DiscordIcon className="w-4 h-4 shrink-0" />
                <span>Continue with Discord</span>
              </button>
            </div>

            <p
              data-testid="login-legal-notice"
              className="text-[11px] text-zinc-500 text-center pt-3 border-t border-zinc-800/80 leading-relaxed"
            >
              By continuing, you agree to our{" "}
              <Link
                to="/terms"
                className="text-zinc-400 hover:text-zinc-200 underline underline-offset-2 transition-colors"
              >
                Terms of Service
              </Link>{" "}
              and acknowledge our{" "}
              <Link
                to="/terms#disclaimers"
                className="text-zinc-400 hover:text-zinc-200 underline underline-offset-2 transition-colors"
              >
                Safety Disclaimers
              </Link>
              .
            </p>
          </div>
        </div>
      )}

      {/* DEVICE A: STEP 2: Verify Email Modal */}
      {!magicToken && !cliLoginApproved && !isOAuthProcessing && step === "otp" && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-800 bg-zinc-900 shadow-xl overflow-hidden relative">
          {renderVerifyEmailScreen(false)}
        </div>
      )}

      {/* DEVICE A: STEP 3: Sliding Transition into Account Setup */}
      {!magicToken && !cliLoginApproved && !isOAuthProcessing && step === "setup" && (
        <div className="w-full max-w-md rounded-2xl border border-zinc-800 bg-zinc-900 shadow-xl overflow-hidden relative">
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
