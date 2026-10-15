import { useState, useEffect, useRef } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { ConnectError, Code } from "@connectrpc/connect";
import {
  User as UserIcon,
  LogOut,
  Check,
  Loader2,
  ExternalLink,
  Mail,
  ArrowRightLeft,
  ArrowLeft,
  Lock,
  Plus,
  X,
  Trash2,
  AlertCircle,
  CheckCircle2,
  UserX,
  Package,
  AlertTriangle,
  Shield,
  Laptop,
  Smartphone,
} from "lucide-react";
import { useAuth } from "../context/AuthContext";
import type { ArchivedUsername, Session } from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import { OtpVerificationErrorSchema } from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import { UsernameAvailabilityStatus } from "@quadsmith/shared/gen/quadsmith/api/user_pb";
import logoDark from "../assets/quadsmith-logo-dark.svg";
import logoLight from "../assets/quadsmith-logo-light.svg";
import { getAvatarPalette, getInitial, getApiBaseUrl } from "@quadsmith/shared";

export function AccountSettingsPage() {
  const {
    user,
    profile,
    emails,
    sessions,
    refreshSessions,
    revokeSession,
    revokeOtherSessions,
    updateProfile,
    logout,
    refreshEmails,
    sendAddEmailOtp,
    verifyAddEmailOtp,
    mergeAccount,
    setPrimaryEmail,
    removeUserEmail,
    changeUsername,
    listArchivedUsernames,
    checkUsernameAvailable,
    sendDataExportVerification,
    verifyDataExport,
  } = useAuth();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const mergeToken = searchParams.get("merge_token");
  const addEmailToken = searchParams.get("add_email_token");
  const exportDataToken = searchParams.get("export_data_token");

  // Active Sessions state
  const [sessionToRevoke, setSessionToRevoke] = useState<Session | null>(null);
  const [isRevokingSession, setIsRevokingSession] = useState(false);
  const [revokeSessionError, setRevokeSessionError] = useState<string | null>(null);

  const [isSignOutAllModalOpen, setIsSignOutAllModalOpen] = useState(false);
  const [isRevokingOtherSessions, setIsRevokingOtherSessions] = useState(false);
  const [signOutAllError, setSignOutAllError] = useState<string | null>(null);

  const currentTheme =
    (typeof localStorage !== "undefined" && localStorage.getItem("theme")) ||
    (typeof document !== "undefined" && document.documentElement.classList.contains("dark")
      ? "dark"
      : "light") ||
    "dark";

  // Change Username Modal state
  const [isChangeUsernameOpen, setIsChangeUsernameOpen] = useState(false);
  const [archivedUsernames, setArchivedUsernames] = useState<ArchivedUsername[]>([]);
  const [nextAllowedNewRenameAt, setNextAllowedNewRenameAt] = useState<Date | null>(null);
  const [isLoadingArchived, setIsLoadingArchived] = useState(false);
  const [newUsernameInput, setNewUsernameInput] = useState("");
  const [usernameAvailability, setUsernameAvailability] = useState<{
    status: "idle" | "checking" | "available" | "unavailable" | "invalid" | "error";
    message: string;
  }>({ status: "idle", message: "" });
  const [isSubmittingUsername, setIsSubmittingUsername] = useState(false);
  const [usernameError, setUsernameError] = useState<string | null>(null);
  const [usernameSuccess, setUsernameSuccess] = useState<string | null>(null);

  useEffect(() => {
    refreshEmails();
    refreshSessions();
  }, [refreshEmails, refreshSessions]);

  // Profile form state
  const [avatarUrl, setAvatarUrl] = useState(profile?.avatarUrl || "");
  const [bio, setBio] = useState(profile?.bio || "");
  const [initialValues, setInitialValues] = useState({
    avatarUrl: profile?.avatarUrl || "",
    bio: profile?.bio || "",
  });
  const [isSaving, setIsSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // Account Deletion state
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [deleteConfirmHandle, setDeleteConfirmHandle] = useState("");

  // Synchronize initial values when profile loads
  useEffect(() => {
    if (profile) {
      setAvatarUrl(profile.avatarUrl || "");
      setBio(profile.bio || "");
      setInitialValues({
        avatarUrl: profile.avatarUrl || "",
        bio: profile.bio || "",
      });
    }
  }, [profile]);

  const isAvatarModified = avatarUrl.trim() !== (initialValues.avatarUrl || "").trim();
  const isBioModified = bio.trim() !== (initialValues.bio || "").trim();
  const isDirty = isAvatarModified || isBioModified;

  // Add Email Modal state (mirrors login flow)
  const [isAddEmailOpen, setIsAddEmailOpen] = useState(false);
  const [addStep, setAddStep] = useState<"email" | "otp">("email");
  const [newEmail, setNewEmail] = useState("");
  const [addOtpDigits, setAddOtpDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const [addHintDigits, setAddHintDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const [isSendingAddOtp, setIsSendingAddOtp] = useState(false);
  const [isVerifyingAddOtp, setIsVerifyingAddOtp] = useState(false);
  const [addEmailError, setAddEmailError] = useState<string | null>(null);
  const [addEmailSuccess, setAddEmailSuccess] = useState<string | null>(null);
  const [addRemainingAttempts, setAddRemainingAttempts] = useState<number | null>(null);
  const [isAddExhausted, setIsAddExhausted] = useState(false);
  const [isAddRateLimited, setIsAddRateLimited] = useState(false);
  const [addCooldown, setAddCooldown] = useState(0);
  const [addExhaustedCooldown, setAddExhaustedCooldown] = useState(0);
  const [addRateLimitCooldown, setAddRateLimitCooldown] = useState(0);
  const addOtpInputRefs = useRef<(HTMLInputElement | null)[]>([]);

  // Add Email Cooldown timer
  useEffect(() => {
    if (addCooldown <= 0) return;
    const timer = setInterval(() => {
      setAddCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [addCooldown]);

  // Add Email Exhausted Cooldown timer
  useEffect(() => {
    if (addExhaustedCooldown <= 0) return;
    const timer = setInterval(() => {
      setAddExhaustedCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [addExhaustedCooldown]);

  // Add Email Rate Limit Cooldown timer
  useEffect(() => {
    if (addRateLimitCooldown <= 0) return;
    const timer = setInterval(() => {
      setAddRateLimitCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [addRateLimitCooldown]);

  const openAddEmailModal = () => {
    setIsAddEmailOpen(true);
    setAddStep("email");
    setNewEmail("");
    setAddOtpDigits(["", "", "", "", "", ""]);
    setAddHintDigits(["", "", "", "", "", ""]);
    setAddEmailError(null);
    setAddRemainingAttempts(null);
    setIsAddExhausted(false);
    setIsAddRateLimited(false);
    setAddCooldown(0);
    setAddExhaustedCooldown(0);
    setAddRateLimitCooldown(0);
  };

  const closeAddEmailModal = () => {
    setIsAddEmailOpen(false);
    setAddStep("email");
    setNewEmail("");
    setAddOtpDigits(["", "", "", "", "", ""]);
    setAddHintDigits(["", "", "", "", "", ""]);
    setAddEmailError(null);
  };

  // Merge Account Modal state
  const [isMergeModalOpen, setIsMergeModalOpen] = useState(false);
  const [mergeModalStep, setMergeModalStep] = useState<"summary" | "success">("summary");
  const [mergeEmail, setMergeEmail] = useState("");
  const [mergeOtherUsername, setMergeOtherUsername] = useState("");
  const [mergeOtherBuildCount, setMergeOtherBuildCount] = useState<number>(0);
  const [mergeOtherBuildNames, setMergeOtherBuildNames] = useState<string[]>([]);
  const [mergeOtherEmails, setMergeOtherEmails] = useState<string[]>([]);
  const [mergeOtherOauthProviders, setMergeOtherOauthProviders] = useState<string[]>([]);
  const [verifiedMergeCode, setVerifiedMergeCode] = useState("");
  const [verifiedMergeToken, setVerifiedMergeToken] = useState("");
  const [isExecutingMerge, setIsExecutingMerge] = useState(false);
  const [mergeError, setMergeError] = useState<string | null>(null);
  const [transferredBuilds, setTransferredBuilds] = useState<number>(0);

  // Magic link auto-merge or auto-add-email
  useEffect(() => {
    if (mergeToken) {
      let isMounted = true;
      async function handleMagicMerge() {
        setMergeError(null);
        setIsMergeModalOpen(true);
        try {
          const res = await verifyAddEmailOtp("", undefined, mergeToken);
          if (!isMounted) return;
          if (res.mergeRequired) {
            setMergeOtherUsername(res.otherUserUsername || "user");
            setMergeOtherBuildCount(res.otherBuildCount || 0);
            setMergeOtherBuildNames(res.otherBuildNames || []);
            setMergeOtherEmails(res.otherEmails || []);
            setMergeOtherOauthProviders(res.otherOauthProviders || []);
            if (res.otherEmails && res.otherEmails.length > 0) {
              setMergeEmail(res.otherEmails[0]);
            }
            setVerifiedMergeToken(mergeToken || "");
            setMergeModalStep("summary");
          } else {
            setAddEmailSuccess(
              "Email verified and added to your account successfully via magic link.",
            );
            setTimeout(() => setAddEmailSuccess(null), 4000);
            setIsMergeModalOpen(false);
            await refreshEmails();
          }
          searchParams.delete("merge_token");
          setSearchParams(searchParams, { replace: true });
        } catch (err: unknown) {
          if (!isMounted) return;
          if (err instanceof Error) {
            setMergeError(err.message);
          } else {
            setMergeError("Failed to authorize merge with magic link.");
          }
        }
      }
      handleMagicMerge();
      return () => {
        isMounted = false;
      };
    } else if (addEmailToken) {
      let isMounted = true;
      async function handleMagicAddEmail() {
        try {
          const res = await verifyAddEmailOtp("", undefined, addEmailToken);
          if (!isMounted) return;
          if (res.mergeRequired) {
            setMergeEmail(res.otherEmails?.[0] || "");
            setMergeOtherUsername(res.otherUserUsername || "user");
            setMergeOtherBuildCount(res.otherBuildCount || 0);
            setMergeOtherBuildNames(res.otherBuildNames || []);
            setMergeOtherEmails(res.otherEmails || []);
            setMergeOtherOauthProviders(res.otherOauthProviders || []);
            setVerifiedMergeToken(addEmailToken || "");
            setMergeModalStep("summary");
            setIsMergeModalOpen(true);
          } else {
            setAddEmailSuccess(
              "Email verified and added to your account successfully via magic link.",
            );
            setTimeout(() => setAddEmailSuccess(null), 4000);
            await refreshEmails();
          }
          searchParams.delete("add_email_token");
          setSearchParams(searchParams, { replace: true });
        } catch (err: unknown) {
          if (!isMounted) return;
          if (err instanceof Error) {
            setEmailActionError(err.message);
          } else {
            setEmailActionError("Failed to verify email magic link.");
          }
        }
      }
      handleMagicAddEmail();
      return () => {
        isMounted = false;
      };
    }
  }, [mergeToken, addEmailToken, verifyAddEmailOtp, refreshEmails, searchParams, setSearchParams]);

  // Data Export state
  const [isExportModalOpen, setIsExportModalOpen] = useState(false);
  const [isExportSending, setIsExportSending] = useState(false);
  const [isExportVerifying, setIsExportVerifying] = useState(false);
  const [exportChallengeId, setExportChallengeId] = useState<string | null>(null);
  const [exportPrimaryEmail, setExportPrimaryEmail] = useState<string>("");
  const [exportCooldown, setExportCooldown] = useState(0);
  const [exportError, setExportError] = useState<string | null>(null);
  const [exportSuccessMessage, setExportSuccessMessage] = useState<string | null>(null);
  const [exportRemainingAttempts, setExportRemainingAttempts] = useState<number | null>(null);
  const [isExportExhausted, setIsExportExhausted] = useState(false);
  const [isExportRateLimited, setIsExportRateLimited] = useState(false);
  const [exportDigits, setExportDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const exportInputRefs = useRef<(HTMLInputElement | null)[]>([]);

  useEffect(() => {
    if (exportCooldown <= 0) return;
    const timer = setInterval(() => {
      setExportCooldown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, [exportCooldown]);

  useEffect(() => {
    if (!exportDataToken) return;

    let active = true;
    (async () => {
      try {
        setIsExportVerifying(true);
        const res = await verifyDataExport(undefined, undefined, exportDataToken);
        if (active && res.success && res.downloadUrl) {
          const link = document.createElement("a");
          const rawUrl = res.downloadUrl;
          link.href = rawUrl.startsWith("http") ? rawUrl : `${getApiBaseUrl()}${rawUrl}`;
          link.setAttribute("download", res.filename || "quadsmith-data-export.zip");
          document.body.appendChild(link);
          link.click();
          document.body.removeChild(link);

          setExportSuccessMessage(
            `Your data download (${res.filename || "export.zip"}) has started.`,
          );
          searchParams.delete("export_data_token");
          setSearchParams(searchParams, { replace: true });
        }
      } catch (err) {
        if (active) {
          const message = err instanceof Error ? err.message : "Failed to verify magic link";
          setExportError(message);
          setIsExportModalOpen(true);
        }
      } finally {
        if (active) {
          setIsExportVerifying(false);
        }
      }
    })();

    return () => {
      active = false;
    };
  }, [exportDataToken, verifyDataExport, searchParams, setSearchParams]);

  const handleOpenExportModal = async () => {
    setIsExportModalOpen(true);
    setExportError(null);
    setExportSuccessMessage(null);
    setExportDigits(["", "", "", "", "", ""]);
    setExportRemainingAttempts(null);
    setIsExportExhausted(false);
    setIsExportRateLimited(false);
    setIsExportSending(true);

    try {
      const res = await sendDataExportVerification();
      if (res.success) {
        setExportChallengeId(res.challengeId);
        setExportPrimaryEmail(res.primaryEmail || "");
        if (res.cooldownSeconds) {
          setExportCooldown(res.cooldownSeconds);
        }
        setTimeout(() => {
          exportInputRefs.current[0]?.focus();
        }, 100);
      }
    } catch (err: unknown) {
      if (err instanceof ConnectError) {
        setExportError(err.rawMessage);
      } else {
        setExportError(err instanceof Error ? err.message : "Failed to initiate data export");
      }
    } finally {
      setIsExportSending(false);
    }
  };

  const handleVerifyExportOtp = async (codeToVerify?: string) => {
    const code = codeToVerify || exportDigits.join("");
    if (code.length !== 6) {
      setExportError("Please enter all 6 digits of your verification code.");
      return;
    }

    setIsExportVerifying(true);
    setExportError(null);

    try {
      const res = await verifyDataExport(exportChallengeId, code, undefined);
      if (res.success && res.downloadUrl) {
        setIsExportModalOpen(false);

        const link = document.createElement("a");
        const rawUrl = res.downloadUrl;
        link.href = rawUrl.startsWith("http") ? rawUrl : `${getApiBaseUrl()}${rawUrl}`;
        link.setAttribute("download", res.filename || "quadsmith-data-export.zip");
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        setExportSuccessMessage(
          `Your data download (${res.filename || "export.zip"}) has started.`,
        );
      }
    } catch (err: unknown) {
      if (err instanceof ConnectError) {
        const errorDetail = err.findDetails(OtpVerificationErrorSchema)[0];
        if (errorDetail) {
          setExportError(errorDetail.message || err.rawMessage);
          setExportRemainingAttempts(errorDetail.remainingAttempts);
          if (errorDetail.isExhausted) {
            setIsExportExhausted(true);
          }
          if (errorDetail.cooldownSeconds > 0) {
            setIsExportRateLimited(true);
          }
        } else {
          setExportError(err.rawMessage);
        }
      } else {
        setExportError(err instanceof Error ? err.message : "Verification failed");
      }
    } finally {
      setIsExportVerifying(false);
    }
  };

  const handleExportOtpChange = (index: number, val: string) => {
    const cleaned = val.replace(/\D/g, "");
    if (!cleaned) {
      const newDigits = [...exportDigits];
      newDigits[index] = "";
      setExportDigits(newDigits);
      return;
    }

    const char = cleaned.slice(-1);
    const newDigits = [...exportDigits];
    newDigits[index] = char;
    setExportDigits(newDigits);

    if (index < 5) {
      exportInputRefs.current[index + 1]?.focus();
    } else {
      const fullCode = newDigits.join("");
      if (fullCode.length === 6) {
        handleVerifyExportOtp(fullCode);
      }
    }
  };

  const handleExportKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Backspace") {
      if (!exportDigits[index] && index > 0) {
        exportInputRefs.current[index - 1]?.focus();
      }
    } else if (e.key === "ArrowLeft" && index > 0) {
      exportInputRefs.current[index - 1]?.focus();
    } else if (e.key === "ArrowRight" && index < 5) {
      exportInputRefs.current[index + 1]?.focus();
    }
  };

  const handleExportPaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault();
    const pasted = e.clipboardData.getData("text").replace(/\D/g, "").slice(0, 6);
    if (!pasted) return;

    const newDigits = [...exportDigits];
    for (let i = 0; i < pasted.length; i++) {
      newDigits[i] = pasted[i];
    }
    setExportDigits(newDigits);

    const nextIndex = Math.min(pasted.length, 5);
    exportInputRefs.current[nextIndex]?.focus();

    if (pasted.length === 6) {
      handleVerifyExportOtp(pasted);
    }
  };

  // Email Action feedback
  const [emailActionLoading, setEmailActionLoading] = useState<string | null>(null);
  const [emailActionError, setEmailActionError] = useState<string | null>(null);

  const palette = getAvatarPalette(user?.uuid);
  const initial = getInitial(user?.id);

  // Handle Profile Save
  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isDirty || isSaving) return;
    setIsSaving(true);
    setSaveSuccess(false);
    setErrorMsg(null);

    try {
      await updateProfile({
        avatarUrl: avatarUrl.trim(),
        bio: bio.trim(),
      });
      setInitialValues({
        avatarUrl: avatarUrl.trim(),
        bio: bio.trim(),
      });
      setSaveSuccess(true);
      setTimeout(() => setSaveSuccess(false), 3000);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg("Failed to save changes. Please try again.");
      }
    } finally {
      setIsSaving(false);
    }
  };

  // Handle Logout
  const handleLogout = async () => {
    await logout();
    navigate("/login", { replace: true });
  };

  // Handle Add Email - Send Code (Step 1)
  const handleSendAddCode = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    const cleanEmail = newEmail.trim().toLowerCase();
    if (!cleanEmail) {
      setAddEmailError("Please enter an email address");
      return;
    }
    if (isSendingAddOtp || addCooldown > 0 || (isAddRateLimited && addRateLimitCooldown > 0)) {
      return;
    }

    setIsSendingAddOtp(true);
    setAddEmailError(null);
    try {
      const res = await sendAddEmailOtp(cleanEmail);
      setAddStep("otp");
      setAddCooldown(res.cooldownSeconds || 20);
      setAddExhaustedCooldown(0);
      setAddRateLimitCooldown(0);
      setAddOtpDigits(["", "", "", "", "", ""]);
      setAddHintDigits(["", "", "", "", "", ""]);
      setIsAddExhausted(false);
      setIsAddRateLimited(false);
      setAddRemainingAttempts(null);
      setTimeout(() => addOtpInputRefs.current[0]?.focus(), 100);
    } catch (err: unknown) {
      if (err instanceof ConnectError) {
        if (
          err.code === Code.ResourceExhausted &&
          (err.rawMessage.toLowerCase().includes("too many") ||
            err.rawMessage.toLowerCase().includes("minutes"))
        ) {
          setIsAddRateLimited(true);
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
          setAddRateLimitCooldown(cd);
          setAddEmailError(err.rawMessage);
          return;
        }
      }
      if (err instanceof Error) {
        setAddEmailError(err.message);
      } else {
        setAddEmailError("Failed to send verification code. Please try again.");
      }
    } finally {
      setIsSendingAddOtp(false);
    }
  };

  // Handle Add Email - Verify Code (Step 2)
  const handleVerifyAddCode = async (explicitCode?: string) => {
    const code = explicitCode || addOtpDigits.join("");
    if (code.length !== 6) {
      setAddEmailError("Please enter all 6 digits of the verification code");
      return;
    }
    if (isVerifyingAddOtp) return;

    setIsVerifyingAddOtp(true);
    setAddEmailError(null);
    try {
      const res = await verifyAddEmailOtp(newEmail.trim().toLowerCase(), code);
      if (res.mergeRequired) {
        setMergeEmail(newEmail.trim().toLowerCase());
        setMergeOtherUsername(res.otherUserUsername || "user");
        setMergeOtherBuildCount(res.otherBuildCount || 0);
        setMergeOtherBuildNames(res.otherBuildNames || []);
        setMergeOtherEmails(res.otherEmails || []);
        setMergeOtherOauthProviders(res.otherOauthProviders || []);
        setVerifiedMergeCode(code);
        setVerifiedMergeToken(res.mergeTicket || "");
        setMergeError(null);
        setMergeModalStep("summary");
        setIsMergeModalOpen(true);
        setIsAddEmailOpen(false);
        return;
      }
      setAddEmailSuccess(`Successfully added ${newEmail.trim()} to your account.`);
      setTimeout(() => setAddEmailSuccess(null), 4000);
      setIsAddEmailOpen(false);
      setNewEmail("");
      setAddOtpDigits(["", "", "", "", "", ""]);
      setAddHintDigits(["", "", "", "", "", ""]);
    } catch (err: unknown) {
      // Clear fields and show incorrect code as faded placeholder hints
      setAddHintDigits(code.split(""));
      setAddOtpDigits(["", "", "", "", "", ""]);
      setTimeout(() => addOtpInputRefs.current[0]?.focus(), 10);

      if (err instanceof ConnectError) {
        if (
          err.code === Code.ResourceExhausted &&
          (err.rawMessage.toLowerCase().includes("too many") ||
            err.rawMessage.toLowerCase().includes("minutes"))
        ) {
          setIsAddRateLimited(true);
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
          setAddRateLimitCooldown(cd);
          setAddEmailError(err.rawMessage);
          return;
        }
        const detail = err.findDetails(OtpVerificationErrorSchema)[0];
        if (detail) {
          setAddRemainingAttempts(detail.remainingAttempts);
          setIsAddExhausted(detail.isExhausted);
          if (detail.isExhausted) {
            setAddExhaustedCooldown(detail.cooldownSeconds || 15);
          }
          setAddEmailError(detail.message);
        } else {
          setAddEmailError(err.rawMessage);
        }
      } else if (err instanceof Error) {
        setAddEmailError(err.message);
      } else {
        setAddEmailError("Verification failed. Please check the code and try again.");
      }
    } finally {
      setIsVerifyingAddOtp(false);
    }
  };

  const handleAddOtpChange = (index: number, value: string) => {
    const clean = value.replace(/\D/g, "");
    if (!clean && value !== "") return;

    const newDigits = [...addOtpDigits];

    if (clean.length > 1) {
      const chars = clean.slice(0, 6).split("");
      chars.forEach((c, i) => {
        newDigits[i] = c;
      });
      setAddOtpDigits(newDigits);
      const nextIndex = Math.min(chars.length, 5);
      addOtpInputRefs.current[nextIndex]?.focus();
      const code = newDigits.join("");
      if (code.length === 6 && !isVerifyingAddOtp) {
        handleVerifyAddCode(code);
      }
      return;
    }

    newDigits[index] = clean;
    setAddOtpDigits(newDigits);

    // Auto-advance
    if (clean && index < 5) {
      addOtpInputRefs.current[index + 1]?.focus();
    }

    const code = newDigits.join("");
    if (code.length === 6 && !isVerifyingAddOtp) {
      handleVerifyAddCode(code);
    }
  };

  const handleAddKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Backspace" && !addOtpDigits[index] && index > 0) {
      addOtpInputRefs.current[index - 1]?.focus();
    }
  };

  const handleAddPaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault();
    const pasted = e.clipboardData.getData("text").replace(/\D/g, "").slice(0, 6);
    if (!pasted) return;
    const newDigits = [...addOtpDigits];
    pasted.split("").forEach((c, i) => {
      newDigits[i] = c;
    });
    setAddOtpDigits(newDigits);
    const nextIndex = Math.min(pasted.length, 5);
    addOtpInputRefs.current[nextIndex]?.focus();
    if (pasted.length === 6 && !isVerifyingAddOtp) {
      handleVerifyAddCode(pasted);
    }
  };

  // Step 2: Confirm Merge after reviewing the impact summary
  const handleConfirmMerge = async () => {
    setIsExecutingMerge(true);
    setMergeError(null);
    try {
      const res = await mergeAccount(
        mergeEmail.trim(),
        verifiedMergeCode || undefined,
        verifiedMergeToken || undefined,
      );
      setTransferredBuilds(res.transferredBuildCount || 0);
      if (res.mergedEmails && res.mergedEmails.length > 0) {
        setMergeOtherEmails(res.mergedEmails);
      }
      setMergeModalStep("success");
      await refreshEmails();
    } catch (err: unknown) {
      if (err instanceof Error) {
        setMergeError(err.message);
      } else {
        setMergeError("Merge failed. Please try again.");
      }
    } finally {
      setIsExecutingMerge(false);
    }
  };

  // Handle Set Primary
  const handleSetPrimary = async (emailAddr: string) => {
    setEmailActionLoading(emailAddr);
    setEmailActionError(null);
    try {
      await setPrimaryEmail(emailAddr);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setEmailActionError(err.message);
      } else {
        setEmailActionError("Failed to set primary email.");
      }
    } finally {
      setEmailActionLoading(null);
    }
  };

  // Handle Remove Email
  const handleRemoveEmail = async (emailAddr: string) => {
    if (
      !window.confirm(
        `Are you sure you want to remove ${emailAddr} from your account? You will no longer be able to use it to log in.`,
      )
    ) {
      return;
    }
    setEmailActionLoading(emailAddr);
    setEmailActionError(null);
    try {
      await removeUserEmail(emailAddr);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setEmailActionError(err.message);
      } else {
        setEmailActionError("Failed to remove email.");
      }
    } finally {
      setEmailActionLoading(null);
    }
  };

  // Fallback for displayed emails
  const displayedEmails =
    (emails?.length ?? 0) > 0
      ? emails
      : user?.email
        ? [
            {
              $typeName: "quadsmith.UserEmail" as const,
              uuid: "",
              userUuid: "",
              email: user.email,
              isPrimary: true,
              createdAt: undefined,
            },
          ]
        : [];

  const formatDate = (seconds?: bigint) => {
    if (!seconds) return "Recent";
    const ms = Number(seconds) * 1000;
    return new Date(ms).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  };

  const formatTimestamp = (ts?: { seconds?: bigint }) => {
    if (!ts || !ts.seconds) return "";
    return new Date(Number(ts.seconds) * 1000).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  };

  const isMobileDevice = (deviceHint: string): boolean => {
    const lower = (deviceHint || "").toLowerCase();
    return (
      lower.includes("ios") ||
      lower.includes("android") ||
      lower.includes("mobile") ||
      lower.includes("iphone") ||
      lower.includes("ipad")
    );
  };

  const formatSessionActivity = (lastActiveDate?: Date | null, isCurrent?: boolean): string => {
    if (isCurrent) return "Active now";
    if (!lastActiveDate) return "Active recently";
    const diffMs = Date.now() - lastActiveDate.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    if (diffMins < 5) return "Active now";
    if (diffMins < 60) return `Active ${diffMins}m ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `Active ${diffHours}h ago`;
    const diffDays = Math.floor(diffHours / 24);
    return `Active ${diffDays}d ago`;
  };

  const activeNowCount = (sessions || []).filter((s) => {
    if (s.isCurrent) return true;
    if (!s.lastActiveAt || !s.lastActiveAt.seconds) return false;
    return Date.now() - Number(s.lastActiveAt.seconds) * 1000 < 5 * 60 * 1000;
  }).length;

  const hasOtherSessions = (sessions || []).some((s) => !s.isCurrent);

  const handleConfirmRevokeSession = async () => {
    if (!sessionToRevoke) return;
    setIsRevokingSession(true);
    setRevokeSessionError(null);
    try {
      await revokeSession(sessionToRevoke.uuid);
      setSessionToRevoke(null);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to revoke session";
      setRevokeSessionError(msg);
    } finally {
      setIsRevokingSession(false);
    }
  };

  const handleConfirmSignOutAllOther = async () => {
    setIsRevokingOtherSessions(true);
    setSignOutAllError(null);
    try {
      await revokeOtherSessions();
      setIsSignOutAllModalOpen(false);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to sign out other sessions";
      setSignOutAllError(msg);
    } finally {
      setIsRevokingOtherSessions(false);
    }
  };

  const openChangeUsernameModal = async () => {
    setIsChangeUsernameOpen(true);
    setNewUsernameInput("");
    setUsernameError(null);
    setUsernameSuccess(null);
    setUsernameAvailability({ status: "idle", message: "" });
    setIsLoadingArchived(true);
    try {
      const res = await listArchivedUsernames();
      setArchivedUsernames(res.archivedUsernames || []);
      if (res.nextAllowedNewRenameAt?.seconds) {
        setNextAllowedNewRenameAt(new Date(Number(res.nextAllowedNewRenameAt.seconds) * 1000));
      } else {
        setNextAllowedNewRenameAt(null);
      }
    } catch {
      // ignore
    } finally {
      setIsLoadingArchived(false);
    }
  };

  useEffect(() => {
    const trimmed = newUsernameInput.trim().toLowerCase();
    if (!trimmed) {
      setUsernameAvailability({ status: "idle", message: "" });
      return;
    }

    if (trimmed === user?.id?.toLowerCase()) {
      setUsernameAvailability({ status: "unavailable", message: "This is your current username" });
      return;
    }

    // Client-side regex precheck
    const regex = /^[a-z][a-z0-9_]{1,18}[a-z0-9]$/;
    if (!regex.test(trimmed)) {
      setUsernameAvailability({
        status: "invalid",
        message: "",
      });
      return;
    }
    if (trimmed.includes("__")) {
      setUsernameAvailability({
        status: "invalid",
        message: "",
      });
      return;
    }
    if (trimmed.includes("quadsmith") || trimmed.includes("quad_smith")) {
      setUsernameAvailability({ status: "invalid", message: "" });
      return;
    }

    setUsernameAvailability({ status: "checking", message: "Checking availability..." });
    const timer = setTimeout(async () => {
      try {
        const res = await checkUsernameAvailable(trimmed);
        if (res.available) {
          setUsernameAvailability({
            status: "available",
            message: res.message || "Username is available",
          });
        } else if (res.status === UsernameAvailabilityStatus.INVALID) {
          setUsernameAvailability({
            status: "invalid",
            message: "",
          });
        } else {
          setUsernameAvailability({
            status: "unavailable",
            message: res.message || "This username is taken",
          });
        }
      } catch {
        setUsernameAvailability({
          status: "error",
          message: "Error checking username availability",
        });
      }
    }, 300);

    return () => clearTimeout(timer);
  }, [newUsernameInput, user?.id, checkUsernameAvailable]);

  const handleReclaimUsername = async (targetUsername: string) => {
    setIsSubmittingUsername(true);
    setUsernameError(null);
    try {
      const res = await changeUsername(targetUsername);
      setUsernameSuccess(res.message || `Username changed to @${targetUsername}`);
      setTimeout(() => {
        setIsChangeUsernameOpen(false);
        setUsernameSuccess(null);
      }, 1500);
    } catch (err: unknown) {
      setUsernameError(err instanceof Error ? err.message : "Failed to reclaim username");
    } finally {
      setIsSubmittingUsername(false);
    }
  };

  const handleSubmitNewUsername = async (e: React.FormEvent) => {
    e.preventDefault();
    if (usernameAvailability.status !== "available" || !newUsernameInput.trim()) return;

    setIsSubmittingUsername(true);
    setUsernameError(null);
    try {
      const res = await changeUsername(newUsernameInput.trim().toLowerCase());
      setUsernameSuccess(
        res.message || `Username changed to @${newUsernameInput.trim().toLowerCase()}`,
      );
      setTimeout(() => {
        setIsChangeUsernameOpen(false);
        setUsernameSuccess(null);
      }, 1500);
    } catch (err: unknown) {
      setUsernameError(err instanceof Error ? err.message : "Failed to change username");
    } finally {
      setIsSubmittingUsername(false);
    }
  };

  const isNewUsernameOnCooldown =
    nextAllowedNewRenameAt !== null && nextAllowedNewRenameAt.getTime() > Date.now();
  const newUsernameCooldownDays = nextAllowedNewRenameAt
    ? Math.max(1, Math.ceil((nextAllowedNewRenameAt.getTime() - Date.now()) / (24 * 3600 * 1000)))
    : 0;

  const isBanned = Boolean(user?.activeBan);

  return (
    <div className="max-w-4xl mx-auto px-4 py-8 space-y-8">
      {/* Prominent Ban Alert Banner */}
      {isBanned && user?.activeBan && (
        <div
          data-testid="ban-alert-banner"
          className="rounded-2xl border border-rose-500/30 bg-rose-500/10 dark:bg-rose-950/30 p-5 flex items-start gap-4 shadow-sm"
        >
          <div className="w-10 h-10 rounded-xl bg-rose-500/20 text-rose-600 dark:text-rose-400 flex items-center justify-center shrink-0 mt-0.5">
            <AlertTriangle size={20} />
          </div>
          <div className="space-y-1 text-xs flex-1">
            <div className="font-bold text-rose-700 dark:text-rose-300 text-sm flex items-center gap-2">
              <span>Account Suspended</span>
              <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-rose-500/20 text-rose-600 dark:text-rose-400">
                {user.activeBan.expiresAt ? "TEMPORARY" : "INDEFINITE"}
              </span>
            </div>
            <p className="text-zinc-700 dark:text-zinc-300 leading-relaxed">
              Your account functionality has been restricted due to:{" "}
              <em className="font-medium text-rose-700 dark:text-rose-300">
                &ldquo;{user.activeBan.reason}&rdquo;
              </em>
            </p>
            {user.activeBan.expiresAt && (
              <p className="text-zinc-600 dark:text-zinc-400">
                Suspension expires:{" "}
                <span className="font-semibold text-zinc-800 dark:text-zinc-200">
                  {new Date(Number(user.activeBan.expiresAt.seconds) * 1000).toLocaleDateString(
                    undefined,
                    { year: "numeric", month: "long", day: "numeric" },
                  )}
                </span>
              </p>
            )}
            <p className="text-[11px] text-zinc-500 dark:text-zinc-400">
              Handle changes and new email associations are locked. You maintain the right to update
              your bio, manage active login sessions, export data, or delete your account.
            </p>
          </div>
        </div>
      )}

      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-200 dark:border-zinc-800 pb-6">
        <div className="flex items-center gap-4">
          {avatarUrl ? (
            <img
              src={avatarUrl}
              alt={user?.id || "User avatar"}
              className="w-16 h-16 rounded-full object-cover border-2 border-zinc-200 dark:border-zinc-800 shadow-sm"
            />
          ) : (
            <div
              className={`w-16 h-16 rounded-full border flex items-center justify-center font-bold text-2xl shadow-sm ${palette.badgeClass}`}
            >
              {initial}
            </div>
          )}
          <div>
            <h1 className="text-2xl font-bold tracking-tight">Account Settings</h1>
            <div className="flex items-center gap-2 mt-0.5">
              <p className="text-xs text-zinc-500 dark:text-zinc-400 font-mono">
                Username:{" "}
                <span className="font-semibold text-zinc-800 dark:text-zinc-200">@{user?.id}</span>
              </p>
              {!isBanned ? (
                <button
                  type="button"
                  onClick={openChangeUsernameModal}
                  className="px-2 py-0.5 text-[11px] font-semibold text-blue-600 dark:text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/20 rounded-md transition flex items-center gap-1 shadow-2xs cursor-pointer"
                >
                  Change Username
                </button>
              ) : (
                <span className="px-2 py-0.5 text-[11px] font-medium text-zinc-500 dark:text-zinc-400 bg-zinc-100 dark:bg-zinc-800/60 rounded-md flex items-center gap-1">
                  <Lock size={10} />
                  Handle locked
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Header Action Buttons */}
        <div className="flex items-center gap-3">
          <Link
            to={`/users/${user?.id}`}
            className="px-4 py-2 rounded-xl text-xs font-semibold border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-200 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition flex items-center gap-1.5 shadow-2xs"
          >
            <UserIcon size={14} />
            <span>View Public Profile</span>
            <ExternalLink size={12} className="opacity-70" />
          </Link>
          <button
            onClick={handleLogout}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-rose-600 dark:text-rose-400 border border-rose-500/20 bg-rose-500/10 hover:bg-rose-500/20 transition flex items-center gap-1.5 shadow-2xs"
          >
            <LogOut size={14} />
            <span>Log Out</span>
          </button>
        </div>
      </div>

      {/* Section 1: Associated Email Addresses */}
      <section className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-6 space-y-5 shadow-2xs">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div>
            <h2 className="text-sm font-bold flex items-center gap-2">
              <Mail size={16} className="text-blue-500" />
              <span>Associated Email Addresses</span>
            </h2>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
              Any verified email address listed below can be used to receive login codes and access
              your account.
            </p>
          </div>
          {!isBanned ? (
            <button
              type="button"
              onClick={openAddEmailModal}
              className="px-3.5 py-1.5 rounded-xl text-xs font-semibold bg-blue-600 hover:bg-blue-500 text-white transition flex items-center gap-1.5 self-start sm:self-auto shadow-sm cursor-pointer"
            >
              <Plus size={14} />
              <span>Add Email</span>
            </button>
          ) : (
            <span className="px-3 py-1 rounded-xl text-xs font-medium text-zinc-500 dark:text-zinc-400 bg-zinc-100 dark:bg-zinc-800/60 border border-zinc-200 dark:border-zinc-800 flex items-center gap-1.5">
              <Lock size={12} />
              <span>Email additions locked</span>
            </span>
          )}
        </div>

        {addEmailSuccess && (
          <div className="p-3 rounded-xl border border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 text-xs flex items-center gap-2">
            <Check size={14} />
            <span>{addEmailSuccess}</span>
          </div>
        )}

        {emailActionError && (
          <div className="p-3 rounded-xl border border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400 text-xs flex items-center gap-2">
            <AlertCircle size={14} />
            <span>{emailActionError}</span>
          </div>
        )}

        {/* Email List Cards */}
        <div role="table" aria-label="Associated Emails" className="space-y-2">
          {displayedEmails.map((em) => (
            <div
              role="row"
              key={em.email}
              className={`p-3.5 rounded-xl border flex flex-col sm:flex-row sm:items-center justify-between gap-3 transition-all ${
                em.isPrimary
                  ? "border-emerald-500/40 bg-emerald-500/5 dark:bg-emerald-950/20"
                  : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-950/40"
              }`}
            >
              <div className="flex items-center gap-3 min-w-0">
                <div
                  className={`p-1.5 rounded-lg border shrink-0 ${
                    em.isPrimary
                      ? "bg-emerald-500/15 border-emerald-500/30 text-emerald-600 dark:text-emerald-400"
                      : "bg-zinc-100 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 text-zinc-500 dark:text-zinc-400"
                  }`}
                >
                  <Mail size={16} />
                </div>
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-semibold text-xs text-zinc-900 dark:text-zinc-100 truncate">
                      {em.email}
                    </span>
                    {em.isPrimary && (
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold tracking-wide uppercase bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 shrink-0">
                        Primary
                      </span>
                    )}
                  </div>
                  <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                    Added {formatDate(em.createdAt?.seconds)}
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                {em.isPrimary ? (
                  <span className="text-zinc-400 dark:text-zinc-600 text-[11px] italic font-sans">
                    Default login address
                  </span>
                ) : isBanned ? (
                  <span className="text-zinc-400 dark:text-zinc-600 text-[11px] italic font-sans">
                    Read-only (Suspended)
                  </span>
                ) : (
                  <>
                    <button
                      type="button"
                      disabled={emailActionLoading === em.email}
                      onClick={() => handleSetPrimary(em.email)}
                      className="px-2.5 py-1 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-300 text-[11px] font-medium transition inline-flex items-center gap-1 cursor-pointer"
                    >
                      {emailActionLoading === em.email && (
                        <Loader2 size={11} className="animate-spin" />
                      )}
                      <span>Make Primary</span>
                    </button>
                    <button
                      type="button"
                      disabled={emailActionLoading === em.email}
                      onClick={() => handleRemoveEmail(em.email)}
                      aria-label={`Remove ${em.email}`}
                      title="Remove email"
                      className="p-1.5 rounded-lg border border-rose-500/20 text-rose-600 dark:text-rose-400 hover:bg-rose-500/10 transition inline-flex items-center justify-center cursor-pointer"
                    >
                      {emailActionLoading === em.email ? (
                        <Loader2 size={13} className="animate-spin" />
                      ) : (
                        <Trash2 size={13} />
                      )}
                    </button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Section 2: Active Sessions */}
      <section className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-6 space-y-4 shadow-2xs">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-bold flex items-center gap-2">
                <Shield size={16} className="text-blue-500" />
                <span>Active Sessions</span>
              </h2>
              {sessions.length > 0 && (
                <span className="px-2 py-0.2 rounded-full text-[10px] font-mono font-bold bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700">
                  {sessions.length} {sessions.length === 1 ? "Session" : "Sessions"}
                  {activeNowCount > 1 ? ` (${activeNowCount} Active Now)` : ""}
                </span>
              )}
            </div>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
              Devices and browsers currently logged into your account. Multiple devices may be
              active simultaneously.
            </p>
          </div>

          {/* Bulk Action Button */}
          {hasOtherSessions && (
            <button
              type="button"
              onClick={() => {
                setSignOutAllError(null);
                setIsSignOutAllModalOpen(true);
              }}
              className="px-3 py-1.5 rounded-xl text-xs font-semibold text-rose-600 dark:text-rose-400 border border-rose-500/30 bg-rose-500/10 hover:bg-rose-500/20 transition flex items-center gap-1.5 self-start sm:self-auto shadow-2xs cursor-pointer shrink-0"
            >
              <LogOut size={14} />
              <span>Sign Out All Other Sessions</span>
            </button>
          )}
        </div>

        {/* Scrollable Container (> 3 sessions) */}
        <div className="session-scroll-list max-h-[224px] overflow-y-auto pr-1.5 space-y-2">
          {sessions.length === 0 ? (
            <div className="text-xs text-zinc-500 py-3 text-center">
              No active sessions recorded.
            </div>
          ) : (
            sessions.map((sess) => {
              const isCurrent = sess.isCurrent;
              const isMobile = isMobileDevice(sess.deviceHint);
              const isActiveNow =
                isCurrent ||
                (sess.lastActiveAt &&
                  sess.lastActiveAt.seconds &&
                  Date.now() - Number(sess.lastActiveAt.seconds) * 1000 < 5 * 60 * 1000);

              if (isCurrent) {
                return (
                  <div
                    key={sess.uuid}
                    className="px-3.5 py-2.5 rounded-xl border-2 border-emerald-500/40 bg-emerald-500/5 dark:bg-emerald-950/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3 transition-all shadow-2xs"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="p-1.5 rounded-lg bg-emerald-500/15 border border-emerald-500/30 text-emerald-600 dark:text-emerald-400 shrink-0">
                        {isMobile ? <Smartphone size={16} /> : <Laptop size={16} />}
                      </div>
                      <div className="min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-bold text-xs text-zinc-900 dark:text-zinc-100 truncate">
                            {sess.deviceHint || "Unknown Device"}
                          </span>
                          <span className="inline-flex items-center gap-1 px-1.5 py-0.2 rounded text-[9px] font-bold uppercase tracking-wider bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border border-emerald-500/30 shrink-0">
                            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                            This Device
                          </span>
                          <span className="text-xs text-emerald-600 dark:text-emerald-400 font-semibold shrink-0">
                            • Active now
                          </span>
                        </div>
                        <div className="flex items-center gap-1.5 text-[11px] text-zinc-500 dark:text-zinc-400 truncate mt-0.5">
                          {sess.ipAddress && (
                            <>
                              <span className="font-mono text-zinc-700 dark:text-zinc-300">
                                {sess.ipAddress}
                              </span>
                              <span>•</span>
                            </>
                          )}
                          <span className="truncate">
                            {sess.locationHint || "Unknown Location"}
                          </span>
                          {sess.createdAt && sess.createdAt.seconds && (
                            <>
                              <span>•</span>
                              <span className="text-zinc-400 dark:text-zinc-500">
                                Signed in {formatTimestamp(sess.createdAt)}
                              </span>
                            </>
                          )}
                        </div>
                      </div>
                    </div>

                    <div className="shrink-0 flex items-center self-end sm:self-center">
                      <span className="text-[11px] text-zinc-400 italic font-mono px-2 py-0.5">
                        Current
                      </span>
                    </div>
                  </div>
                );
              }

              return (
                <div
                  key={sess.uuid}
                  className="px-3.5 py-2.5 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50/80 dark:bg-zinc-950/40 hover:border-zinc-300 dark:hover:border-zinc-700 flex flex-col sm:flex-row sm:items-center justify-between gap-3 transition-all"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="p-1.5 rounded-lg bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 text-zinc-600 dark:text-zinc-300 shrink-0">
                      {isMobile ? <Smartphone size={16} /> : <Laptop size={16} />}
                    </div>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-semibold text-xs text-zinc-900 dark:text-zinc-100 truncate">
                          {sess.deviceHint || "Unknown Device"}
                        </span>
                        {isMobile && (
                          <span className="px-1.5 py-0.2 rounded text-[9px] font-medium bg-zinc-200/60 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 shrink-0">
                            Mobile
                          </span>
                        )}
                        {isActiveNow ? (
                          <span className="inline-flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400 font-medium shrink-0">
                            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                            Active now
                          </span>
                        ) : (
                          <span className="text-xs text-zinc-400 dark:text-zinc-500 shrink-0">
                            •{" "}
                            {formatSessionActivity(
                              sess.lastActiveAt?.seconds
                                ? new Date(Number(sess.lastActiveAt.seconds) * 1000)
                                : null,
                              false,
                            )}
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-1.5 text-[11px] text-zinc-500 dark:text-zinc-400 truncate mt-0.5">
                        {sess.ipAddress && (
                          <>
                            <span className="font-mono text-zinc-700 dark:text-zinc-300">
                              {sess.ipAddress}
                            </span>
                            <span>•</span>
                          </>
                        )}
                        <span className="truncate">{sess.locationHint || "Unknown Location"}</span>
                        {sess.createdAt && sess.createdAt.seconds && (
                          <>
                            <span>•</span>
                            <span className="text-zinc-400 dark:text-zinc-500">
                              Signed in {formatTimestamp(sess.createdAt)}
                            </span>
                          </>
                        )}
                      </div>
                    </div>
                  </div>

                  <button
                    type="button"
                    onClick={() => {
                      setRevokeSessionError(null);
                      setSessionToRevoke(sess);
                    }}
                    className="px-2.5 py-1 rounded-lg text-xs font-semibold text-zinc-700 dark:text-zinc-300 hover:text-rose-600 dark:hover:text-rose-400 border border-zinc-300 dark:border-zinc-700 hover:border-rose-400 dark:hover:border-rose-500/50 bg-white dark:bg-zinc-900 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition flex items-center gap-1 shadow-2xs cursor-pointer self-end sm:self-center shrink-0"
                  >
                    <X size={12} />
                    <span>Revoke</span>
                  </button>
                </div>
              );
            })
          )}
        </div>
      </section>

      {/* Section 3: Profile Details Form */}
      <form
        onSubmit={handleSave}
        className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-6 space-y-6 shadow-2xs"
      >
        <h2 className="text-sm font-bold flex items-center gap-2">
          <UserIcon size={16} className="text-blue-500" />
          <span>Profile Details</span>
        </h2>

        {errorMsg && (
          <div className="p-3.5 rounded-xl border border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400 text-xs">
            {errorMsg}
          </div>
        )}

        <div className="space-y-6">
          {/* Avatar Image URL */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                Avatar Image URL (Optional)
              </label>
              {isAvatarModified && (
                <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                  Modified
                </span>
              )}
            </div>
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-full shrink-0 overflow-hidden flex items-center justify-center font-bold text-sm bg-blue-500/10 text-blue-500 border border-zinc-200 dark:border-zinc-800 shadow-xs">
                {avatarUrl ? (
                  <img
                    src={avatarUrl}
                    alt={user?.id || "Avatar"}
                    className="w-full h-full object-cover"
                    onError={(e) => {
                      (e.currentTarget as HTMLElement).style.display = "none";
                    }}
                  />
                ) : (
                  <span>{initial}</span>
                )}
              </div>
              <input
                type="url"
                placeholder="https://example.com/avatar.jpg"
                value={avatarUrl}
                onChange={(e) => setAvatarUrl(e.target.value)}
                className={`flex-1 px-3.5 py-2.5 rounded-xl border text-xs text-zinc-900 dark:text-zinc-100 focus:outline-none transition-colors ${
                  isAvatarModified
                    ? "border-amber-500/60 dark:border-amber-500/50 bg-amber-500/5 ring-1 ring-amber-500/30 focus:ring-2 focus:ring-amber-500"
                    : "border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 focus:ring-2 focus:ring-blue-500"
                }`}
              />
            </div>
            <p className="text-[11px] text-zinc-400 mt-1">
              Leave blank to automatically generate an initial with adaptive category chip colors.
            </p>
          </div>

          {/* Bio */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="text-xs font-medium text-zinc-700 dark:text-zinc-300">Bio</label>
              {isBioModified && (
                <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                  Modified
                </span>
              )}
            </div>
            <textarea
              rows={3}
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="Tell other users about your flying style, favorite frames, and builds..."
              className={`w-full px-3.5 py-2.5 rounded-xl border text-xs text-zinc-900 dark:text-zinc-100 focus:outline-none transition-colors ${
                isBioModified
                  ? "border-amber-500/60 dark:border-amber-500/50 bg-amber-500/5 ring-1 ring-amber-500/30 focus:ring-2 focus:ring-amber-500"
                  : "border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 focus:ring-2 focus:ring-blue-500"
              }`}
            />
          </div>
        </div>

        {/* Save Button Row */}
        <div className="flex items-center justify-between pt-4 border-t border-zinc-200 dark:border-zinc-800">
          <div className="flex items-center gap-2">
            {isDirty ? (
              <span className="text-xs text-amber-600 dark:text-amber-400 font-medium flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-amber-500 animate-pulse" />
                <span>Unsaved changes</span>
              </span>
            ) : saveSuccess ? (
              <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                <Check size={14} />
                <span>Saved successfully</span>
              </span>
            ) : (
              <span className="text-xs text-zinc-400 dark:text-zinc-500">No unsaved changes</span>
            )}
          </div>

          <button
            type="submit"
            disabled={!isDirty || isSaving}
            className="px-5 py-2 rounded-xl text-xs font-semibold text-white bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed transition shadow-sm flex items-center gap-1.5"
          >
            {isSaving && <Loader2 size={14} className="animate-spin" />}
            <span>Save</span>
          </button>
        </div>
      </form>

      {/* Section 3: Privacy */}
      <section className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-6 space-y-4 shadow-2xs">
        <div className="flex items-center justify-between gap-4">
          <h2 className="text-sm font-bold flex items-center gap-2">
            <Lock size={16} className="text-blue-500" />
            <span>Privacy</span>
          </h2>

          <button
            type="button"
            data-testid="download-data-button"
            onClick={handleOpenExportModal}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-blue-600 hover:bg-blue-500 transition shadow-sm flex items-center gap-1.5 cursor-pointer shrink-0"
          >
            <span>Download My Data</span>
          </button>
        </div>

        {exportSuccessMessage && (
          <div className="p-3.5 rounded-xl border border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 text-xs flex items-center gap-2">
            <CheckCircle2 size={16} className="text-emerald-500 shrink-0" />
            <span>{exportSuccessMessage}</span>
          </div>
        )}
      </section>

      {/* Section 4: Danger Zone */}
      <section className="rounded-2xl border border-rose-200 dark:border-rose-900/40 bg-rose-50/20 dark:bg-rose-950/10 p-6 space-y-4 shadow-2xs">
        <div className="flex items-center justify-between gap-4 flex-wrap">
          <div>
            <h2 className="text-sm font-bold text-rose-700 dark:text-rose-400 flex items-center gap-2">
              <Trash2 size={16} className="text-rose-500" />
              <span>Danger Zone</span>
            </h2>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
              Permanently delete your Quadsmith account and personal data after a 30-day grace
              period.
            </p>
          </div>

          <button
            type="button"
            onClick={() => setIsDeleteModalOpen(true)}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 transition shadow-sm flex items-center gap-1.5 cursor-pointer shrink-0"
          >
            <Trash2 size={14} />
            <span>Delete Account</span>
          </button>
        </div>
      </section>

      {/* Modal: Voluntary Account Deletion Dialog */}
      {isDeleteModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
          <div className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl p-6 space-y-5 relative">
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-2.5 text-rose-600 dark:text-rose-400">
                <div className="p-2 rounded-xl bg-rose-500/10 border border-rose-500/20">
                  <AlertTriangle size={20} />
                </div>
                <div>
                  <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                    Delete Account
                  </h3>
                  <p className="text-[11px] text-zinc-500 dark:text-zinc-400">
                    This action will begin the deletion process
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setIsDeleteModalOpen(false)}
                className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 cursor-pointer"
              >
                <X size={18} />
              </button>
            </div>

            <div className="p-3.5 rounded-xl border border-amber-500/20 bg-amber-500/10 text-amber-800 dark:text-amber-300 text-xs space-y-1">
              <p className="font-semibold">30-day grace period</p>
              <p className="text-[11px] opacity-90">
                Your account enters a 30 days grace period before being permanently deleted. You can
                log in at any time during this grace period to cancel the deletion request and
                recover your account.
              </p>
            </div>

            <div className="space-y-2">
              <label className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                To confirm, type your username{" "}
                <span className="font-mono font-bold text-zinc-900 dark:text-zinc-100">
                  @{user?.id}
                </span>
                :
              </label>
              <input
                type="text"
                placeholder={user?.id || "username"}
                value={deleteConfirmHandle}
                onChange={(e) => setDeleteConfirmHandle(e.target.value)}
                className="w-full px-3 py-2 text-xs rounded-xl border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 font-mono focus:outline-none focus:ring-2 focus:ring-rose-500 text-zinc-900 dark:text-zinc-100"
              />
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setIsDeleteModalOpen(false)}
                className="px-3.5 py-2 text-xs font-medium rounded-xl border border-zinc-200 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={deleteConfirmHandle.trim().toLowerCase() !== user?.id?.toLowerCase()}
                className="px-3.5 py-2 text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 disabled:opacity-40 disabled:cursor-not-allowed rounded-xl transition cursor-pointer"
              >
                Confirm Account Deletion
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Modal: Add Email Address Dialog (mirrors login flow) */}
      {isAddEmailOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
          <div className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl p-8 space-y-6 relative">
            {/* Close button in top-right */}
            <button
              type="button"
              onClick={closeAddEmailModal}
              className="absolute top-4 right-4 p-1.5 rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 transition cursor-pointer"
              aria-label="Close"
            >
              <X size={18} />
            </button>

            {addStep === "email" ? (
              <>
                {/* Header Logo & Title */}
                <div className="text-center space-y-3">
                  <div className="flex justify-center">
                    <img
                      src={currentTheme === "dark" ? logoDark : logoLight}
                      alt="Quadsmith"
                      className="h-10 w-auto"
                    />
                  </div>
                  <h1 className="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
                    Add Email Address
                  </h1>
                  <p className="text-xs text-zinc-500 dark:text-zinc-400">
                    Associate a new email address with your Quadsmith account.
                  </p>
                </div>

                <div className="space-y-4">
                  {addEmailError && (
                    <div
                      data-testid="otp-error-banner"
                      className="p-3.5 rounded-xl border border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400 text-xs flex items-start gap-2.5"
                    >
                      {isAddRateLimited ? (
                        <Lock size={16} className="text-red-500 shrink-0 mt-0.5" />
                      ) : (
                        <AlertTriangle size={16} className="text-red-500 shrink-0 mt-0.5" />
                      )}
                      <div className="space-y-0.5 flex-1">
                        <p className="font-semibold">{addEmailError}</p>
                      </div>
                    </div>
                  )}

                  <form onSubmit={handleSendAddCode} className="space-y-4">
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
                          value={newEmail}
                          onChange={(e) => setNewEmail(e.target.value)}
                          className="w-full pl-9 pr-3 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 text-sm text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      </div>
                    </div>

                    {(() => {
                      const isSendDisabled =
                        isSendingAddOtp ||
                        !newEmail.trim() ||
                        addCooldown > 0 ||
                        (isAddRateLimited && addRateLimitCooldown > 0);

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
                          {isSendingAddOtp ? (
                            <>
                              <Loader2 size={16} className="animate-spin" />
                              <span>Sending code...</span>
                            </>
                          ) : isAddRateLimited && addRateLimitCooldown > 0 ? (
                            <span>
                              Send Verification Code ({Math.floor(addRateLimitCooldown / 60)}m
                              {addRateLimitCooldown % 60}s)
                            </span>
                          ) : addCooldown > 0 ? (
                            <span>Send Verification Code ({addCooldown}s)</span>
                          ) : !newEmail.trim() ? (
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
              </>
            ) : (
              <>
                {/* Step 2: OTP Verification Screen */}
                <div className="space-y-6">
                  {/* Header */}
                  <div className="text-center space-y-3">
                    <div className="flex justify-center">
                      <img
                        src={currentTheme === "dark" ? logoDark : logoLight}
                        alt="Quadsmith"
                        className="h-10 w-auto"
                      />
                    </div>
                    <h1 className="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
                      Verify your email
                    </h1>
                    <p className="text-xs text-zinc-500 dark:text-zinc-400">
                      We sent a single-use verification code to {newEmail}
                    </p>
                  </div>

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
                      {addRemainingAttempts !== null &&
                        addRemainingAttempts > 0 &&
                        !isAddExhausted &&
                        !isAddRateLimited && (
                          <span
                            className={`text-[10px] font-mono font-bold ${
                              addRemainingAttempts === 1 ? "text-rose-500" : "text-amber-500"
                            }`}
                          >
                            {addRemainingAttempts} attempt
                            {addRemainingAttempts === 1 ? "" : "s"} left
                          </span>
                        )}
                    </div>

                    <div className="flex items-center justify-between gap-2">
                      {addOtpDigits.map((digit, index) => (
                        <input
                          key={index}
                          ref={(el) => {
                            addOtpInputRefs.current[index] = el;
                          }}
                          type="text"
                          inputMode="numeric"
                          pattern="[0-9]*"
                          autoComplete={index === 0 ? "one-time-code" : "off"}
                          maxLength={1}
                          placeholder={addHintDigits[index] || ""}
                          value={digit}
                          disabled={isAddExhausted || isAddRateLimited || isVerifyingAddOtp}
                          onChange={(e) => handleAddOtpChange(index, e.target.value)}
                          onKeyDown={(e) => handleAddKeyDown(index, e)}
                          onPaste={handleAddPaste}
                          className={`w-12 h-14 text-center font-mono text-2xl font-bold rounded-xl border bg-zinc-50 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-300 dark:placeholder:text-zinc-700 focus:outline-none transition ${
                            isAddExhausted || isAddRateLimited
                              ? "opacity-40 border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 text-zinc-400 dark:text-zinc-600 cursor-not-allowed"
                              : "border-zinc-300 dark:border-zinc-700 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
                          }`}
                        />
                      ))}
                    </div>
                  </div>

                  {/* Error / Attempt Warning Banner */}
                  {addEmailError && (
                    <div
                      data-testid="otp-error-banner"
                      className={`p-3.5 rounded-xl border text-xs flex items-start gap-2.5 ${
                        isAddExhausted || isAddRateLimited
                          ? "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
                          : addRemainingAttempts === 2
                            ? "border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300"
                            : addRemainingAttempts === 1
                              ? "border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-400"
                              : "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
                      }`}
                    >
                      {isAddExhausted || isAddRateLimited ? (
                        <Lock size={16} className="text-red-500 shrink-0 mt-0.5" />
                      ) : addRemainingAttempts === 2 ? (
                        <AlertTriangle size={16} className="text-amber-500 shrink-0 mt-0.5" />
                      ) : (
                        <AlertTriangle size={16} className="text-rose-500 shrink-0 mt-0.5" />
                      )}
                      <div className="space-y-0.5 flex-1">
                        <p className="font-semibold">{addEmailError}</p>
                        {addRemainingAttempts !== null && addRemainingAttempts > 0 && (
                          <p className="text-[11px] opacity-90">
                            {addRemainingAttempts === 1
                              ? "Final attempt remaining before this code is invalidated."
                              : `${addRemainingAttempts} attempts remaining for this verification challenge.`}
                          </p>
                        )}
                      </div>
                    </div>
                  )}

                  {/* Action Buttons & Status */}
                  <div className="space-y-2.5">
                    {isVerifyingAddOtp && (
                      <div className="flex items-center justify-center gap-2 py-1.5 text-xs font-medium text-blue-600 dark:text-blue-400">
                        <Loader2 size={16} className="animate-spin" />
                        <span>Verifying code...</span>
                      </div>
                    )}

                    {(isAddExhausted || isAddRateLimited) &&
                      (() => {
                        const isReqDisabled =
                          (isAddRateLimited
                            ? addRateLimitCooldown > 0
                            : addExhaustedCooldown > 0) || isSendingAddOtp;
                        return (
                          <button
                            type="button"
                            onClick={() => handleSendAddCode()}
                            disabled={isReqDisabled}
                            className={`w-full py-2.5 rounded-xl font-semibold text-xs transition shadow-sm flex items-center justify-center gap-2 ${
                              isReqDisabled
                                ? "text-zinc-400 dark:text-zinc-500 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-not-allowed opacity-60"
                                : "text-white bg-blue-600 hover:bg-blue-500 active:bg-blue-700 cursor-pointer"
                            }`}
                          >
                            <span>
                              {isAddRateLimited
                                ? addRateLimitCooldown > 0
                                  ? `Request New Verification Code (${Math.floor(addRateLimitCooldown / 60)}m${addRateLimitCooldown % 60}s)`
                                  : "Request New Verification Code"
                                : addExhaustedCooldown > 0
                                  ? `Request New Verification Code (${addExhaustedCooldown}s)`
                                  : "Request New Verification Code"}
                            </span>
                          </button>
                        );
                      })()}

                    {/* Back Option */}
                    <div className="flex items-center text-xs pt-1">
                      <button
                        type="button"
                        onClick={() => {
                          setAddStep("email");
                          setAddEmailError(null);
                          setAddOtpDigits(["", "", "", "", "", ""]);
                          setAddHintDigits(["", "", "", "", "", ""]);
                          setIsAddExhausted(false);
                          setIsAddRateLimited(false);
                        }}
                        className="text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 flex items-center gap-1 transition cursor-pointer"
                      >
                        <ArrowLeft size={13} />
                        <span>Back</span>
                      </button>
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Modal: Account Merging Dialog (triggered when an existing account email is entered) */}
      {isMergeModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
          <div className="max-w-lg w-full rounded-2xl border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 shadow-2xl p-6 space-y-6">
            {/* Modal Header */}
            <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-800 pb-4">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-purple-500/20 text-purple-600 dark:text-purple-400 flex items-center justify-center font-bold text-sm">
                  <ArrowRightLeft size={16} />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-zinc-900 dark:text-zinc-100">
                    {mergeModalStep === "summary" ? "Confirm Account Merge" : "Accounts Merged"}
                  </h3>
                  <p className="text-[11px] text-zinc-500 dark:text-zinc-400">
                    {mergeModalStep === "summary"
                      ? `Consolidate @${mergeOtherUsername} into @${user?.id}`
                      : "Consolidation complete"}
                  </p>
                </div>
              </div>
              <button
                onClick={() => setIsMergeModalOpen(false)}
                className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 text-sm font-bold"
              >
                <X size={16} />
              </button>
            </div>

            {mergeError && (
              <div className="p-3 rounded-xl border border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400 text-xs">
                {mergeError}
              </div>
            )}

            {mergeModalStep === "summary" && (
              <div className="space-y-4">
                {/* Ownership Verified Badge */}
                <div className="flex items-center gap-2 p-2.5 rounded-xl border border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 text-xs font-medium">
                  <CheckCircle2
                    size={15}
                    className="text-emerald-600 dark:text-emerald-400 shrink-0"
                  />
                  <span>
                    Ownership verified for <strong>@{mergeOtherUsername}</strong>
                  </span>
                </div>

                <div className="text-xs text-zinc-600 dark:text-zinc-300">
                  Please review the summary below of everything that will happen upon merging:
                </div>

                {/* Summary Cards */}
                <div className="space-y-2.5">
                  {/* Item 1: Deleting other username */}
                  <div className="p-3 rounded-xl border border-red-500/20 bg-red-500/5 flex items-start gap-3">
                    <div className="w-7 h-7 rounded-lg bg-red-500/15 text-red-600 dark:text-red-400 flex items-center justify-center shrink-0 mt-0.5">
                      <UserX size={15} />
                    </div>
                    <div className="space-y-0.5 text-xs">
                      <div className="font-semibold text-zinc-900 dark:text-zinc-100">
                        Delete Username:{" "}
                        <span className="font-mono text-red-600 dark:text-red-400">
                          @{mergeOtherUsername}
                        </span>
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 leading-relaxed">
                        The username{" "}
                        <strong className="text-zinc-700 dark:text-zinc-300">
                          @{mergeOtherUsername}
                        </strong>{" "}
                        will be held for 30 days before being released. During this 30-day window,
                        existing profile links will automatically redirect to{" "}
                        <strong className="text-zinc-700 dark:text-zinc-300">@{user?.id}</strong>.
                      </div>
                    </div>
                  </div>

                  {/* Item 2: Associating secondary email */}
                  <div className="p-3 rounded-xl border border-blue-500/20 bg-blue-500/5 flex items-start gap-3">
                    <div className="w-7 h-7 rounded-lg bg-blue-500/15 text-blue-600 dark:text-blue-400 flex items-center justify-center shrink-0 mt-0.5">
                      <Mail size={15} />
                    </div>
                    <div className="space-y-0.5 text-xs">
                      <div className="font-semibold text-zinc-900 dark:text-zinc-100">
                        {mergeOtherEmails.length > 1
                          ? `Associate ${mergeOtherEmails.length} Email Addresses`
                          : "Associate Email Address"}
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 leading-relaxed">
                        {mergeOtherEmails.length > 1 ? (
                          <>
                            All associated emails (
                            <strong className="font-mono text-zinc-800 dark:text-zinc-200">
                              {mergeOtherEmails.join(", ")}
                            </strong>
                            ) will be added as verified login emails under your account (
                            <strong className="text-zinc-700 dark:text-zinc-300">
                              @{user?.id}
                            </strong>
                            ).
                          </>
                        ) : (
                          <>
                            <strong className="font-mono text-zinc-800 dark:text-zinc-200">
                              {mergeEmail}
                            </strong>{" "}
                            will be added as a verified login email under your account (
                            <strong className="text-zinc-700 dark:text-zinc-300">
                              @{user?.id}
                            </strong>
                            ).
                          </>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Item 3: Migrating resources (builds, OAuth, etc.) */}
                  <div className="p-3 rounded-xl border border-purple-500/20 bg-purple-500/5 flex items-start gap-3">
                    <div className="w-7 h-7 rounded-lg bg-purple-500/15 text-purple-600 dark:text-purple-400 flex items-center justify-center shrink-0 mt-0.5">
                      <Package size={15} />
                    </div>
                    <div className="space-y-1 text-xs">
                      <div className="font-semibold text-zinc-900 dark:text-zinc-100">
                        Migrated Resources ({mergeOtherBuildCount} Drone{" "}
                        {mergeOtherBuildCount === 1 ? "Build" : "Builds"})
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 leading-relaxed">
                        All {mergeOtherBuildCount} drone configurations, specs, and parts will be
                        transferred to your profile.
                      </div>
                      {mergeOtherBuildNames && mergeOtherBuildNames.length > 0 && (
                        <div className="flex flex-wrap gap-1.5 pt-1">
                          {mergeOtherBuildNames.map((name, i) => (
                            <span
                              key={i}
                              className="px-2 py-0.5 rounded-md bg-purple-500/10 text-purple-700 dark:text-purple-300 text-[10px] font-medium border border-purple-500/20"
                            >
                              {name}
                            </span>
                          ))}
                        </div>
                      )}
                      {mergeOtherOauthProviders && mergeOtherOauthProviders.length > 0 && (
                        <div className="text-[11px] text-zinc-500 dark:text-zinc-400 pt-0.5">
                          Linked OAuth providers:{" "}
                          <strong className="capitalize text-zinc-700 dark:text-zinc-300">
                            {mergeOtherOauthProviders.join(", ")}
                          </strong>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Permanent Warning */}
                  <div className="p-3 rounded-xl border border-amber-500/30 bg-amber-500/10 flex items-start gap-2.5 text-amber-700 dark:text-amber-300 text-xs">
                    <AlertTriangle
                      size={15}
                      className="shrink-0 mt-0.5 text-amber-600 dark:text-amber-400"
                    />
                    <div className="text-[11px] leading-relaxed">
                      This action is permanent and cannot be undone. Once merged,{" "}
                      <strong className="text-zinc-900 dark:text-zinc-100">
                        @{mergeOtherUsername}
                      </strong>{" "}
                      will be completely closed and cannot be restored.
                    </div>
                  </div>
                </div>

                {/* Footer Buttons */}
                <div className="space-y-2.5 pt-2">
                  <div className="flex gap-2.5">
                    <button
                      type="button"
                      onClick={() => setIsMergeModalOpen(false)}
                      disabled={isExecutingMerge}
                      className="flex-1 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-300 text-xs font-semibold transition"
                    >
                      Cancel
                    </button>
                    <button
                      type="button"
                      onClick={handleConfirmMerge}
                      disabled={isExecutingMerge}
                      className="px-6 py-2.5 rounded-xl bg-red-600 hover:bg-red-500 active:bg-red-700 disabled:opacity-50 text-white text-xs font-semibold transition flex items-center justify-center gap-1.5 shadow-sm shadow-red-500/20 shrink-0"
                    >
                      {isExecutingMerge && <Loader2 size={13} className="animate-spin" />}
                      <span>Merge</span>
                    </button>
                  </div>
                </div>
              </div>
            )}

            {mergeModalStep === "success" && (
              /* Success State */
              <div className="text-center py-4 space-y-3">
                <div className="w-12 h-12 rounded-full bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 mx-auto flex items-center justify-center text-xl font-bold">
                  ✓
                </div>
                <h4 className="text-sm font-bold text-zinc-900 dark:text-zinc-100">
                  Accounts Merged Successfully!
                </h4>
                <p className="text-xs text-zinc-600 dark:text-zinc-400 max-w-sm mx-auto">
                  {transferredBuilds > 0
                    ? `All ${transferredBuilds} builds from @${mergeOtherUsername} have been reassigned to your account, and `
                    : `The account @${mergeOtherUsername} has been consolidated into your profile, and `}
                  {mergeOtherEmails.length > 1 ? (
                    <>
                      all {mergeOtherEmails.length} email addresses (
                      <span className="font-mono text-zinc-900 dark:text-zinc-200">
                        {mergeOtherEmails.join(", ")}
                      </span>
                      ) have been added to your login emails.
                    </>
                  ) : (
                    <>
                      <span className="font-mono text-zinc-900 dark:text-zinc-200">
                        {mergeEmail}
                      </span>{" "}
                      has been added to your login emails.
                    </>
                  )}
                </p>
                <button
                  type="button"
                  onClick={() => {
                    setIsMergeModalOpen(false);
                    setNewEmail("");
                  }}
                  className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold transition shadow-sm"
                >
                  Done
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Modal: Change Username Dialog */}
      {isChangeUsernameOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
          <div
            role="dialog"
            aria-label="Change Username"
            className="max-w-lg w-full rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-2xl p-6 space-y-5 text-zinc-900 dark:text-zinc-100"
          >
            {/* Modal Header */}
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-xl bg-blue-500/10 text-blue-600 dark:text-blue-400 flex items-center justify-center shrink-0 border border-blue-500/20">
                  <UserIcon size={18} />
                </div>
                <div>
                  <h3 className="text-base font-bold text-zinc-900 dark:text-zinc-100">
                    Change Username
                  </h3>
                  <p className="text-xs text-zinc-500 dark:text-zinc-400">
                    Current:{" "}
                    <strong className="font-mono text-zinc-800 dark:text-zinc-200">
                      @{user?.id}
                    </strong>
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setIsChangeUsernameOpen(false)}
                className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 p-1 rounded-lg transition cursor-pointer"
                aria-label="Close"
              >
                <X size={18} />
              </button>
            </div>

            {/* Error Banner */}
            {usernameError && (
              <div className="p-3 rounded-xl border border-rose-500/30 bg-rose-500/10 text-rose-600 dark:text-rose-400 text-xs flex items-start gap-2">
                <AlertCircle size={15} className="shrink-0 mt-0.5" />
                <div className="leading-relaxed text-[11px]">{usernameError}</div>
              </div>
            )}

            {/* Success Banner */}
            {usernameSuccess && (
              <div className="p-3 rounded-xl border border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 text-xs flex items-start gap-2">
                <CheckCircle2 size={15} className="shrink-0 mt-0.5" />
                <div className="leading-relaxed text-[11px] font-medium">{usernameSuccess}</div>
              </div>
            )}

            {/* Loading Archived Usernames */}
            {isLoadingArchived && (
              <div className="flex items-center gap-2 p-3 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 text-xs text-zinc-500">
                <Loader2 size={14} className="animate-spin text-blue-500" />
                <span>Checking your held usernames...</span>
              </div>
            )}

            {/* SECTION A: Held Archived Usernames (Relaxed Rules) */}
            {archivedUsernames.length > 0 && (
              <div className="p-4 rounded-xl border border-purple-500/20 bg-purple-500/5 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="text-xs font-bold text-purple-700 dark:text-purple-300 flex items-center gap-1.5">
                    <span>Your Held Usernames (30d Hold)</span>
                  </div>
                  <span className="text-[10px] text-zinc-500 dark:text-zinc-400 font-mono">
                    {archivedUsernames.length}{" "}
                    {archivedUsernames.length === 1 ? "username" : "usernames"} reserved
                  </span>
                </div>

                {archivedUsernames.map((item) => (
                  <div
                    key={item.username}
                    className="p-3 rounded-lg border border-purple-500/30 bg-zinc-50 dark:bg-zinc-950/80 flex items-center justify-between gap-3"
                  >
                    <div className="space-y-0.5">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-bold text-sm text-zinc-900 dark:text-zinc-100">
                          @{item.username}
                        </span>
                        {item.canReclaimNow ? (
                          <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30">
                            Reclaim Available
                          </span>
                        ) : (
                          <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30">
                            Cooldown: Available in {item.daysRemaining}d
                          </span>
                        )}
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400">
                        Held until{" "}
                        <span className="text-zinc-700 dark:text-zinc-300 font-medium">
                          {formatTimestamp(item.releaseAt)}
                        </span>{" "}
                        ({item.daysRemaining} days remaining before release)
                      </div>
                    </div>

                    <button
                      type="button"
                      disabled={!item.canReclaimNow || isSubmittingUsername}
                      onClick={() => handleReclaimUsername(item.username)}
                      className="px-3 py-1.5 rounded-xl bg-purple-600 hover:bg-purple-500 disabled:opacity-50 disabled:pointer-events-none text-white text-xs font-semibold transition shrink-0 shadow-sm cursor-pointer"
                    >
                      {isSubmittingUsername ? (
                        <Loader2 size={13} className="animate-spin" />
                      ) : (
                        "Reclaim"
                      )}
                    </button>
                  </div>
                ))}

                <p className="text-[11px] text-purple-700 dark:text-purple-300/80 leading-relaxed">
                  ⚡ <strong>Relaxed Reclaim:</strong> You can revert to your archived username
                  within the 24-hour grace period, or every 7 days thereafter, without waiting for
                  the normal 30-day cooldown.
                </p>
              </div>
            )}

            {/* SECTION B: Choose a Brand-New Username */}
            <form onSubmit={handleSubmitNewUsername} className="space-y-4">
              <div className="space-y-2.5">
                <div className="flex items-center justify-between">
                  <label className="block text-xs font-semibold text-zinc-800 dark:text-zinc-200">
                    Choose a Brand-New Username
                  </label>
                  {!isNewUsernameOnCooldown && (
                    <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">
                      ✓ Normal 30d cooldown ready
                    </span>
                  )}
                </div>

                <div className="relative">
                  <div
                    data-testid="username-input-container"
                    className={`flex items-center rounded-xl border px-3.5 py-2.5 transition ${
                      usernameAvailability.status === "available"
                        ? "border-emerald-500/80 bg-emerald-500/5 focus-within:ring-2 focus-within:ring-emerald-500"
                        : usernameAvailability.status === "invalid"
                          ? "border-rose-500 bg-rose-500/5 focus-within:ring-2 focus-within:ring-rose-500"
                          : usernameAvailability.status === "unavailable"
                            ? "border-amber-500 bg-amber-500/5 focus-within:ring-2 focus-within:ring-amber-500"
                            : "border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 focus-within:border-blue-500 focus-within:ring-2 focus-within:ring-blue-500/20"
                    }`}
                  >
                    <span className="font-mono text-zinc-400 dark:text-zinc-500 text-xs mr-1 select-none">
                      @
                    </span>
                    <input
                      type="text"
                      disabled={isNewUsernameOnCooldown || isSubmittingUsername}
                      value={newUsernameInput}
                      onChange={(e) => setNewUsernameInput(e.target.value.toLowerCase().trim())}
                      placeholder="new_username"
                      maxLength={20}
                      aria-invalid={
                        usernameAvailability.status === "invalid" ||
                        usernameAvailability.status === "unavailable"
                      }
                      className="bg-transparent text-xs font-mono text-zinc-900 dark:text-zinc-100 outline-none w-full disabled:opacity-50"
                    />
                    {usernameAvailability.status === "checking" && (
                      <Loader2 size={13} className="animate-spin text-zinc-400 shrink-0 ml-2" />
                    )}
                    {usernameAvailability.status === "available" && (
                      <span className="text-[11px] font-medium text-emerald-600 dark:text-emerald-400 shrink-0 ml-2">
                        ✓ Available
                      </span>
                    )}
                    {usernameAvailability.status === "unavailable" &&
                      usernameAvailability.message && (
                        <span className="text-[11px] font-medium text-amber-600 dark:text-amber-400 shrink-0 ml-2">
                          {usernameAvailability.message}
                        </span>
                      )}
                    {usernameAvailability.status === "error" && usernameAvailability.message && (
                      <span className="text-[11px] font-medium text-rose-600 dark:text-rose-400 shrink-0 ml-2">
                        {usernameAvailability.message}
                      </span>
                    )}
                  </div>
                </div>

                {/* Validation / Guidelines */}
                <div className="text-[11px] text-zinc-500 dark:text-zinc-400 space-y-1">
                  <div>• 3-20 characters: lowercase letters, numbers, and single underscores.</div>
                  <div>
                    • Changing to a brand-new username requires a <strong>30-day cooldown</strong>{" "}
                    before you can change to another new username.
                  </div>
                </div>

                {/* Cooldown Warning Banner (if cooldown active) */}
                {isNewUsernameOnCooldown && (
                  <div className="p-3 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300 text-xs flex items-start gap-2">
                    <span className="shrink-0">⏳</span>
                    <div className="leading-relaxed text-[11px]">
                      <strong>Cooldown Active:</strong> You renamed your username recently. You can
                      choose a brand-new username in{" "}
                      <strong>
                        {newUsernameCooldownDays} day{newUsernameCooldownDays === 1 ? "" : "s"}
                      </strong>{" "}
                      (
                      {nextAllowedNewRenameAt?.toLocaleDateString("en-US", {
                        month: "short",
                        day: "numeric",
                        year: "numeric",
                      })}
                      ), or reclaim one of your held usernames above.
                    </div>
                  </div>
                )}
              </div>

              {/* Hold Notice */}
              <div className="p-3 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 text-xs text-zinc-500 dark:text-zinc-400 flex items-start gap-2.5">
                <span className="text-blue-500 text-base leading-none mt-0.5">ℹ</span>
                <div className="text-[11px] leading-relaxed">
                  Your current username{" "}
                  <strong className="text-zinc-800 dark:text-zinc-200">@{user?.id}</strong> will be
                  reserved in your username history for <strong>30 days</strong> and automatically
                  redirect to your profile. An email alert will be sent 7 days before it is
                  released.
                </div>
              </div>

              {/* Footer Buttons */}
              <div className="flex gap-2.5 pt-2">
                <button
                  type="button"
                  onClick={() => setIsChangeUsernameOpen(false)}
                  className="flex-1 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-zinc-100 dark:bg-zinc-800 hover:bg-zinc-200 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-300 text-xs font-semibold transition cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={
                    isNewUsernameOnCooldown ||
                    usernameAvailability.status !== "available" ||
                    isSubmittingUsername ||
                    !newUsernameInput.trim()
                  }
                  className="px-6 py-2.5 rounded-xl bg-red-600 hover:bg-red-500 active:bg-red-700 disabled:opacity-50 text-white text-xs font-semibold transition shrink-0 shadow-sm flex items-center justify-center gap-1.5 cursor-pointer"
                >
                  {isSubmittingUsername && <Loader2 size={13} className="animate-spin" />}
                  <span>Change Username</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Modal: Export Data OTP Verification Dialog */}
      {isExportModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
          <div className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl p-8 space-y-6 relative">
            <button
              type="button"
              onClick={() => setIsExportModalOpen(false)}
              className="absolute top-4 right-4 p-1.5 rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 transition cursor-pointer"
              aria-label="Close"
            >
              <X size={18} />
            </button>

            {isExportSending ? (
              <div className="text-center py-8 space-y-4">
                <Loader2 size={32} className="animate-spin text-blue-500 mx-auto" />
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  Sending verification challenge to your primary email...
                </p>
              </div>
            ) : (
              <>
                <div className="text-center space-y-3">
                  <div className="flex justify-center">
                    <img
                      src={currentTheme === "dark" ? logoDark : logoLight}
                      alt="Quadsmith"
                      className="h-10 w-auto"
                    />
                  </div>
                  <h1 className="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
                    Verify your email
                  </h1>
                  <p className="text-xs text-zinc-500 dark:text-zinc-400">
                    We sent a single-use verification code to{" "}
                    {exportPrimaryEmail || "your primary email"}
                  </p>
                </div>

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

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <label className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                      Verification Code
                    </label>
                    {exportRemainingAttempts !== null &&
                      exportRemainingAttempts > 0 &&
                      !isExportExhausted &&
                      !isExportRateLimited && (
                        <span
                          className={`text-[10px] font-mono font-bold ${
                            exportRemainingAttempts === 1 ? "text-rose-500" : "text-amber-500"
                          }`}
                        >
                          {exportRemainingAttempts} attempt
                          {exportRemainingAttempts === 1 ? "" : "s"} left
                        </span>
                      )}
                  </div>

                  <div className="flex items-center justify-between gap-2">
                    {exportDigits.map((digit, index) => (
                      <input
                        key={index}
                        ref={(el) => {
                          exportInputRefs.current[index] = el;
                        }}
                        type="text"
                        inputMode="numeric"
                        pattern="[0-9]*"
                        autoComplete={index === 0 ? "one-time-code" : "off"}
                        maxLength={1}
                        value={digit}
                        disabled={isExportExhausted || isExportRateLimited || isExportVerifying}
                        onChange={(e) => handleExportOtpChange(index, e.target.value)}
                        onKeyDown={(e) => handleExportKeyDown(index, e)}
                        onPaste={handleExportPaste}
                        className={`w-12 h-14 text-center font-mono text-2xl font-bold rounded-xl border bg-zinc-50 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-300 dark:placeholder:text-zinc-700 focus:outline-none transition ${
                          isExportExhausted || isExportRateLimited
                            ? "opacity-40 border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 text-zinc-400 dark:text-zinc-600 cursor-not-allowed"
                            : "border-zinc-300 dark:border-zinc-700 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
                        }`}
                      />
                    ))}
                  </div>
                </div>

                {exportError && (
                  <div
                    data-testid="otp-error-banner"
                    className={`p-3.5 rounded-xl border text-xs flex items-start gap-2.5 ${
                      isExportExhausted || isExportRateLimited
                        ? "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
                        : exportRemainingAttempts === 2
                          ? "border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300"
                          : exportRemainingAttempts === 1
                            ? "border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-400"
                            : "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400"
                    }`}
                  >
                    {isExportExhausted || isExportRateLimited ? (
                      <Lock size={16} className="text-red-500 shrink-0 mt-0.5" />
                    ) : exportRemainingAttempts === 2 ? (
                      <AlertTriangle size={16} className="text-amber-500 shrink-0 mt-0.5" />
                    ) : (
                      <AlertTriangle size={16} className="text-rose-500 shrink-0 mt-0.5" />
                    )}
                    <div className="space-y-0.5 flex-1">
                      <p className="font-semibold">{exportError}</p>
                      {exportRemainingAttempts !== null && exportRemainingAttempts > 0 && (
                        <p className="text-[11px] opacity-90">
                          {exportRemainingAttempts === 1
                            ? "Final attempt remaining before this code is invalidated."
                            : `${exportRemainingAttempts} attempts remaining before this code is invalidated.`}
                        </p>
                      )}
                    </div>
                  </div>
                )}

                <div className="flex items-center justify-between pt-1">
                  <button
                    type="button"
                    onClick={() => setIsExportModalOpen(false)}
                    className="px-4 py-2 rounded-xl text-xs font-medium text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 transition cursor-pointer"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    disabled={
                      isExportVerifying ||
                      isExportExhausted ||
                      isExportRateLimited ||
                      exportDigits.join("").length !== 6
                    }
                    onClick={() => handleVerifyExportOtp()}
                    className="px-5 py-2.5 rounded-xl font-semibold text-xs text-white bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed transition shadow-sm flex items-center gap-1.5 cursor-pointer"
                  >
                    {isExportVerifying && <Loader2 size={14} className="animate-spin" />}
                    <span>Verify &amp; Download</span>
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Modal: Confirm Revoke Session */}
      {sessionToRevoke && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
          <div
            role="dialog"
            aria-label="Revoke Session"
            className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-2xl p-6 space-y-5"
          >
            <div className="flex items-start gap-3">
              <div className="p-2.5 rounded-xl bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20 shrink-0">
                <AlertTriangle size={20} />
              </div>
              <div className="space-y-1">
                <h4 className="text-base font-bold text-zinc-900 dark:text-zinc-100">
                  Revoke Session?
                </h4>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 leading-relaxed">
                  This will immediately log out{" "}
                  <strong className="text-zinc-800 dark:text-zinc-200 font-mono">
                    {sessionToRevoke.deviceHint || "Unknown Device"}
                  </strong>
                  {sessionToRevoke.ipAddress ? ` (${sessionToRevoke.ipAddress})` : ""}. Any active
                  work on that device will require re-authentication.
                </p>
              </div>
            </div>

            {revokeSessionError && (
              <div className="p-3 rounded-xl border border-rose-500/30 bg-rose-500/10 text-rose-600 dark:text-rose-400 text-xs flex items-start gap-2">
                <AlertCircle size={15} className="shrink-0 mt-0.5" />
                <div className="leading-relaxed text-[11px]">{revokeSessionError}</div>
              </div>
            )}

            <div className="flex items-center justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => {
                  setSessionToRevoke(null);
                  setRevokeSessionError(null);
                }}
                disabled={isRevokingSession}
                className="px-4 py-2 rounded-xl text-xs font-semibold border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition cursor-pointer disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleConfirmRevokeSession}
                disabled={isRevokingSession}
                className="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 transition shadow-2xs cursor-pointer flex items-center gap-1.5 disabled:opacity-50"
              >
                {isRevokingSession && <Loader2 size={13} className="animate-spin" />}
                <span>Revoke Session</span>
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Modal: Confirm Sign Out All Other Sessions */}
      {isSignOutAllModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
          <div
            role="dialog"
            aria-label="Sign Out All Other Sessions"
            className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-2xl p-6 space-y-5"
          >
            <div className="flex items-start gap-3">
              <div className="p-2.5 rounded-xl bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20 shrink-0">
                <LogOut size={20} />
              </div>
              <div className="space-y-1">
                <h4 className="text-base font-bold text-zinc-900 dark:text-zinc-100">
                  Sign Out All Other Sessions?
                </h4>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 leading-relaxed">
                  All active login sessions on your other devices will be immediately revoked. Your
                  current browser session on this device will remain active.
                </p>
              </div>
            </div>

            {signOutAllError && (
              <div className="p-3 rounded-xl border border-rose-500/30 bg-rose-500/10 text-rose-600 dark:text-rose-400 text-xs flex items-start gap-2">
                <AlertCircle size={15} className="shrink-0 mt-0.5" />
                <div className="leading-relaxed text-[11px]">{signOutAllError}</div>
              </div>
            )}

            <div className="flex items-center justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => {
                  setIsSignOutAllModalOpen(false);
                  setSignOutAllError(null);
                }}
                disabled={isRevokingOtherSessions}
                className="px-4 py-2 rounded-xl text-xs font-semibold border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition cursor-pointer disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleConfirmSignOutAllOther}
                disabled={isRevokingOtherSessions}
                className="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 transition shadow-2xs cursor-pointer flex items-center gap-1.5 disabled:opacity-50"
              >
                {isRevokingOtherSessions && <Loader2 size={13} className="animate-spin" />}
                <span>Sign Out Others</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
