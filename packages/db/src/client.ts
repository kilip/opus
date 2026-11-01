import { Database } from "bun:sqlite";
import { existsSync, mkdirSync } from "node:fs";
import { dirname } from "node:path";
import { drizzle } from "drizzle-orm/bun-sqlite";
import { dbEnv } from "./env";
import * as schema from "./schema";

export function createDatabase(dbPath: string = dbEnv.DATABASE_URL) {
  if (dbPath !== ":memory:") {
    const dir = dirname(dbPath);
    if (dir && dir !== "." && !existsSync(dir)) {
      mkdirSync(dir, { recursive: true });
    }
  }

  const sqlite = new Database(dbPath);
  if (dbPath !== ":memory:") {
    sqlite.run("PRAGMA journal_mode = WAL;");
  }
  sqlite.run("PRAGMA foreign_keys = ON;");

  return drizzle(sqlite, { schema });
}

export const db = createDatabase();
export type DatabaseInstance = ReturnType<typeof createDatabase>;
