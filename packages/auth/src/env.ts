import { createEnv } from "@t3-oss/env-core";
import { z } from "zod";

export const authEnv = createEnv({
  server: {
    BETTER_AUTH_SECRET: z
      .string()
      .min(1)
      .default("development-auth-secret-change-in-production-1234567890"),
    BETTER_AUTH_URL: z.string().url().default("http://localhost:3000"),
    GOOGLE_CLIENT_ID: z.string().optional(),
    GOOGLE_CLIENT_SECRET: z.string().optional(),
    GITHUB_CLIENT_ID: z.string().optional(),
    GITHUB_CLIENT_SECRET: z.string().optional(),
    SMTP_HOST: z.string().optional(),
    SMTP_PORT: z.coerce.number().default(587),
    SMTP_USER: z.string().optional(),
    SMTP_PASSWORD: z.string().optional(),
    SMTP_SECURE: z
      .string()
      .optional()
      .transform((val) => val === "true"),
    EMAIL_FROM: z.string().default("Opus <noreply@opus.local>"),
  },
  runtimeEnv: process.env,
  emptyStringAsUndefined: true,
});
