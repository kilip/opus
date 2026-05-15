#!/usr/bin/env node

import { intro, outro, select, text, confirm, spinner, note } from '@clack/prompts';
import color from 'picocolors';
import { nanoid } from 'nanoid';
import fs from 'fs-extra';
import path from 'path';
import os from 'os';
import { execa } from 'execa';

async function main() {
  intro(color.bgCyan(color.black(' Opus AI Installation Wizard ')));

  const opusDir = path.join(os.homedir(), '.opus');
  const configFile = path.join(opusDir, 'config.toml');
  const dbFile = path.join(opusDir, 'opus.db');

  // Ensure .opus directory exists
  await fs.ensureDir(opusDir);

  const config = {
    server: {
      host: '0.0.0.0',
      port: 8080,
      env: 'production'
    },
    database: {
      driver: 'sqlite',
      dsn: dbFile
    },
    auth: {
      secret: nanoid(32),
      access_token_ttl: 15,
      refresh_token_ttl: 10080
    }
  };

  const mode = await select({
    message: 'Select installation mode:',
    options: [
      { value: 'production', label: 'Production (Recommended)', hint: 'Optimized for stability' },
      { value: 'development', label: 'Development', hint: 'For debugging and local dev' }
    ]
  });

  config.server.env = mode;

  const port = await text({
    message: 'API Port:',
    placeholder: '8080',
    initialValue: '8080',
    validate(value) {
      if (isNaN(parseInt(value))) return 'Please enter a valid number';
    }
  });

  config.server.port = parseInt(port);

  const customDb = await confirm({
    message: 'Use default SQLite database at ~/.opus/opus.db?',
    initialValue: true
  });

  if (!customDb) {
    const dsn = await text({
      message: 'Enter Database DSN (Postgres or custom SQLite path):',
      placeholder: 'user:pass@host:port/dbname'
    });
    config.database.dsn = dsn;
    if (dsn.includes('postgres')) {
        config.database.driver = 'postgres';
    }
  }

  const s = spinner();
  s.start('Writing configuration to ' + configFile);
  
  // Convert object to simple TOML-ish string (minimal requirement)
  const tomlContent = `
[server]
host = "${config.server.host}"
port = ${config.server.port}
env = "${config.server.env}"

[database]
driver = "${config.database.driver}"
dsn = "${config.database.dsn}"

[auth]
secret = "${config.auth.secret}"
access_token_ttl = ${config.auth.access_token_ttl}
refresh_token_ttl = ${config.auth.refresh_token_ttl}
`;

  await fs.writeFile(configFile, tomlContent.trim());
  s.stop('Configuration saved!');

  note(
    `Auth Secret: ${color.yellow(config.auth.secret)}\nKeep this secret safe!`,
    'Security Note'
  );

  const deploy = await confirm({
    message: 'Would you like to deploy using Docker Compose now?',
    initialValue: false
  });

  if (deploy) {
    s.start('Launching Opus via Docker Compose...');
    try {
      await execa('docker-compose', ['up', '-d']);
      s.stop('Opus is running!');
      outro(color.green('Installation complete! Visit http://localhost:3000'));
    } catch (err) {
      s.stop(color.red('Failed to launch Docker Compose.'));
      console.error(err.message);
      outro('Config saved, but you may need to run docker-compose manually.');
    }
  } else {
    outro(color.green('Setup complete! You can now start the API with: OPUS_SERVER_ENV=production opus start'));
  }
}

main().catch(console.error);
