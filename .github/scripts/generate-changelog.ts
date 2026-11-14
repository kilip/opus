import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { execSync } from "node:child_process";
import { join } from "node:path";

interface CommitInfo {
  hash: string;
  shortHash: string;
  type: string;
  scope: string;
  subject: string;
}

const SECTION_CONFIG: Array<{
  key: string;
  title: string;
  match: (c: CommitInfo) => boolean;
}> = [
  { key: "feat", title: "✨ Features", match: (c) => c.type === "feat" },
  { key: "fix", title: "🐛 Bug Fixes", match: (c) => c.type === "fix" },
  {
    key: "perf",
    title: "⚡ Performance Improvements",
    match: (c) => c.type === "perf",
  },
  { key: "revert", title: "⏪ Reverts", match: (c) => c.type === "revert" },
  {
    key: "deps",
    title: "📦 Dependencies",
    match: (c) =>
      c.type === "deps" || (c.type === "chore" && c.scope === "deps"),
  },
  {
    key: "deploy",
    title: "🚀 Deployment",
    match: (c) => c.type === "deploy" || c.scope === "deploy",
  },
  {
    key: "refactor",
    title: "♻️ Code Refactoring",
    match: (c) => c.type === "refactor",
  },
  { key: "docs", title: "📝 Documentation", match: (c) => c.type === "docs" },
  { key: "test", title: "🧪 Tests", match: (c) => c.type === "test" },
  {
    key: "ci",
    title: "👷 Continuous Integration",
    match: (c) => c.type === "ci",
  },
  {
    key: "tools",
    title: "🔧 Tooling & Chores",
    match: (c) => ["chore", "tools", "build", "style"].includes(c.type),
  },
];

function sh(cmd: string): string {
  try {
    return execSync(cmd, { encoding: "utf-8" }).trim();
  } catch {
    return "";
  }
}

function cleanUnicode(str: string): string {
  return str
    .replace(/[^\x20-\x7E]/g, "")
    .replace(/\s+/g, " ")
    .replace(/^[:\-\s]+/, "")
    .trim();
}

function getRepoUrl(): string {
  const ghRepo = process.env.GITHUB_REPOSITORY;
  if (ghRepo) return `https://github.com/${ghRepo}`;
  const remote = sh("git config --get remote.origin.url");
  if (remote) {
    const match = remote.match(/[:/]([^/:]+\/[^/.]+)(?:\.git)?$/);
    if (match) return `https://github.com/${match[1]}`;
  }
  return "https://github.com/kilip/opus";
}

function getVersion(): string {
  const args = process.argv.slice(2);
  const vIndex = args.indexOf("--version");
  if (vIndex !== -1 && args[vIndex + 1]) return args[vIndex + 1];

  const manifestPath = join(process.cwd(), ".github/release-manifest.json");
  if (existsSync(manifestPath)) {
    try {
      const manifest = JSON.parse(readFileSync(manifestPath, "utf-8"));
      if (manifest["."]) return manifest["."];
    } catch {
      // ignore
    }
  }
  return "0.1.0";
}

function getCommits(prevTag: string): CommitInfo[] {
  const range = prevTag ? `${prevTag}..HEAD` : "HEAD";
  const logOutput = sh(`git log ${range} --no-merges --format="%H|%s"`);
  if (!logOutput) return [];

  const commits: CommitInfo[] = [];
  for (const line of logOutput.split("\n")) {
    const pipeIdx = line.indexOf("|");
    if (pipeIdx === -1) continue;
    const hash = line.slice(0, pipeIdx);
    const rawMsg = line.slice(pipeIdx + 1);

    const ccMatch = rawMsg.match(/^(\w+)(?:\(([^)]+)\))?!?: (.+)$/);
    if (ccMatch) {
      const type = ccMatch[1].toLowerCase();
      const scope = ccMatch[2]?.toLowerCase() || "";
      const subject = cleanUnicode(ccMatch[3]);
      if (subject) {
        commits.push({
          hash,
          shortHash: hash.slice(0, 7),
          type,
          scope,
          subject,
        });
      }
    } else {
      const cleaned = cleanUnicode(rawMsg);
      if (cleaned) {
        commits.push({
          hash,
          shortHash: hash.slice(0, 7),
          type: "chore",
          scope: "",
          subject: cleaned,
        });
      }
    }
  }
  return commits;
}

export function generateReleaseNotes(): string {
  const version = getVersion();
  const repoUrl = getRepoUrl();
  const prevTag = sh("git describe --tags --abbrev=0 2>/dev/null");
  const commits = getCommits(prevTag);

  const date = new Date().toISOString().split("T")[0];
  const compareUrl = prevTag
    ? `${repoUrl}/compare/${prevTag}...v${version}`
    : `${repoUrl}/releases/tag/v${version}`;

  const templateDir = join(process.cwd(), ".github/release-templates");
  const mainTemplate = readFileSync(join(templateDir, "template.md"), "utf-8");
  const sectionTemplate = readFileSync(
    join(templateDir, "section.md"),
    "utf-8",
  );
  const itemTemplate = readFileSync(join(templateDir, "item.md"), "utf-8");

  const renderedSections: string[] = [];

  for (const sec of SECTION_CONFIG) {
    const matchingCommits = commits.filter(sec.match);
    if (matchingCommits.length === 0) continue;

    const items = matchingCommits
      .map((c) => {
        const scopePrefix = c.scope ? `**${c.scope}:** ` : "";
        const commitUrl = `${repoUrl}/commit/${c.hash}`;
        return itemTemplate
          .replace("{{scopePrefix}}", scopePrefix)
          .replace("{{subject}}", c.subject)
          .replace("{{shortHash}}", c.shortHash)
          .replace("{{commitUrl}}", commitUrl)
          .trim();
      })
      .join("\n");

    const renderedSec = sectionTemplate
      .replace("{{sectionTitle}}", sec.title)
      .replace("{{items}}", items)
      .trim();

    renderedSections.push(renderedSec);
  }

  const sectionsContent = renderedSections.join("\n\n");
  return mainTemplate
    .replace("{{version}}", `v${version}`)
    .replace("{{compareUrl}}", compareUrl)
    .replace("{{date}}", date)
    .replace("{{sections}}", sectionsContent)
    .trim();
}

function main() {
  const isStdout = process.argv.includes("--stdout");
  const releaseNotes = generateReleaseNotes();

  if (isStdout) {
    process.stdout.write(`${releaseNotes}\n`);
    return;
  }

  const changelogPath = join(process.cwd(), "CHANGELOG.md");
  let existingContent = "";
  if (existsSync(changelogPath)) {
    existingContent = readFileSync(changelogPath, "utf-8");
  }

  const header =
    "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n";

  let newChangelog: string;
  if (!existingContent?.startsWith("# Changelog")) {
    newChangelog = `${header}${releaseNotes}\n`;
  } else {
    // Insert after header
    const afterHeader = existingContent.slice(header.length).trim();
    newChangelog = `${header}${releaseNotes}\n\n${afterHeader}`;
  }

  writeFileSync(changelogPath, `${newChangelog.trim()}\n`);
  console.log(`Updated ${changelogPath} for version ${getVersion()}`);
}

if (import.meta.main) {
  main();
}
