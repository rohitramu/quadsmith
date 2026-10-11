import React, { createContext, useContext, useState, useEffect, useCallback, useMemo } from "react";
import { useTransport } from "@connectrpc/connect-query";
import { createClient } from "@connectrpc/connect";
import { AuthService } from "../gen/quadsmith/auth_pb";
import type {
  ChangeUsernameResponse,
  ListArchivedUsernamesResponse,
} from "../gen/quadsmith/auth_pb";
import { UserService } from "../gen/quadsmith/user_pb";
import type {
  User,
  UserProfile,
  UserEmail,
  UsernameAvailabilityStatus,
} from "../gen/quadsmith/user_pb";
import { setAccessToken, getAccessToken } from "../lib/api";

export interface AuthContextType {
  user: User | null;
  profile: UserProfile | null;
  emails: UserEmail[];
  isAuthenticated: boolean;
  isLoading: boolean;
  challengeId: string | null;
  sendEmailOtp: (email: string) => Promise<{
    success: boolean;
    cooldownSeconds: number;
    message: string;
    challengeId?: string;
  }>;
  verifyEmailOtp: (
    email: string,
    code: string,
    challengeId?: string,
  ) => Promise<{ accountSetupRequired?: boolean; setupToken?: string }>;
  verifyMagicLink: (
    token: string,
  ) => Promise<{ accountSetupRequired?: boolean; setupToken?: string }>;
  completeAccountSetup: (setupToken: string, username: string, avatarUrl?: string) => Promise<void>;
  updateProfile: (data: { avatarUrl?: string; bio?: string }) => Promise<void>;
  logout: () => Promise<void>;
  refreshSession: () => Promise<void>;
  refreshEmails: () => Promise<void>;
  sendAddEmailOtp: (email: string) => Promise<{
    success: boolean;
    cooldownSeconds: number;
    message: string;
    isExistingAccount?: boolean;
    challengeId?: string;
  }>;
  verifyAddEmailOtp: (
    email: string,
    code?: string | null,
    token?: string | null,
    challengeId?: string | null,
  ) => Promise<{
    success: boolean;
    message: string;
    mergeRequired: boolean;
    otherUserUsername?: string;
    otherBuildCount?: number;
    otherBuildNames?: string[];
    otherEmails?: string[];
    otherOauthProviders?: string[];
    mergeTicket?: string;
  }>;
  mergeAccount: (
    email: string,
    code?: string | null,
    token?: string | null,
    mergeTicket?: string | null,
    challengeId?: string | null,
  ) => Promise<{
    success: boolean;
    message: string;
    transferredBuildCount: number;
    mergedUsername: string;
    mergedEmails?: string[];
  }>;
  setPrimaryEmail: (email: string) => Promise<{ success: boolean; message: string }>;
  removeUserEmail: (email: string) => Promise<{ success: boolean; message: string }>;
  changeUsername: (newUsername: string) => Promise<ChangeUsernameResponse>;
  listArchivedUsernames: () => Promise<ListArchivedUsernamesResponse>;
  checkUsernameAvailable: (username: string) => Promise<{
    status: UsernameAvailabilityStatus;
    available: boolean;
    message: string;
  }>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const transport = useTransport();
  const authClient = useMemo(() => createClient(AuthService, transport), [transport]);
  const [user, setUser] = useState<User | null>(null);
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [emails, setEmails] = useState<UserEmail[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [challengeId, setChallengeId] = useState<string | null>(() => {
    try {
      return sessionStorage.getItem("qs_challenge_id");
    } catch {
      return null;
    }
  });
  const [mergeTicket, setMergeTicket] = useState<string | null>(null);

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
    if (res.challengeId) {
      setChallengeId(res.challengeId);
      try {
        sessionStorage.setItem("qs_challenge_id", res.challengeId);
      } catch {
        // ignore
      }
    }
    return {
      success: res.success,
      cooldownSeconds: res.cooldownSeconds || 20,
      message: res.message || "Verification code sent to your email",
      challengeId: res.challengeId,
    };
  };

