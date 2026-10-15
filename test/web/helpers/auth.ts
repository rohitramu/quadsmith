import crypto from "node:crypto";
import { Page } from "@playwright/test";

export interface TestUser {
  uuid: string;
  id: string;
  email: string;
  role?: string;
  secret?: string;
}

export const SEEDED_PILOT: TestUser = {
  uuid: "018f0000-0000-7000-8000-000000000001",
  id: "test_pilot",
  email: "pilot@example.com",
  role: "USER",
};

export const SEEDED_ALEX: TestUser = {
  uuid: "018f0000-0000-7000-8000-000000000010",
  id: "alex",
  email: "alex@quadsmith.dev",
  role: "USER",
};

export function createTestJwt(user: TestUser): string {
  const masterSecret = Buffer.from(
    user.secret || process.env.QS_AUTH_SECRET || "quadsmith-dev-secret-key-32bytes-long!",
  );
  const key = crypto.hkdfSync(
    "sha256",
    masterSecret,
    Buffer.alloc(32),
    Buffer.from("quadsmith:access_token"),
    32,
  );

  const now = Math.floor(Date.now() / 1000);
  const header = { alg: "HS256", typ: "JWT" };
  const claims = {
    user_uuid: user.uuid,
    user_id: user.id,
    email: user.email,
    role: user.role || "USER",
    session_uuid: "playwright-e2e-session",
    iat: now,
    exp: now + 86400,
  };

  const b64 = (obj: unknown) => Buffer.from(JSON.stringify(obj)).toString("base64url");
  const payload = `${b64(header)}.${b64(claims)}`;
  const sig = crypto.createHmac("sha256", Buffer.from(key)).update(payload).digest("base64url");

  return `${payload}.${sig}`;
}

export async function loginAs(page: Page, user: TestUser = SEEDED_PILOT): Promise<string> {
  const token = createTestJwt(user);
  await page.addInitScript((jwt) => {
    try {
      localStorage.setItem("qs_access_token", jwt);
    } catch {
      // ignore
    }
  }, token);

  try {
    if (page.url() && page.url() !== "about:blank") {
      await page.evaluate((jwt) => {
        try {
          localStorage.setItem("qs_access_token", jwt);
        } catch {
          // ignore
        }
      }, token);
    }
  } catch {
    // ignore
  }

  return token;
}
