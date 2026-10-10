import React, { createContext, useContext, useState, useEffect, useCallback, useMemo } from "react";
import { useTransport } from "@connectrpc/connect-query";
import { createClient } from "@connectrpc/connect";
import { AuthService } from "../gen/quadsmith/auth_pb";
import type { User, UserProfile } from "../gen/quadsmith/user_pb";
import { setAccessToken, getAccessToken } from "../lib/api";

export interface AuthContextType {
  user: User | null;
  profile: UserProfile | null;
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
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const transport = useTransport();
  const authClient = useMemo(() => createClient(AuthService, transport), [transport]);
  const [user, setUser] = useState<User | null>(null);
  const [profile, setProfile] = useState<UserProfile | null>(null);
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

  const logout = async () => {
    try {
      await authClient.logout({});
    } catch {
      // ignore
    } finally {
      setAccessToken(null);
      setUser(null);
      setProfile(null);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        profile,
        isAuthenticated: !!user,
        isLoading,
        sendEmailOtp,
        verifyEmailOtp,
        verifyMagicLink,
        updateProfile,
        logout,
        refreshSession,
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
