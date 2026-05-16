import { intro, outro, select, text, confirm, spinner } from '@clack/prompts';
import color from 'picocolors';
import { detectPlatform } from './platform';
import { downloadBinary } from './download';
import { writeConfig, generateJwtSecret } from './config';
import { registerService } from './service';
import os from 'os';
import path from 'path';

export async function runInstaller() {
  intro(color.bgCyan(color.black(' Opus AI Installation Wizard ')));

  // 1. Detect Platform
  const s1 = spinner();
  s1.start('[1/5] Detecting platform...');
  const platformInfo = detectPlatform();
  s1.stop(`[1/5] Detected: ${platformInfo.os}/${platformInfo.arch}`);

  // 2. Download Binary
  const s2 = spinner();
  s2.start('[2/5] Downloading binary...');
  const dest = os.platform() === 'win32' ? 'C:\ProgramData\Opus\opus.exe' : '/usr/local/bin/opus';
  await downloadBinary(platformInfo.binaryName, dest);
  s2.stop('[2/5] Binary downloaded');

  // 3. Configure
  const s3 = spinner();
  s3.start('[3/5] Configuring...');
  const port = await text({ message: 'Server port:', initialValue: '8080' });
  const defaultOpusDir = process.env.OPUS_HOME || (await fs.pathExists('.opus') ? path.resolve('.opus') : path.join(os.homedir(), '.opus'));
  const dbPath = await text({ message: 'Database path:', initialValue: path.join(defaultOpusDir, 'opus.db') });
  const jwtSecret = await text({ message: 'JWT Secret (blank to auto-generate):' });
  
  await writeConfig({
    port: parseInt(port as string),
    dbPath: dbPath as string,
    driver: 'sqlite',
    jwtSecret: (jwtSecret as string) || generateJwtSecret()
  });
  s3.stop('[3/5] Configured');

  // 4. Register Service
  const s4 = spinner();
  s4.start('[4/5] Installing system service...');
  await registerService();
  s4.stop('[4/5] Service installed');

  outro(color.green('✓ Opus is installed and running!'));
}
