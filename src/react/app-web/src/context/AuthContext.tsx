import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useMemo,
  useRef,
} from "react";
import { useTransport } from "@connectrpc/connect-query";
import { createClient } from "@connectrpc/connect";
import {
  AuthService,
  EmailOtpPurpose,
  LoginChallengeStatus,
  OAuthProvider,
  OAuthPurpose,
} from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import type {
  ChangeUsernameResponse,
  ListArchivedUsernamesResponse,
  GetLoginChallengeDetailsResponse,
  ApproveLoginChallengeResponse,
  DenyLoginChallengeResponse,
  GetLoginChallengeStatusResponse,
  SendEmailOtpResponse,
  VerifyEmailOtpResponse,
  CompleteOAuthResponse,
  Session,
} from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import { UserService } from "@quadsmith/shared/gen/quadsmith/api/user_pb";
import type {
  User,
  UserProfile,
  UserEmail,
  UsernameAvailabilityStatus,
} from "@quadsmith/shared/gen/quadsmith/api/user_pb";
import { setAccessToken, getAccessToken } from "../lib/api";

export interface AuthContextType {
  user: User | null;
  profile: UserProfile | null;
  emails: UserEmail[];
  isAuthenticated: boolean;
  isLoading: boolean;
  challengeId: string | null;
  sendEmailOtp: (
    email?: string,
    purpose?: EmailOtpPurpose,
  ) => Promise<{
    success: boolean;
    cooldownSeconds: number;
    message: string;
    isExistingAccount?: boolean;
    challengeId?: string;
    email?: string;
  }>;
  verifyEmailOtp: (
    emailOrParams:
      | string
      | {
          email?: string;
          code?: string;
          token?: string;
          challengeId?: string | null;
          purpose?: EmailOtpPurpose;
        },
    code?: string,
    challengeId?: string,
    purpose?: EmailOtpPurpose,
  ) => Promise<VerifyEmailOtpResponse>;
  verifyMagicLink: (
    token: string,
  ) => Promise<{ accountSetupRequired?: boolean; setupToken?: string }>;
  getLoginChallengeDetails: (token: string) => Promise<GetLoginChallengeDetailsResponse>;
  approveLoginChallenge: (token: string) => Promise<ApproveLoginChallengeResponse>;
  denyLoginChallenge: (token: string) => Promise<DenyLoginChallengeResponse>;
  getLoginChallengeStatus: (
    challengeId?: string | null,
  ) => Promise<GetLoginChallengeStatusResponse>;
  completeAccountSetup: (setupToken: string, username: string, avatarUrl?: string) => Promise<void>;
  updateProfile: (data: { avatarUrl?: string; bio?: string }) => Promise<void>;
  logout: () => Promise<void>;
  refreshSession: () => Promise<void>;
  refreshEmails: () => Promise<void>;
  sessions: Session[];
  refreshSessions: () => Promise<void>;
  revokeSession: (sessionUuid: string) => Promise<void>;
  revokeOtherSessions: () => Promise<{ revokedCount: number }>;
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
  sendDataExportVerification: () => Promise<SendEmailOtpResponse>;
  verifyDataExport: (
    challengeId?: string | null,
    code?: string | null,
    magicToken?: string | null,
  ) => Promise<VerifyEmailOtpResponse>;
  startOAuth: (
    provider: OAuthProvider,
    purpose?: OAuthPurpose,
    redirectPath?: string,
  ) => Promise<{ authorizationUrl: string; state: string; challengeId?: string }>;
  completeOAuth: (code: string, state: string) => Promise<CompleteOAuthResponse>;
  unlinkOAuth: (provider: OAuthProvider) => Promise<{ success: boolean; message: string }>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const transport = useTransport();
  const authClient = useMemo(() => createClient(AuthService, transport), [transport]);
  const [user, setUser] = useState<User | null>(null);
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [emails, setEmails] = useState<UserEmail[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [challengeId, setChallengeId] = useState<string | null>(() => {
    try {
      return sessionStorage.getItem("qs_challenge_id");
    } catch {
      return null;
    }
  });
  const [mergeTicket, setMergeTicket] = useState<string | null>(null);
  const authMutationRef = useRef(0);

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
  }, [authClient]);

  useEffect(() => {
    let mounted = true;
    const startMutation = authMutationRef.current;

    async function initAuth() {
      try {
        if (typeof window !== "undefined") {
          const params = new URLSearchParams(window.location.search);
          const opsRefreshToken = params.get("ops_refresh_token");
          if (opsRefreshToken) {
            params.delete("ops_refresh_token");
            const remaining = params.toString();
            const cleanUrl =
              window.location.pathname + (remaining ? `?${remaining}` : "") + window.location.hash;
            window.history.replaceState({}, document.title, cleanUrl);

            const res = await authClient.refreshToken({ refreshToken: opsRefreshToken });
            if (mounted && authMutationRef.current === startMutation) {
              if (res.accessToken) setAccessToken(res.accessToken);
              if (res.user) setUser(res.user);
              if (res.profile) setProfile(res.profile);
              try {
                const emailsRes = await authClient.listUserEmails({});
                if (mounted && authMutationRef.current === startMutation) {
                  setEmails(emailsRes.emails || []);
                }
              } catch {
                // ignore
              }
            }
            return;
          }
        }

        const token = getAccessToken();
        if (token) {
          try {
            const currentUser = await authClient.getCurrentUser({});
            if (mounted && authMutationRef.current === startMutation && currentUser) {
              setUser(currentUser);
              try {
                const emailsRes = await authClient.listUserEmails({});
                if (mounted && authMutationRef.current === startMutation) {
                  setEmails(emailsRes.emails || []);
                }
              } catch {
                // ignore
              }
              // Profile refresh via silent refresh
              try {
                const refreshRes = await authClient.refreshToken({});
                if (mounted && authMutationRef.current === startMutation) {
                  if (refreshRes.accessToken) setAccessToken(refreshRes.accessToken);
                  if (refreshRes.profile) setProfile(refreshRes.profile);
                }
              } catch {
                // Refresh cookie may not be present (e.g. bearer-only session)
              }
              return;
            }
          } catch {
            // Token might be expired, try refresh cookie
          }
        }
        if (!mounted || authMutationRef.current !== startMutation) return;
        // Try refresh token from cookie
        const res = await authClient.refreshToken({});
        if (mounted && authMutationRef.current === startMutation) {
          if (res.accessToken) setAccessToken(res.accessToken);
          if (res.user) setUser(res.user);
          if (res.profile) setProfile(res.profile);
          try {
            const emailsRes = await authClient.listUserEmails({});
            if (mounted && authMutationRef.current === startMutation) {
              setEmails(emailsRes.emails || []);
            }
          } catch {
            // ignore
          }
        }
      } catch {
        if (mounted && authMutationRef.current === startMutation) {
          setAccessToken(null);
          setUser(null);
          setProfile(null);
        }
      } finally {
        if (mounted && authMutationRef.current === startMutation) {
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

  const refreshSessions = useCallback(async () => {
    try {
      const res = await authClient.listActiveSessions({});
      setSessions(res.sessions || []);
    } catch {
      // ignore
    }
  }, [authClient]);

  useEffect(() => {
    if (user) {
      refreshEmails();
      refreshSessions();
    }
  }, [user, refreshEmails, refreshSessions]);

  const revokeSession = async (sessionUuid: string) => {
    await authClient.revokeSession({ sessionUuid });
    await refreshSessions();
  };

  const revokeOtherSessions = async () => {
    const res = await authClient.revokeOtherSessions({});
    await refreshSessions();
    return { revokedCount: res.revokedCount };
  };

  const sendEmailOtp = async (email?: string, purpose: EmailOtpPurpose = EmailOtpPurpose.LOGIN) => {
    const res = await authClient.sendEmailOtp({
      email: email || "",
      purpose,
    });
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
      cooldownSeconds: res.cooldownSeconds ?? 20,
      message: res.message || "Verification code sent to your email",
      isExistingAccount: res.isExistingAccount,
      challengeId: res.challengeId,
      email: res.email,
    };
  };

  const verifyEmailOtp = async (
    emailOrParams:
      | string
      | {
          email?: string;
          code?: string;
          token?: string;
          challengeId?: string | null;
          purpose?: EmailOtpPurpose;
        },
    explicitCode?: string,
    explicitChallengeId?: string,
    explicitPurpose?: EmailOtpPurpose,
  ): Promise<VerifyEmailOtpResponse> => {
    let email: string = "";
    let code: string = "";
    let token: string = "";
    let cid: string | undefined = undefined;
    let purpose: EmailOtpPurpose = EmailOtpPurpose.LOGIN;

    if (typeof emailOrParams === "object") {
      email = emailOrParams.email || "";
      code = emailOrParams.code || "";
      token = emailOrParams.token || "";
      cid =
        emailOrParams.challengeId ||
        challengeId ||
        (() => {
          try {
            return sessionStorage.getItem("qs_challenge_id");
          } catch {
            return null;
          }
        })() ||
        undefined;
      purpose = emailOrParams.purpose ?? EmailOtpPurpose.LOGIN;
    } else {
      email = emailOrParams;
      code = explicitCode || "";
      cid =
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
      purpose = explicitPurpose ?? EmailOtpPurpose.LOGIN;
    }

    const res = await authClient.verifyEmailOtp({
      email,
      code,
      challengeId: cid,
      token,
      purpose,
    });

    setChallengeId(null);
    try {
      sessionStorage.removeItem("qs_challenge_id");
    } catch {
      // ignore
    }

    if (res.accessToken || res.user) {
      authMutationRef.current += 1;
      setIsLoading(false);
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
    if (res.mergeTicket) {
      setMergeTicket(res.mergeTicket);
    }
    if (purpose === EmailOtpPurpose.ADD_EMAIL) {
      await refreshEmails();
    }

    return res;
  };

  const verifyMagicLink = async (token: string) => {
    const res = await authClient.verifyMagicLink({ token });
    if (res.accessToken || res.user) {
      authMutationRef.current += 1;
      setIsLoading(false);
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

  const getLoginChallengeDetails = useCallback(
    async (token: string) => {
      return await authClient.getLoginChallengeDetails({ token });
    },
    [authClient],
  );

  const approveLoginChallenge = useCallback(
    async (token: string) => {
      return await authClient.approveLoginChallenge({ token });
    },
    [authClient],
  );

  const denyLoginChallenge = useCallback(
    async (token: string) => {
      return await authClient.denyLoginChallenge({ token });
    },
    [authClient],
  );

  const getLoginChallengeStatus = async (explicitChallengeId?: string | null) => {
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

    const res = await authClient.getLoginChallengeStatus({ challengeId: cid });
    if (res.status === LoginChallengeStatus.APPROVED) {
      if (res.verifyResponse?.mergeTicket) {
        setMergeTicket(res.verifyResponse.mergeTicket);
      }
      if (res.authResponse) {
        if (res.authResponse.accessToken || res.authResponse.user) {
          authMutationRef.current += 1;
          setIsLoading(false);
        }
        if (res.authResponse.accessToken) {
          setAccessToken(res.authResponse.accessToken);
        }
        if (res.authResponse.user) {
          setUser(res.authResponse.user);
        }
        if (res.authResponse.profile) {
          setProfile(res.authResponse.profile);
        }
        setChallengeId(null);
        try {
          sessionStorage.removeItem("qs_challenge_id");
        } catch {
          // ignore
        }
      }
    }
    return res;
  };

  const completeAccountSetup = useCallback(
    async (setupToken: string, username: string, avatarUrl?: string) => {
      const res = await authClient.completeAccountSetup({
        setupToken,
        username,
        avatarUrl: avatarUrl || undefined,
      });
      authMutationRef.current += 1;
      setIsLoading(false);
      if (res.accessToken) {
        setAccessToken(res.accessToken);
      }
      if (res.user) {
        setUser(res.user);
      }
      if (res.profile) {
        setProfile(res.profile);
      }
    },
    [authClient],
  );

  const updateProfile = async (data: { avatarUrl?: string; bio?: string }) => {
    const updated = await authClient.updateProfile(data);
    setProfile(updated);
  };

  const sendAddEmailOtp = async (email: string) => {
    return await sendEmailOtp(email, EmailOtpPurpose.ADD_EMAIL);
  };

  const verifyAddEmailOtp = async (
    email: string,
    code?: string | null,
    token?: string | null,
    explicitChallengeId?: string | null,
  ) => {
    return await verifyEmailOtp({
      email,
      code: code || "",
      token: token || "",
      challengeId: explicitChallengeId,
      purpose: EmailOtpPurpose.ADD_EMAIL,
    });
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

  const checkUsernameAvailable = useCallback(
    async (username: string) => {
      const res = await userClient.checkUsernameAvailable({ username });
      return {
        status: res.status,
        available: res.available,
        message: res.message,
      };
    },
    [userClient],
  );

  const logout = useCallback(async () => {
    authMutationRef.current += 1;
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
      setIsLoading(false);
      try {
        sessionStorage.removeItem("qs_challenge_id");
      } catch {
        // ignore
      }
    }
  }, [authClient]);

  const sendDataExportVerification = async (): Promise<SendEmailOtpResponse> => {
    return await authClient.sendEmailOtp({
      email: "",
      purpose: EmailOtpPurpose.EXPORT_DATA,
    });
  };

  const verifyDataExport = async (
    challengeId?: string | null,
    code?: string | null,
    magicToken?: string | null,
  ): Promise<VerifyEmailOtpResponse> => {
    return await verifyEmailOtp({
      challengeId,
      code: code || "",
      token: magicToken || "",
      purpose: EmailOtpPurpose.EXPORT_DATA,
    });
  };

  const startOAuth = useCallback(
    async (
      provider: OAuthProvider,
      purpose: OAuthPurpose = OAuthPurpose.OAUTH_PURPOSE_LOGIN,
      redirectPath?: string,
    ) => {
      const res = await authClient.startOAuth({
        provider,
        purpose,
        redirectPath: redirectPath || "/login",
      });
      return {
        authorizationUrl: res.authorizationUrl,
        state: res.state,
        challengeId: res.challengeId,
      };
    },
    [authClient],
  );

  const completeOAuth = useCallback(
    async (code: string, state: string): Promise<CompleteOAuthResponse> => {
      const res = await authClient.completeOAuth({ code, state });
      authMutationRef.current += 1;
      setIsLoading(false);
      if (res.mergeTicket) {
        setMergeTicket(res.mergeTicket);
      }
      if (res.authResponse?.accessToken) {
        setAccessToken(res.authResponse.accessToken);
        if (res.authResponse.user) {
          setUser(res.authResponse.user);
        }
        if (res.authResponse.profile) {
          setProfile(res.authResponse.profile);
        }
        await refreshEmails();
      } else if (res.success && res.purpose === OAuthPurpose.OAUTH_PURPOSE_ADD_EMAIL) {
        await refreshEmails();
      }
      return res;
    },
    [authClient, refreshEmails],
  );

  const unlinkOAuth = useCallback(
    async (provider: OAuthProvider) => {
      const res = await authClient.unlinkOAuth({ provider });
      await refreshEmails();
      return { success: res.success, message: res.message };
    },
    [authClient, refreshEmails],
  );

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
        getLoginChallengeDetails,
        approveLoginChallenge,
        denyLoginChallenge,
        getLoginChallengeStatus,
        completeAccountSetup,
        updateProfile,
        logout,
        refreshSession,
        refreshEmails,
        sessions,
        refreshSessions,
        revokeSession,
        revokeOtherSessions,
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
        startOAuth,
        completeOAuth,
        unlinkOAuth,
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

export function useOptionalAuth(): AuthContextType | undefined {
  return useContext(AuthContext);
}
