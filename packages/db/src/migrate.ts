import { join } from "node:path";
import { migrate } from "drizzle-orm/bun-sqlite/migrator";
import { type DatabaseInstance, db } from "./client";

export function runMigrations(
  targetDb: DatabaseInstance = db,
  migrationsFolder = join(import.meta.dir, "../drizzle"),
) {
  migrate(targetDb, { migrationsFolder });
}
