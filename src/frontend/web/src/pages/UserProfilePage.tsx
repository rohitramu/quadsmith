import { useState, useEffect, useMemo } from "react";
import { useParams, Link } from "react-router-dom";
import { useTransport } from "@connectrpc/connect-query";
import { createClient } from "@connectrpc/connect";
import { Loader2, Hammer, Calendar, AlertCircle } from "lucide-react";
import { UserService, type PublicUserProfile } from "../gen/quadsmith/user_pb";
import { getAvatarPalette, getInitial } from "../lib/avatarColor";

export function UserProfilePage() {
  const { id } = useParams<{ id: string }>();
  const transport = useTransport();
  const userClient = useMemo(() => createClient(UserService, transport), [transport]);
  const [profile, setProfile] = useState<PublicUserProfile | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;

    async function fetchProfile() {
      if (!id) return;
      setIsLoading(true);
      setErrorMsg(null);
      try {
        const res = await userClient.getPublicProfile({ id });
        if (mounted) {
          setProfile(res);
        }
      } catch (err: unknown) {
        if (mounted) {
          if (err instanceof Error) {
            setErrorMsg(err.message);
          } else {
            setErrorMsg("User profile not found");
          }
        }
      } finally {
        if (mounted) {
          setIsLoading(false);
        }
      }
    }

    fetchProfile();
    return () => {
      mounted = false;
    };
  }, [id]);

  if (isLoading) {
    return (
      <div className="flex h-96 w-full items-center justify-center text-zinc-400">
        <Loader2 size={24} className="animate-spin text-blue-500" />
      </div>
    );
  }

  if (errorMsg || !profile) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center space-y-4">
        <div className="w-12 h-12 rounded-full bg-red-500/10 text-red-500 flex items-center justify-center mx-auto">
          <AlertCircle size={24} />
        </div>
        <h2 className="text-xl font-bold">User Not Found</h2>
        <p className="text-xs text-zinc-500">
          {errorMsg || "The requested username does not exist."}
        </p>
        <Link
          to="/"
          className="inline-block px-4 py-2 rounded-xl text-xs font-semibold bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
        >
          Return Home
        </Link>
      </div>
    );
  }

  const palette = getAvatarPalette(profile.id);
  const initial = getInitial(profile.displayName || profile.id);

  return (
    <div className="max-w-5xl mx-auto px-4 py-8 space-y-8">
      {/* User Profile Header Card */}
      <div className="p-8 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-6">
          <div className="flex items-center gap-5">
            {profile.avatarUrl ? (
              <img
                src={profile.avatarUrl}
                alt={profile.displayName}
                className="w-20 h-20 rounded-full object-cover border-2 border-zinc-200 dark:border-zinc-800 shadow-md"
              />
            ) : (
              <div
                className={`w-20 h-20 rounded-full border-2 flex items-center justify-center font-bold text-3xl shadow-md ${palette.badgeClass}`}
              >
                {initial}
              </div>
            )}
            <div className="space-y-1">
              <h1 className="text-2xl font-extrabold tracking-tight">{profile.displayName}</h1>
              <p className="text-xs font-mono text-zinc-500 dark:text-zinc-400">@{profile.id}</p>
              {profile.createdAt && (
                <div className="flex items-center gap-1.5 text-[11px] text-zinc-400 pt-1">
                  <Calendar size={12} />
                  <span>
                    Member since{" "}
                    {new Date(Number(profile.createdAt.seconds) * 1000).toLocaleDateString(
                      undefined,
                      {
                        year: "numeric",
                        month: "short",
                      },
                    )}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Stats Badge */}
          <div className="flex items-center gap-4 bg-zinc-50 dark:bg-zinc-950 p-4 rounded-xl border border-zinc-200 dark:border-zinc-800">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-lg bg-blue-500/10 text-blue-500">
                <Hammer size={20} />
              </div>
              <div>
                <div className="text-xl font-bold font-mono">{profile.buildCount}</div>
                <div className="text-[11px] text-zinc-400 font-medium">Authored Builds</div>
              </div>
            </div>
          </div>
        </div>

        {/* Bio */}
        {profile.bio && (
          <div className="p-4 rounded-xl bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 text-xs text-zinc-600 dark:text-zinc-300 leading-relaxed">
            {profile.bio}
          </div>
        )}
      </div>

      {/* Builds Section */}
      <div className="space-y-4">
        <h2 className="text-base font-bold flex items-center gap-2">
          <Hammer size={16} className="text-blue-500" />
          <span>Builds by @{profile.id}</span>
        </h2>

        {profile.buildCount === 0 ? (
          <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 text-zinc-400 space-y-2">
            <p className="text-xs">This user hasn&apos;t published any custom builds yet.</p>
          </div>
        ) : (
          <div className="p-6 text-center rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-zinc-500 text-xs">
            Viewing {profile.buildCount} published build{profile.buildCount === 1 ? "" : "s"}.
          </div>
        )}
      </div>
    </div>
  );
}
