import { defineConfig } from "drizzle-kit";
import { dbEnv } from "./src/env";

export default defineConfig({
  schema: "./src/schema/index.ts",
  out: "./drizzle",
  dialect: "sqlite",
  dbCredentials: {
    url: dbEnv.DATABASE_URL,
  },
});
