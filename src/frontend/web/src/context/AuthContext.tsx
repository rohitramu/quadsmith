import React, { createContext, useContext, useState, useEffect, useCallback, useMemo } from "react";
import { useTransport } from "@connectrpc/connect-query";
import { createClient } from "@connectrpc/connect";
import { AuthService } from "../gen/quadsmith/auth_pb";
import type { User, UserProfile, UserEmail } from "../gen/quadsmith/user_pb";
import { setAccessToken, getAccessToken } from "../lib/api";

export interface AuthContextType {
  user: User | null;
  profile: UserProfile | null;
  emails: UserEmail[];
  isAuthenticated: boolean;
  isLoading: boolean;
  sendEmailOtp: (
    email: string,
  ) => Promise<{ success: boolean; cooldownSeconds: number; message: string }>;
  verifyEmailOtp: (email: string, code: string) => Promise<void>;
  verifyMagicLink: (token: string) => Promise<void>;
  updateProfile: (data: {
    displayName?: string;
    avatarUrl?: string;
    bio?: string;
  }) => Promise<void>;
  logout: () => Promise<void>;
  refreshSession: () => Promise<void>;
  refreshEmails: () => Promise<void>;
  sendAddEmailOtp: (email: string) => Promise<{
    success: boolean;
    cooldownSeconds: number;
    message: string;
    isExistingAccount?: boolean;
    existingUserHandle?: string;
    existingBuildCount?: number;
  }>;
  verifyAddEmailOtp: (
    email: string,
    code?: string | null,
    token?: string | null,
  ) => Promise<{
    success: boolean;
    message: string;
    mergeRequired: boolean;
    otherUserHandle?: string;
    otherBuildCount?: number;
    otherBuildNames?: string[];
    otherEmails?: string[];
    otherOauthProviders?: string[];
  }>;
  mergeAccount: (
    email: string,
    code?: string | null,
    token?: string | null,
  ) => Promise<{
    success: boolean;
    message: string;
    transferredBuildCount: number;
    mergedHandle: string;
  }>;
  setPrimaryEmail: (email: string) => Promise<{ success: boolean; message: string }>;
  removeUserEmail: (email: string) => Promise<{ success: boolean; message: string }>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const transport = useTransport();
  const authClient = useMemo(() => createClient(AuthService, transport), [transport]);
  const [user, setUser] = useState<User | null>(null);
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [emails, setEmails] = useState<UserEmail[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const refreshSession = useCallback(async () => {
    try {
      const res = await authClient.refreshToken({});
      if (res.accessToken) {
        setAccessToken(res.accessToken);
      }
      if (res.user) {
        setUser(res.user);
      }
      if (res.profile) {
        setProfile(res.profile);
      }
    } catch {
      setAccessToken(null);
      setUser(null);
      setProfile(null);
    }
  }, []);

  useEffect(() => {
    let mounted = true;

    async function initAuth() {
      try {
        const token = getAccessToken();
        if (token) {
          try {
            const currentUser = await authClient.getCurrentUser({});
            if (mounted && currentUser) {
              setUser(currentUser);
              try {
                const emailsRes = await authClient.listUserEmails({});
                if (mounted) setEmails(emailsRes.emails || []);
              } catch {
                // ignore
              }
              // Profile refresh via silent refresh
              const refreshRes = await authClient.refreshToken({});
              if (mounted) {
                if (refreshRes.accessToken) setAccessToken(refreshRes.accessToken);
                if (refreshRes.profile) setProfile(refreshRes.profile);
              }
              return;
            }
          } catch {
            // Token might be expired, try refresh cookie
          }
        }
        // Try refresh token from cookie
        const res = await authClient.refreshToken({});
        if (mounted) {
          if (res.accessToken) setAccessToken(res.accessToken);
          if (res.user) setUser(res.user);
          if (res.profile) setProfile(res.profile);
          try {
            const emailsRes = await authClient.listUserEmails({});
            if (mounted) setEmails(emailsRes.emails || []);
          } catch {
            // ignore
          }
        }
      } catch {
        if (mounted) {
          setAccessToken(null);
          setUser(null);
          setProfile(null);
        }
      } finally {
        if (mounted) {
          setIsLoading(false);
        }
      }
    }

    initAuth();

    // 5-day sliding window: periodically roll the refresh token forward every 30 minutes
    const interval = setInterval(
      () => {
        if (getAccessToken()) {
          refreshSession();
        }
      },
      30 * 60 * 1000,
    );

    return () => {
      mounted = false;
      clearInterval(interval);
    };
  }, [refreshSession]);

  const refreshEmails = useCallback(async () => {
    try {
      const res = await authClient.listUserEmails({});
      setEmails(res.emails || []);
    } catch {
      // ignore
    }
  }, [authClient]);

  useEffect(() => {
    if (user) {
      refreshEmails();
    }
  }, [user, refreshEmails]);

  const sendEmailOtp = async (email: string) => {
    const res = await authClient.sendEmailOtp({ email });
    return {
      success: res.success,
      cooldownSeconds: res.cooldownSeconds || 60,
      message: res.message || "Verification code sent to your email",
    };
  };

  const verifyEmailOtp = async (email: string, code: string) => {
    const res = await authClient.verifyEmailOtp({ email, code });
    if (res.accessToken) {
      setAccessToken(res.accessToken);
    }
    if (res.user) {
      setUser(res.user);
    }
    if (res.profile) {
      setProfile(res.profile);
    }
  };

  const verifyMagicLink = async (token: string) => {
    const res = await authClient.verifyMagicLink({ token });
    if (res.accessToken) {
      setAccessToken(res.accessToken);
    }
    if (res.user) {
      setUser(res.user);
    }
    if (res.profile) {
      setProfile(res.profile);
    }
  };

  const updateProfile = async (data: {
    displayName?: string;
    avatarUrl?: string;
    bio?: string;
  }) => {
    const updated = await authClient.updateProfile(data);
    setProfile(updated);
  };

  const sendAddEmailOtp = async (email: string) => {
    const res = await authClient.sendAddEmailOtp({ email });
    return {
      success: res.success,
      cooldownSeconds: res.cooldownSeconds || 60,
      message: res.message || "Verification code sent to your email",
      isExistingAccount: res.isExistingAccount || false,
      existingUserHandle: res.existingUserHandle || "",
      existingBuildCount: res.existingBuildCount || 0,
    };
  };

  const verifyAddEmailOtp = async (email: string, code?: string | null, token?: string | null) => {
    const res = await authClient.verifyAddEmailOtp({
      email,
      code: code || "",
      token: token || "",
    });
    await refreshEmails();
    return res;
  };

  const mergeAccount = async (email: string, code?: string | null, token?: string | null) => {
    const res = await authClient.mergeAccount({
      email,
      code: code || "",
      token: token || "",
    });
    await refreshEmails();
    try {
      const currentUser = await authClient.getCurrentUser({});
      if (currentUser) setUser(currentUser);
    } catch {
      // ignore
    }
    return res;
  };

  const setPrimaryEmail = async (email: string) => {
    const res = await authClient.setPrimaryEmail({ email });
    await refreshEmails();
    return res;
  };

  const removeUserEmail = async (email: string) => {
    const res = await authClient.removeUserEmail({ email });
    await refreshEmails();
    return res;
  };

  const logout = async () => {
    try {
      await authClient.logout({});
    } catch {
      // ignore
    } finally {
      setAccessToken(null);
      setUser(null);
      setProfile(null);
      setEmails([]);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        profile,
        emails,
        isAuthenticated: !!user,
        isLoading,
        sendEmailOtp,
        verifyEmailOtp,
        verifyMagicLink,
        updateProfile,
        logout,
        refreshSession,
        refreshEmails,
        sendAddEmailOtp,
        verifyAddEmailOtp,
        mergeAccount,
        setPrimaryEmail,
        removeUserEmail,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
