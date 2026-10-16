import { useState, useEffect, useRef } from "react";
import { Loader2, Check, Lock, AlertTriangle } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { EmailOtpPurpose, LoginChallengeStatus } from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import type {
  GetLoginChallengeDetailsResponse,
  ApproveLoginChallengeResponse,
} from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import logoDark from "../assets/quadsmith-logo-dark.svg";
import logoLight from "../assets/quadsmith-logo-light.svg";

export interface ChallengeApprovalScreenProps {
  token: string;
  onApproved?: (res: ApproveLoginChallengeResponse) => void;
  onDenied?: () => void;
  onError?: (err: Error) => void;
  className?: string;
  theme?: "light" | "dark";
}

export function ChallengeApprovalScreen({
  token,
  onApproved,
  onDenied,
  onError,
  className = "",
  theme: propTheme,
}: ChallengeApprovalScreenProps) {
  const { getLoginChallengeDetails, approveLoginChallenge, denyLoginChallenge } = useAuth();

  const [challengeDetails, setChallengeDetails] = useState<GetLoginChallengeDetailsResponse | null>(
    null,
  );
  const [approvalState, setApprovalState] = useState<
    "loading" | "prompt" | "approving" | "approved" | "denying" | "denied" | "error"
  >("loading");
  const [approvalError, setApprovalError] = useState<string | null>(null);

  const onApprovedRef = useRef(onApproved);
  const onDeniedRef = useRef(onDenied);
  const onErrorRef = useRef(onError);

  useEffect(() => {
    onApprovedRef.current = onApproved;
    onDeniedRef.current = onDenied;
    onErrorRef.current = onError;
  }, [onApproved, onDenied, onError]);

  const activeTheme =
    propTheme || (typeof localStorage !== "undefined" && localStorage.getItem("theme")) || "dark";

  useEffect(() => {
    if (!token) return;
    let active = true;

    async function loadDetails() {
      try {
        const details = await getLoginChallengeDetails(token);
        if (!active) return;
        setChallengeDetails(details);
        if (details.status === LoginChallengeStatus.APPROVED) {
          setApprovalState("approved");
        } else if (details.status === LoginChallengeStatus.DENIED) {
          setApprovalState("denied");
        } else if (details.status === LoginChallengeStatus.EXPIRED) {
          setApprovalState("error");
          setApprovalError(
            "This request is invalid or has expired. Please request a new verification code.",
          );
        } else {
          setApprovalState("prompt");
        }
      } catch (err: unknown) {
        if (!active) return;
        setApprovalState("error");
        const msg =
          err instanceof Error
            ? err.message
            : "Failed to load request details. It may be invalid or expired.";
        setApprovalError(msg);
        if (err instanceof Error) {
          onErrorRef.current?.(err);
        }
      }
    }

    loadDetails();
    return () => {
      active = false;
    };
  }, [token, getLoginChallengeDetails]);

  const handleApprove = async () => {
    if (!token || approvalState !== "prompt") return;
    setApprovalState("approving");
    try {
      const res = await approveLoginChallenge(token);
      setApprovalState("approved");
      onApprovedRef.current?.(res);
    } catch (err: unknown) {
      setApprovalState("error");
      const msg =
        err instanceof Error ? err.message : "Failed to approve request. Please try again.";
      setApprovalError(msg);
      if (err instanceof Error) {
        onErrorRef.current?.(err);
      }
    }
  };

  const handleDeny = async () => {
    if (!token || approvalState !== "prompt") return;
    setApprovalState("denying");
    try {
      await denyLoginChallenge(token);
      setApprovalState("denied");
      onDeniedRef.current?.();
    } catch (err: unknown) {
      setApprovalState("error");
      const msg = err instanceof Error ? err.message : "Failed to deny request.";
      setApprovalError(msg);
      if (err instanceof Error) {
        onErrorRef.current?.(err);
      }
    }
  };

  const isExport = challengeDetails?.purpose === EmailOtpPurpose.EXPORT_DATA;
  const isAddEmail = challengeDetails?.purpose === EmailOtpPurpose.ADD_EMAIL;
  const isDelete = challengeDetails?.purpose === EmailOtpPurpose.DELETE_ACCOUNT;

  const title = isExport
    ? "Approve Data Export"
    : isAddEmail
      ? "Approve Email Verification"
      : isDelete
        ? "Approve Account Deletion"
        : "Approve Login Request";

  const subtitle = isExport
    ? "A device is requesting to export your Quadsmith account data."
    : isAddEmail
      ? "A device is requesting to link this email address to your account."
      : isDelete
        ? "A device is requesting to delete your Quadsmith account."
        : "A device is requesting to sign in to your Quadsmith account.";

  const approveButtonText = isExport
    ? "Approve Data Export"
    : isAddEmail
      ? "Approve Email"
      : isDelete
        ? "Confirm Account Deletion"
        : "Approve Login";

  const successTitle = isExport
    ? "Export Approved!"
    : isAddEmail
      ? "Email Verified!"
      : isDelete
        ? "Deletion Confirmed"
        : "Login Approved!";

  const successMessage = isExport
    ? "Your data export has been approved and the download has been initiated on your other device."
    : isAddEmail
      ? "Your email address has been successfully verified on your other device."
      : isDelete
        ? "Your account deletion request has been confirmed."
        : `Your other device${challengeDetails?.deviceHint ? ` (${challengeDetails.deviceHint})` : ""} has been logged in.`;

  const deniedTitle = isExport
    ? "Data Export Denied"
    : isAddEmail
      ? "Email Verification Denied"
      : isDelete
        ? "Account Deletion Denied"
        : "Login Request Denied";

  const deniedMessage = isExport
    ? "The data export request has been canceled."
    : isAddEmail
      ? "The email verification request has been canceled."
      : isDelete
        ? "The account deletion request has been canceled."
        : "The sign-in request has been canceled.";

  return (
    <div className={`w-full p-8 space-y-6 ${className}`}>
      {/* Header */}
      <div className="text-center space-y-3">
        <div className="flex justify-center">
          <img
            src={activeTheme === "dark" ? logoDark : logoLight}
            alt="Quadsmith"
            className="h-10 w-auto"
          />
        </div>
        <h1 className="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
          {title}
        </h1>
        <p className="text-xs text-zinc-500 dark:text-zinc-400">{subtitle}</p>
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
            {approvalError || "This request is invalid or has expired."}
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
            {successTitle}
          </h2>
          <p className="text-xs text-zinc-600 dark:text-zinc-400">{successMessage}</p>
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
          <h2 className="text-lg font-bold text-rose-700 dark:text-rose-400">{deniedTitle}</h2>
          <p className="text-xs text-zinc-600 dark:text-zinc-400">{deniedMessage}</p>
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
                    <span>{approveButtonText}</span>
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
}
