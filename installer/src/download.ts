import fetch from 'node-fetch';
import fs from 'fs-extra';
import path from 'path';

export async function downloadBinary(binaryName: string, destination: string) {
  const url = `https://github.com/kilip/opus/releases/latest/download/${binaryName}`;
  const response = await fetch(url);
  if (!response.ok) throw new Error(`Failed to download ${binaryName}: ${response.statusText}`);
  
  await fs.ensureDir(path.dirname(destination));
  const fileStream = fs.createWriteStream(destination, { mode: 0o755 });
  await new Promise<void>((resolve, reject) => {
    response.body!.pipe(fileStream);
    response.body!.on('error', reject);
    fileStream.on('finish', () => resolve());
  });
}
