import os from 'os';

export interface PlatformInfo {
  os: string;
  arch: string;
  binaryName: string;
}

export function detectPlatform(): PlatformInfo {
  const osType = os.platform(); // 'linux', 'darwin', 'win32'
  const archType = os.arch();   // 'x64', 'arm64'

  let binaryName = 'opus';

  if (osType === 'linux') {
    binaryName = archType === 'arm64' ? 'opus-linux-arm64' : 'opus-linux-amd64';
  } else if (osType === 'darwin') {
    binaryName = archType === 'arm64' ? 'opus-darwin-arm64' : 'opus-darwin-amd64';
  } else if (osType === 'win32') {
    binaryName = 'opus-windows-amd64.exe';
  } else {
    throw new Error(`Unsupported platform: ${osType}-${archType}`);
  }

  return { os: osType, arch: archType, binaryName };
}
