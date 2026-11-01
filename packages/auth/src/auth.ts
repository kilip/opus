import { db, schema } from "@opus/db";
import { betterAuth } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { emailOTP } from "better-auth/plugins";
import { authEnv } from "./env";
import { sendOTPEmail } from "./mailer";

export const auth = betterAuth({
  secret: authEnv.BETTER_AUTH_SECRET,
  baseURL: authEnv.BETTER_AUTH_URL,
  database: drizzleAdapter(db, {
    provider: "sqlite",
    schema: {
      user: schema.user,
      session: schema.session,
      account: schema.account,
      verification: schema.verification,
    },
  }),
  socialProviders: {
    ...(authEnv.GOOGLE_CLIENT_ID && authEnv.GOOGLE_CLIENT_SECRET
      ? {
          google: {
            clientId: authEnv.GOOGLE_CLIENT_ID,
            clientSecret: authEnv.GOOGLE_CLIENT_SECRET,
          },
        }
      : {}),
    ...(authEnv.GITHUB_CLIENT_ID && authEnv.GITHUB_CLIENT_SECRET
      ? {
          github: {
            clientId: authEnv.GITHUB_CLIENT_ID,
            clientSecret: authEnv.GITHUB_CLIENT_SECRET,
          },
        }
      : {}),
  },
  plugins: [
    emailOTP({
      sendVerificationOTP: sendOTPEmail,
    }),
  ],
});

export type Auth = typeof auth;
export type Session = typeof auth.$Infer.Session.session;
export type User = typeof auth.$Infer.Session.user;
