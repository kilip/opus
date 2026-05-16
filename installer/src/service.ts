import os from "node:os";
import path from "node:path";
import { execa } from "execa";
import fs from "fs-extra";

export async function registerService() {
  const osType = os.platform();
  const user = os.userInfo().username;
  const home = os.homedir();

  if (osType === "linux") {
    const servicePath = "/etc/systemd/system/opus.service";
    let template = await fs.readFile(
      path.join(__dirname, "../templates/opus.service"),
      "utf8",
    );
    template = template.replace(/{{USER}}/g, user);

    await fs.writeFile(servicePath, template);
    await execa("systemctl", ["daemon-reload"]);
    await execa("systemctl", ["enable", "--now", "opus"]);
  } else if (osType === "darwin") {
    const plistPath = path.join(
      home,
      "Library/LaunchAgents/com.opus.agent.plist",
    );
    let template = await fs.readFile(
      path.join(__dirname, "../templates/com.opus.agent.plist"),
      "utf8",
    );
    template = template.replace(/{{HOME}}/g, home);

    await fs.writeFile(plistPath, template);
    await execa("launchctl", ["load", plistPath]);
  } else if (osType === "win32") {
    await execa("sc.exe", [
      "create",
      "Opus",
      "binPath=",
      "C:ProgramDataOpusopus.exe start",
    ]);
  }
}
