import { createEnv } from "@t3-oss/env-core";
import { z } from "zod";

export const dbEnv = createEnv({
  server: {
    DATABASE_URL: z.string().default("./data/opus.sqlite"),
  },
  runtimeEnv: process.env,
  emptyStringAsUndefined: true,
});
