import fs from 'fs-extra';
import path from 'path';
import { nanoid } from 'nanoid';

export interface Config {
  port: number;
  dbPath: string;
  driver: string;
  jwtSecret: string;
}

export async function writeConfig(config: Config) {
  const opusDir = process.env.OPUS_HOME || path.join(process.env.HOME || process.env.USERPROFILE || '', '.opus');
  await fs.ensureDir(opusDir);
  const configFile = path.join(opusDir, 'config.toml');

  const content = `
[server]
port = ${config.port}

[database]
driver = "${config.driver}"
dsn = "${config.dbPath}"

[auth]
secret = "${config.jwtSecret}"
`;

  await fs.writeFile(configFile, content.trim());
}

export function generateJwtSecret(): string {
  return nanoid(32);
}
