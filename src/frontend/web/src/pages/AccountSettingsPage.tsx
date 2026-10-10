import { useState, useEffect, useRef } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import {
  User as UserIcon,
  LogOut,
  Check,
  Loader2,
  ExternalLink,
  ShieldCheck,
  Mail,
  ArrowRightLeft,
  Plus,
  X,
  Trash2,
  AlertCircle,
  CheckCircle2,
  UserX,
  Package,
  AlertTriangle,
} from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { getAvatarPalette, getInitial } from "../lib/avatarColor";

export function AccountSettingsPage() {
  const {
    user,
    profile,
    emails,
    updateProfile,
    logout,
    refreshEmails,
    sendAddEmailOtp,
    verifyAddEmailOtp,
    mergeAccount,
    setPrimaryEmail,
    removeUserEmail,
  } = useAuth();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const mergeToken = searchParams.get("merge_token");
  const addEmailToken = searchParams.get("add_email_token");

  useEffect(() => {
    refreshEmails();
  }, [refreshEmails]);

  // Profile form state
  const [displayName, setDisplayName] = useState(profile?.displayName || user?.id || "");
  const [avatarUrl, setAvatarUrl] = useState(profile?.avatarUrl || "");
  const [bio, setBio] = useState(profile?.bio || "");
  const [isSaving, setIsSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // Add Email state
  const [isAddEmailOpen, setIsAddEmailOpen] = useState(false);
  const [newEmail, setNewEmail] = useState("");
  const [isSendingAddOtp, setIsSendingAddOtp] = useState(false);
  const [addOtpSent, setAddOtpSent] = useState(false);
  const [addOtpCode, setAddOtpCode] = useState("");
  const [isVerifyingAddOtp, setIsVerifyingAddOtp] = useState(false);
  const [addEmailError, setAddEmailError] = useState<string | null>(null);
  const [addEmailSuccess, setAddEmailSuccess] = useState<string | null>(null);

  // Merge Account Modal state
  const [isMergeModalOpen, setIsMergeModalOpen] = useState(false);
  const [mergeModalStep, setMergeModalStep] = useState<"ownership" | "summary" | "success">(
    "ownership",
  );
  const [mergeEmail, setMergeEmail] = useState("");
  const [mergeOtherHandle, setMergeOtherHandle] = useState("");
  const [mergeOtherBuildCount, setMergeOtherBuildCount] = useState<number>(0);
  const [mergeOtherBuildNames, setMergeOtherBuildNames] = useState<string[]>([]);
  const [mergeOtherEmails, setMergeOtherEmails] = useState<string[]>([]);
  const [mergeOtherOauthProviders, setMergeOtherOauthProviders] = useState<string[]>([]);
  const [verifiedMergeCode, setVerifiedMergeCode] = useState("");
  const [verifiedMergeToken, setVerifiedMergeToken] = useState("");
  const [mergeOtpDigits, setMergeOtpDigits] = useState<string[]>(["", "", "", "", "", ""]);
  const [isVerifyingOwnership, setIsVerifyingOwnership] = useState(false);
  const [isExecutingMerge, setIsExecutingMerge] = useState(false);
  const [mergeError, setMergeError] = useState<string | null>(null);
  const [transferredBuilds, setTransferredBuilds] = useState<number>(0);

  const mergeOtpInputRefs = useRef<(HTMLInputElement | null)[]>([]);

  // Magic link auto-merge or auto-add-email
  useEffect(() => {
    if (mergeToken) {
      let isMounted = true;
      async function handleMagicMerge() {
        setIsVerifyingOwnership(true);
        setMergeError(null);
        setIsMergeModalOpen(true);
        try {
          const res = await verifyAddEmailOtp("", undefined, mergeToken);
          if (!isMounted) return;
          if (res.mergeRequired) {
            setMergeOtherHandle(res.otherUserHandle || "pilot");
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
        } finally {
          if (isMounted) {
            setIsVerifyingOwnership(false);
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
            setMergeOtherHandle(res.otherUserHandle || "pilot");
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

  // Email Action feedback
  const [emailActionLoading, setEmailActionLoading] = useState<string | null>(null);
  const [emailActionError, setEmailActionError] = useState<string | null>(null);

  const palette = getAvatarPalette(user?.uuid);
  const initial = getInitial(profile?.displayName || user?.id);

  // Handle Profile Save
  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setSaveSuccess(false);
    setErrorMsg(null);

    try {
      await updateProfile({
        displayName: displayName.trim(),
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

  // Handle Add Email - Send Code
  const handleSendAddCode = async () => {
    if (!newEmail.trim()) {
      setAddEmailError("Please enter an email address");
      return;
    }
    setIsSendingAddOtp(true);
    setAddEmailError(null);
    try {
      const res = await sendAddEmailOtp(newEmail.trim());
      if (res.isExistingAccount) {
        // Email belongs to an existing Quadsmith account!
        // Bring up the Merge Modal for ownership check
        setMergeEmail(newEmail.trim());
        setMergeOtherHandle("");
        setMergeOtherBuildCount(0);
        setMergeOtherBuildNames([]);
        setMergeOtherEmails([]);
        setMergeOtherOauthProviders([]);
        setVerifiedMergeCode("");
        setVerifiedMergeToken("");
        setMergeOtpDigits(["", "", "", "", "", ""]);
        setMergeError(null);
        setMergeModalStep("ownership");
        setIsMergeModalOpen(true);
        setIsAddEmailOpen(false);
        setTimeout(() => mergeOtpInputRefs.current[0]?.focus(), 100);
      } else {
        setAddOtpSent(true);
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        setAddEmailError(err.message);
      } else {
        setAddEmailError("Failed to send verification code. Please try again.");
      }
    } finally {
      setIsSendingAddOtp(false);
    }
  };

  // Handle Add Email - Verify Code
  const handleVerifyAddCode = async () => {
    if (!addOtpCode.trim() || addOtpCode.trim().length !== 6) {
      setAddEmailError("Please enter a valid 6-digit code");
      return;
    }
    setIsVerifyingAddOtp(true);
    setAddEmailError(null);
    try {
      const res = await verifyAddEmailOtp(newEmail.trim(), addOtpCode.trim());
      if (res.mergeRequired) {
        setMergeEmail(newEmail.trim());
        setMergeOtherHandle(res.otherUserHandle || "pilot");
        setMergeOtherBuildCount(res.otherBuildCount || 0);
        setMergeOtherBuildNames(res.otherBuildNames || []);
        setMergeOtherEmails(res.otherEmails || []);
        setMergeOtherOauthProviders(res.otherOauthProviders || []);
        setVerifiedMergeCode(addOtpCode.trim());
        setVerifiedMergeToken("");
        setMergeError(null);
        // Ownership verified in inline form, proceed directly to summary
        setMergeModalStep("summary");
        setIsMergeModalOpen(true);
        setIsAddEmailOpen(false);
        return;
      }
      setAddEmailSuccess(`Successfully added ${newEmail.trim()} to your account.`);
      setTimeout(() => setAddEmailSuccess(null), 4000);
      setIsAddEmailOpen(false);
      setNewEmail("");
      setAddOtpSent(false);
      setAddOtpCode("");
    } catch (err: unknown) {
      if (err instanceof Error) {
        setAddEmailError(err.message);
      } else {
        setAddEmailError("Verification failed. Please check the code and try again.");
      }
    } finally {
      setIsVerifyingAddOtp(false);
    }
  };

  const handleMergeOtpChange = (index: number, value: string) => {
    const clean = value.replace(/\D/g, "");
    if (!clean && value !== "") return;

    const newDigits = [...mergeOtpDigits];
    if (clean.length > 1) {
      // Pasted multi-character string
      const chars = clean.slice(0, 6).split("");
      chars.forEach((c, i) => {
        newDigits[i] = c;
      });
      setMergeOtpDigits(newDigits);
      const nextIndex = Math.min(chars.length, 5);
      mergeOtpInputRefs.current[nextIndex]?.focus();
      return;
    }

    newDigits[index] = clean;
    setMergeOtpDigits(newDigits);

    // Auto-advance
    if (clean && index < 5) {
      mergeOtpInputRefs.current[index + 1]?.focus();
    }
  };

  const handleMergeKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Backspace" && !mergeOtpDigits[index] && index > 0) {
      mergeOtpInputRefs.current[index - 1]?.focus();
    }
  };

  // Step 1: Verify Ownership (checks 6-digit code for existing account)
  const handleVerifyOwnership = async () => {
    const code = mergeOtpDigits.join("");
    if (code.length !== 6) {
      setMergeError("Please enter all 6 digits of the verification code");
      return;
    }
    setIsVerifyingOwnership(true);
    setMergeError(null);
    try {
      const res = await verifyAddEmailOtp(mergeEmail.trim(), code);
      if (res.mergeRequired) {
        setMergeOtherHandle(res.otherUserHandle || "pilot");
        setMergeOtherBuildCount(res.otherBuildCount || 0);
        setMergeOtherBuildNames(res.otherBuildNames || []);
        setMergeOtherEmails(res.otherEmails || []);
        setMergeOtherOauthProviders(res.otherOauthProviders || []);
        setVerifiedMergeCode(code);
        setMergeModalStep("summary");
      } else {
        setAddEmailSuccess(`Successfully added ${mergeEmail.trim()} to your account.`);
        setTimeout(() => setAddEmailSuccess(null), 4000);
        setIsMergeModalOpen(false);
        await refreshEmails();
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        setMergeError(err.message);
      } else {
        setMergeError("Verification failed. Please check the code and try again.");
      }
    } finally {
      setIsVerifyingOwnership(false);
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
              verifiedAt: undefined,
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

  return (
    <div className="max-w-4xl mx-auto px-4 py-8 space-y-8">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-200 dark:border-zinc-800 pb-6">
        <div className="flex items-center gap-4">
          {avatarUrl ? (
            <img
              src={avatarUrl}
              alt={displayName}
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
            <p className="text-xs text-zinc-500 dark:text-zinc-400 font-mono mt-0.5">
              Pilot Handle: @{user?.id}
            </p>
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
              <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-blue-500/10 text-blue-500 dark:text-blue-400 border border-blue-500/20">
                Multi-Email Login
              </span>
            </h2>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
              Any verified email address listed below can be used to receive login codes and access
              your account.
            </p>
          </div>
          {!isAddEmailOpen && (
            <button
              onClick={() => {
                setIsAddEmailOpen(true);
                setAddEmailError(null);
                setAddOtpSent(false);
              }}
              className="px-3.5 py-1.5 rounded-xl text-xs font-semibold bg-blue-600 hover:bg-blue-500 text-white transition flex items-center gap-1.5 self-start sm:self-auto shadow-sm"
            >
              <Plus size={14} />
              <span>Add Email</span>
            </button>
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

        {/* Collapsible Add Email Form */}
        {isAddEmailOpen && (
          <div className="rounded-xl border border-blue-500/30 bg-blue-500/5 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-semibold text-blue-600 dark:text-blue-300">
                Add New Email Address
              </h3>
              <button
                onClick={() => {
                  setIsAddEmailOpen(false);
                  setNewEmail("");
                  setAddOtpSent(false);
                  setAddOtpCode("");
                  setAddEmailError(null);
                }}
                className="text-xs text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200"
              >
                <X size={14} />
              </button>
            </div>
            <p className="text-[11px] text-zinc-500 dark:text-zinc-400">
              A 6-digit verification code will be sent to confirm you own this address.
            </p>

            {addEmailError && (
              <div className="p-2.5 rounded-lg border border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400 text-xs">
                {addEmailError}
              </div>
            )}

            <div className="flex flex-col sm:flex-row gap-2">
              <input
                type="email"
                placeholder="pilot.secondary@example.com"
                value={newEmail}
                disabled={addOtpSent}
                onChange={(e) => setNewEmail(e.target.value)}
                className="flex-1 px-3 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 dark:placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              {!addOtpSent ? (
                <button
                  type="button"
                  onClick={handleSendAddCode}
                  disabled={isSendingAddOtp || !newEmail.trim()}
                  className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-xs font-semibold transition flex items-center justify-center gap-1.5"
                >
                  {isSendingAddOtp && <Loader2 size={13} className="animate-spin" />}
                  <span>Send Code</span>
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => setAddOtpSent(false)}
                  className="px-3 py-2 text-xs text-zinc-400 hover:text-zinc-200"
                >
                  Change Email
                </button>
              )}
            </div>

            {addOtpSent && (
              <div className="space-y-2 pt-2 border-t border-blue-500/20">
                <p className="text-[11px] text-emerald-600 dark:text-emerald-400 font-medium">
                  ✓ Verification code sent! Enter the 6-digit code from the email:
                </p>
                <div className="flex gap-2">
                  <input
                    type="text"
                    maxLength={6}
                    placeholder="123456"
                    value={addOtpCode}
                    onChange={(e) => setAddOtpCode(e.target.value.replace(/\D/g, ""))}
                    className="w-32 px-3 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 text-center font-mono tracking-widest text-xs font-bold text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <button
                    type="button"
                    onClick={handleVerifyAddCode}
                    disabled={isVerifyingAddOtp || addOtpCode.length !== 6}
                    className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-xs font-semibold transition flex items-center gap-1.5"
                  >
                    {isVerifyingAddOtp && <Loader2 size={13} className="animate-spin" />}
                    <span>Verify & Associate</span>
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Email List Table */}
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-zinc-200 dark:border-zinc-800 text-[11px] uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
                <th className="pb-2.5 font-semibold">Email Address</th>
                <th className="pb-2.5 font-semibold">Status</th>
                <th className="pb-2.5 font-semibold">Added</th>
                <th className="pb-2.5 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200/60 dark:divide-zinc-800/60 font-mono">
              {displayedEmails.map((em) => (
                <tr key={em.email} className="group">
                  <td className="py-3 pr-4">
                    <div className="flex items-center gap-2 font-sans font-medium text-zinc-900 dark:text-zinc-100">
                      <span>{em.email}</span>
                      {em.isPrimary ? (
                        <span className="px-2 py-0.5 rounded text-[10px] font-bold tracking-wide uppercase bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                          Primary
                        </span>
                      ) : (
                        <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700">
                          Secondary
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="py-3 text-emerald-600 dark:text-emerald-400 font-sans text-[11px]">
                    <span>✓ Verified</span>
                  </td>
                  <td className="py-3 text-zinc-500 text-[11px] font-sans">
                    {formatDate(em.createdAt?.seconds)}
                  </td>
                  <td className="py-3 text-right space-x-2 font-sans">
                    {em.isPrimary ? (
                      <span className="text-zinc-400 dark:text-zinc-600 text-[11px] italic">
                        Default login address
                      </span>
                    ) : (
                      <>
                        <button
                          type="button"
                          disabled={emailActionLoading === em.email}
                          onClick={() => handleSetPrimary(em.email)}
                          className="px-2.5 py-1 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-300 text-[11px] font-medium transition inline-flex items-center gap-1"
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
                          className="p-1.5 rounded-lg border border-rose-500/20 text-rose-600 dark:text-rose-400 hover:bg-rose-500/10 transition inline-flex items-center justify-center align-middle"
                        >
                          {emailActionLoading === em.email ? (
                            <Loader2 size={13} className="animate-spin" />
                          ) : (
                            <Trash2 size={13} />
                          )}
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* Section 2: Profile Details Form */}
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

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Public Handle (id) */}
          <div>
            <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">
              Public Pilot Handle (<code className="font-mono text-blue-500">id</code>)
            </label>
            <div className="flex items-center rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 px-3.5 py-2.5 text-xs text-zinc-500 dark:text-zinc-400">
              <span className="font-mono mr-1">quadsmith.dev/users/</span>
              <span className="font-mono font-bold text-zinc-900 dark:text-zinc-100">
                {user?.id}
              </span>
            </div>
            <p className="text-[11px] text-zinc-400 mt-1">
              Your unique public handle across build logs and component submissions.
            </p>
          </div>

          {/* Primary Login Email */}
          <div>
            <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">
              Primary Account Email
            </label>
            <div className="flex items-center justify-between rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 px-3.5 py-2.5 text-xs">
              <span className="font-mono text-zinc-900 dark:text-zinc-100">{user?.email}</span>
              <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                Primary
              </span>
            </div>
            <p className="text-[11px] text-zinc-400 mt-1">
              Manage all associated login addresses in the section above.
            </p>
          </div>

          {/* Display Name */}
          <div>
            <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">
              Display Name
            </label>
            <input
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="w-full px-3.5 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 text-xs text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          {/* Avatar Image URL */}
          <div>
            <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">
              Avatar Image URL (Optional)
            </label>
            <input
              type="url"
              placeholder="https://example.com/avatar.jpg"
              value={avatarUrl}
              onChange={(e) => setAvatarUrl(e.target.value)}
              className="w-full px-3.5 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 text-xs text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-[11px] text-zinc-400 mt-1">
              Leave blank to automatically generate an initial with adaptive category chip colors.
            </p>
          </div>

          {/* Bio / Pilot Notes */}
          <div className="md:col-span-2">
            <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">
              Bio / Pilot Notes
            </label>
            <textarea
              rows={3}
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="Tell other pilots about your flying style, favorite frames, and builds..."
              className="w-full px-3.5 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 text-xs text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {/* Session Security Banner & Save Button */}
        <div className="p-4 rounded-xl bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-2 text-xs">
            <ShieldCheck size={16} className="text-emerald-500 shrink-0" />
            <span className="text-zinc-600 dark:text-zinc-300 font-medium">Session Status:</span>
            <span className="text-zinc-500 font-mono text-[11px]">
              Active 5-Day Sliding Window (refreshes automatically on activity)
            </span>
          </div>

          <div className="flex items-center gap-3">
            {saveSuccess && (
              <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                <Check size={14} />
                <span>Saved successfully</span>
              </span>
            )}
            <button
              type="submit"
              disabled={isSaving}
              className="px-5 py-2 rounded-xl text-xs font-semibold text-white bg-blue-600 hover:bg-blue-500 disabled:opacity-50 transition shadow-sm flex items-center gap-1.5"
            >
              {isSaving && <Loader2 size={14} className="animate-spin" />}
              <span>Save Changes</span>
            </button>
          </div>
        </div>
      </form>

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
                    {mergeModalStep === "ownership"
                      ? "Verify Account Ownership"
                      : mergeModalStep === "summary"
                        ? "Confirm Account Merge"
                        : "Accounts Merged"}
                  </h3>
                  <p className="text-[11px] text-zinc-500 dark:text-zinc-400">
                    {mergeModalStep === "ownership"
                      ? "Enter the verification code sent to your email to continue"
                      : mergeModalStep === "summary"
                        ? `Consolidate @${mergeOtherHandle} into @${user?.id}`
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

            {mergeModalStep === "ownership" && (
              <div className="space-y-4">
                {/* Instructions & Notice */}
                <div className="p-4 rounded-xl border border-purple-500/30 bg-purple-500/10 space-y-2.5">
                  <div className="text-xs font-bold text-purple-700 dark:text-purple-300 flex items-center gap-1.5">
                    <AlertCircle size={14} />
                    <span>Existing Account Detected</span>
                  </div>
                  <p className="text-[11px] text-zinc-600 dark:text-zinc-300 leading-relaxed">
                    The email{" "}
                    <strong className="font-mono text-zinc-900 dark:text-zinc-100">
                      {mergeEmail}
                    </strong>{" "}
                    is already associated with an existing Quadsmith account. To merge that account
                    into your profile, please verify ownership of the email address.
                  </p>
                </div>

                {/* Dual-path instructions */}
                <div className="p-3.5 rounded-xl border border-purple-500/20 bg-purple-500/10 text-zinc-700 dark:text-zinc-300 text-xs space-y-1.5">
                  <div className="flex items-center gap-1.5 font-semibold text-purple-600 dark:text-purple-400">
                    <CheckCircle2 size={14} />
                    <span>Two Ways to Verify:</span>
                  </div>
                  <ul className="list-disc list-inside space-y-1 text-[11px] text-zinc-600 dark:text-zinc-400">
                    <li>
                      Enter the <strong>6-digit verification code</strong> sent to your email below.
                    </li>
                    <li>
                      Or click the <strong>1-click Magic Link</strong> in the email to verify
                      instantly.
                    </li>
                  </ul>
                </div>

                {/* 6 Digit Segmented Inputs */}
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <label className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                      Verification Code (3-min expiry)
                    </label>
                  </div>

                  <div className="flex items-center justify-between gap-2">
                    {mergeOtpDigits.map((digit, index) => (
                      <input
                        key={index}
                        ref={(el) => {
                          mergeOtpInputRefs.current[index] = el;
                        }}
                        type="text"
                        inputMode="numeric"
                        maxLength={1}
                        aria-label={`Digit ${index + 1}`}
                        value={digit}
                        disabled={isVerifyingOwnership}
                        onChange={(e) => handleMergeOtpChange(index, e.target.value)}
                        onKeyDown={(e) => handleMergeKeyDown(index, e)}
                        className="w-12 h-14 text-center font-mono text-2xl font-bold rounded-xl border bg-zinc-50 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 focus:outline-none transition border-zinc-300 dark:border-zinc-700 focus:border-purple-500 focus:ring-2 focus:ring-purple-500/20"
                      />
                    ))}
                  </div>
                </div>

                {/* Footer Buttons */}
                <div className="space-y-2.5 pt-2">
                  <div className="flex gap-2.5">
                    <button
                      type="button"
                      onClick={() => setIsMergeModalOpen(false)}
                      className="flex-1 py-2.5 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-300 text-xs font-semibold transition"
                    >
                      Cancel
                    </button>
                    <button
                      type="button"
                      onClick={handleVerifyOwnership}
                      disabled={isVerifyingOwnership || mergeOtpDigits.join("").length !== 6}
                      className="flex-[2] py-2.5 rounded-xl bg-purple-600 hover:bg-purple-500 disabled:opacity-50 text-white text-xs font-semibold transition flex items-center justify-center gap-1.5 shadow-sm"
                    >
                      {isVerifyingOwnership && <Loader2 size={13} className="animate-spin" />}
                      <span>Verify Ownership & Review</span>
                    </button>
                  </div>

                  <div className="flex items-center justify-between text-xs pt-1">
                    <span className="text-zinc-400 text-[11px]">Didn't receive the email?</span>
                    <button
                      type="button"
                      onClick={() => handleSendAddCode()}
                      disabled={isSendingAddOtp || isVerifyingOwnership}
                      className="text-purple-600 dark:text-purple-400 hover:underline font-medium disabled:opacity-50"
                    >
                      Resend code
                    </button>
                  </div>
                </div>
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
                    Ownership verified for <strong>@{mergeOtherHandle}</strong>
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
                          @{mergeOtherHandle}
                        </span>
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 leading-relaxed">
                        The handle{" "}
                        <strong className="text-zinc-700 dark:text-zinc-300">
                          @{mergeOtherHandle}
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
                        Associate Email Address
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 leading-relaxed">
                        <strong className="font-mono text-zinc-800 dark:text-zinc-200">
                          {mergeEmail}
                        </strong>
                        {mergeOtherEmails.length > 1 && (
                          <span>
                            {" "}
                            (along with{" "}
                            {mergeOtherEmails.filter((e) => e !== mergeEmail).join(", ")})
                          </span>
                        )}{" "}
                        will be added as a verified login email under your account (
                        <strong className="text-zinc-700 dark:text-zinc-300">@{user?.id}</strong>).
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
                        @{mergeOtherHandle}
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
                      className="flex-[2] py-2.5 rounded-xl bg-red-600 hover:bg-red-500 active:bg-red-700 disabled:opacity-50 text-white text-xs font-semibold transition flex items-center justify-center gap-1.5 shadow-sm shadow-red-500/20"
                    >
                      {isExecutingMerge && <Loader2 size={13} className="animate-spin" />}
                      <span>Confirm Merge</span>
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
                    ? `All ${transferredBuilds} builds from @${mergeOtherHandle} have been reassigned to your account, and `
                    : `The account @${mergeOtherHandle} has been consolidated into your profile, and `}
                  <span className="font-mono text-zinc-900 dark:text-zinc-200">{mergeEmail}</span>{" "}
                  has been added to your login emails.
                </p>
                <button
                  type="button"
                  onClick={() => {
                    setIsMergeModalOpen(false);
                    setNewEmail("");
                    setAddOtpSent(false);
                    setAddOtpCode("");
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
    </div>
  );
}
