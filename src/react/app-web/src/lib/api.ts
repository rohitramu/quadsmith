import { createClient } from "@connectrpc/connect";
import { AuthService } from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import { UserService } from "@quadsmith/shared/gen/quadsmith/api/user_pb";
import {
  createApiTransport,
  getApiBaseUrl,
  getStoredAccessToken,
  setStoredAccessToken,
} from "@quadsmith/shared";

export { getApiBaseUrl };
export const setAccessToken = setStoredAccessToken;
export const getAccessToken = getStoredAccessToken;

export const transport = createApiTransport();

export const authClient = createClient(AuthService, transport);
export const userClient = createClient(UserService, transport);
