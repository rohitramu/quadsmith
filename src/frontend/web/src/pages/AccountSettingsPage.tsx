import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { User as UserIcon, LogOut, Check, Loader2, ExternalLink, ShieldCheck } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { getAvatarPalette, getInitial } from "../lib/avatarColor";

export function AccountSettingsPage() {
  const { user, profile, updateProfile, logout } = useAuth();
  const navigate = useNavigate();

  const [displayName, setDisplayName] = useState(profile?.displayName || user?.id || "");
  const [avatarUrl, setAvatarUrl] = useState(profile?.avatarUrl || "");
  const [bio, setBio] = useState(profile?.bio || "");
  const [isSaving, setIsSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const palette = getAvatarPalette(user?.uuid);
  const initial = getInitial(profile?.displayName || user?.id);

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

  const handleLogout = async () => {
    await logout();
    navigate("/login", { replace: true });
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

      {/* Main Settings Form */}
      <form onSubmit={handleSave} className="space-y-6">
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

          {/* Account Email (Private) */}
          <div>
            <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">
              Account Email (Private)
            </label>
            <div className="flex items-center justify-between rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 px-3.5 py-2.5 text-xs">
              <span className="font-mono text-zinc-900 dark:text-zinc-100">{user?.email}</span>
              <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/10 text-emerald-500">
                Verified
              </span>
            </div>
            <p className="text-[11px] text-zinc-400 mt-1">
              Used strictly for login verification. Never displayed on public pages.
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
    </div>
  );
}
