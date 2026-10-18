import { ArrowRight, ShieldAlert, LogOut } from "lucide-react";
import { Link } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

export function AccountRestrictedPage() {
  const { user, logout } = useAuth();
  const ban = user?.activeBan;

  const formatDate = (seconds?: bigint) => {
    if (!seconds) return "Recent";
    return new Date(Number(seconds) * 1000).toLocaleDateString(undefined, {
      year: "numeric",
      month: "long",
      day: "numeric",
    });
  };

  const isTemporary = Boolean(ban?.expiresAt);

  return (
    <div
      data-testid="account-restricted-screen"
      className="flex flex-col items-center justify-center py-16 px-4 text-center max-w-2xl mx-auto"
    >
      <div className="relative mb-6">
        <div className="w-20 h-20 rounded-3xl bg-rose-500/15 border border-rose-500/20 text-rose-400 flex items-center justify-center shadow-lg shadow-rose-500/5">
          <ShieldAlert className="w-10 h-10" />
        </div>
        <span className="absolute -top-1 -right-1 w-6 h-6 rounded-full bg-rose-600 text-white flex items-center justify-center text-xs font-bold ring-4 ring-zinc-950">
          !
        </span>
      </div>

      <h1 className="text-2xl md:text-3xl font-bold tracking-tight text-zinc-100 mb-2">
        Account Access Restricted
      </h1>
      <p className="text-sm text-zinc-400 max-w-md mx-auto mb-8">
        Your Quadsmith account has been suspended by community moderation. You are currently
        restricted from public actions and community features.
      </p>

      {/* Ban Details Card */}
      <div className="w-full max-w-lg rounded-2xl border border-rose-500/20 bg-rose-950/20 p-6 text-left space-y-4 mb-8 backdrop-blur shadow-sm">
        <div className="flex items-start justify-between gap-4 border-b border-rose-500/15 pb-4">
          <div>
            <div className="text-[11px] font-semibold text-rose-400 uppercase tracking-wider">
              Reason
            </div>
            <div className="text-sm font-medium text-zinc-100 mt-0.5">
              {ban?.reason || "Account suspended due to community guidelines infraction."}
            </div>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4 text-xs">
          <div>
            <div className="text-[11px] font-medium text-zinc-400">Date Enacted</div>
            <div className="font-semibold text-zinc-200 mt-0.5">
              {formatDate(ban?.createdAt?.seconds)}
            </div>
          </div>
          <div>
            <div className="text-[11px] font-medium text-zinc-400">Suspension Duration</div>
            <div className="font-semibold text-rose-400 mt-0.5">
              {isTemporary && ban?.expiresAt
                ? `Expires ${formatDate(ban.expiresAt.seconds)}`
                : "Indefinite Suspension"}
            </div>
          </div>
        </div>
      </div>

      {/* Call to Action */}
      <div className="flex flex-col sm:flex-row items-center gap-3 w-full sm:w-auto">
        <Link
          to="/account"
          className="w-full sm:w-auto px-6 py-3 rounded-xl bg-white text-zinc-900 font-semibold text-xs hover:bg-zinc-100 transition shadow-sm flex items-center justify-center gap-2 cursor-pointer"
        >
          <span>Account Settings</span>
          <ArrowRight className="w-3.5 h-3.5" />
        </Link>
        <button
          onClick={logout}
          className="w-full sm:w-auto px-5 py-3 rounded-xl border border-zinc-800 text-zinc-400 font-medium text-xs hover:text-zinc-100 transition flex items-center justify-center gap-2 cursor-pointer"
        >
          <LogOut className="w-3.5 h-3.5" />
          <span>Sign Out</span>
        </button>
      </div>

      <p className="text-[11px] text-zinc-400 mt-6">
        You may still view your profile bio, manage auth sessions, export data, or delete your
        account in{" "}
        <Link to="/account" className="underline hover:text-zinc-300">
          Account Settings
        </Link>
        .
      </p>
    </div>
  );
}