  const verifyEmailOtp = async (email: string, code: string, explicitChallengeId?: string) => {
    const cid =
      explicitChallengeId ||
      challengeId ||
      (() => {
        try {
          return sessionStorage.getItem("qs_challenge_id");
        } catch {
          return null;
        }
      })() ||
      undefined;

    const res = await authClient.verifyEmailOtp({ email, code, challengeId: cid });
    setChallengeId(null);
    try {
      sessionStorage.removeItem("qs_challenge_id");
    } catch {
      // ignore
    }
    if (res.accessToken) {
      setAccessToken(res.accessToken);
    }
    if (res.user) {
      setUser(res.user);
    }
    if (res.profile) {
      setProfile(res.profile);
    }
    return {
      accountSetupRequired: res.accountSetupRequired,
      setupToken: res.setupToken,
    };
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
    return {
      accountSetupRequired: res.accountSetupRequired,
      setupToken: res.setupToken,
    };
  };

  const completeAccountSetup = async (setupToken: string, username: string, avatarUrl?: string) => {
    const res = await authClient.completeAccountSetup({
      setupToken,
      username,
      avatarUrl: avatarUrl || undefined,
    });
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

  const updateProfile = async (data: { avatarUrl?: string; bio?: string }) => {
    const updated = await authClient.updateProfile(data);
    setProfile(updated);
  };

  const sendAddEmailOtp = async (email: string) => {
    const res = await authClient.sendAddEmailOtp({ email });
    if (res.challengeId) {
      setChallengeId(res.challengeId);
      try {
        sessionStorage.setItem("qs_challenge_id", res.challengeId);
      } catch {
        // ignore
      }
    }
    return {
      success: res.success,
      cooldownSeconds: res.cooldownSeconds || 20,
      message: res.message || "Verification code sent to your email",
      isExistingAccount: res.isExistingAccount || false,
      challengeId: res.challengeId,
    };
  };

  const verifyAddEmailOtp = async (
    email: string,
    code?: string | null,
    token?: string | null,
    explicitChallengeId?: string | null,
  ) => {
    const cid =
      explicitChallengeId ||
      challengeId ||
      (() => {
        try {
          return sessionStorage.getItem("qs_challenge_id");
        } catch {
          return null;
        }
      })() ||
      undefined;

    const res = await authClient.verifyAddEmailOtp({
      email,
      code: code || "",
      token: token || "",
      challengeId: cid,
    });
    if (res.mergeTicket) {
      setMergeTicket(res.mergeTicket);
    }
    await refreshEmails();
    return res;
  };

  const mergeAccount = async (
    email: string,
    code?: string | null,
    token?: string | null,
    explicitMergeTicket?: string | null,
    explicitChallengeId?: string | null,
  ) => {
    const cid =
      explicitChallengeId ||
      challengeId ||
      (() => {
        try {
          return sessionStorage.getItem("qs_challenge_id");
        } catch {
          return null;
        }
      })() ||
      undefined;
    const ticket = explicitMergeTicket || mergeTicket || undefined;

    const res = await authClient.mergeAccount({
      email,
      code: code || "",
      token: token || "",
      mergeTicket: ticket,
      challengeId: cid,
    });
    setMergeTicket(null);
    setChallengeId(null);
    try {
      sessionStorage.removeItem("qs_challenge_id");
    } catch {
      // ignore
    }
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

  const userClient = useMemo(() => createClient(UserService, transport), [transport]);

  const changeUsername = async (newUsername: string) => {
    const res = await authClient.changeUsername({ newUsername });
    if (res.user) {
      setUser(res.user);
    } else {
      try {
        const currentUser = await authClient.getCurrentUser({});
        if (currentUser) setUser(currentUser);
      } catch {
        // ignore
      }
    }
    return res;
  };

  const listArchivedUsernames = async () => {
    return await authClient.listArchivedUsernames({});
  };

  const checkUsernameAvailable = async (username: string) => {
    const res = await userClient.checkUsernameAvailable({ username });
    return {
      status: res.status,
      available: res.available,
      message: res.message,
    };
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
      setChallengeId(null);
      setMergeTicket(null);
      try {
        sessionStorage.removeItem("qs_challenge_id");
      } catch {
        // ignore
      }
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
        challengeId,
        sendEmailOtp,
        verifyEmailOtp,
        verifyMagicLink,
        completeAccountSetup,
        updateProfile,
        logout,
        refreshSession,
        refreshEmails,
        sendAddEmailOtp,
        verifyAddEmailOtp,
        mergeAccount,
        setPrimaryEmail,
        removeUserEmail,
        changeUsername,
        listArchivedUsernames,
        checkUsernameAvailable,
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
