import type { UserConfig } from "@commitlint/types";

const EMOJI_MAP: Record<string, string> = {
  feat: "✨",
  fix: "🐛",
  refactor: "♻️",
  perf: "⚡",
  test: "✅",
  docs: "📝",
  build: "📦",
  ci: "👷",
  chore: "🔧",
  revert: "⏪",
};

const ALLOWED_TYPES = Object.keys(EMOJI_MAP);

const config: UserConfig = {
  parserPreset: {
    parserOpts: {
      headerPattern: /^([a-z]+)(!?): (.+)$/,
      headerCorrespondence: ["type", "breaking", "subject"],
    },
  },
  plugins: [
    {
      rules: {
        "opus-commit-format": ({ header, raw, body, footer }) => {
          if (!header) {
            return [false, "Commit message cannot be empty"];
          }

          // 1. Max length
          if (header.length > 72) {
            return [
              false,
              `Header must not be longer than 72 characters (currently ${header.length})`,
            ];
          }

          // 2. Reject scopes (e.g. feat(auth): ...)
          if (/^[a-z]+(?:\([^)]*\))!?:/i.test(header)) {
            return [
              false,
              'Do not use scopes in commit message (use "type: <emoji> subject", not "type(scope): ...")',
            ];
          }

          // 3. Match type and subject
          const match = header.match(/^([a-z]+)(!?): (.+)$/);
          if (!match) {
            return [
              false,
              'Commit message must follow format: "type: <emoji> subject" (e.g. "feat: ✨ add feature")',
            ];
          }

          const [, type, breaking, rest] = match;

          // 4. Validate type
          if (!ALLOWED_TYPES.includes(type)) {
            return [
              false,
              `Type "${type}" is not allowed. Must be one of: ${ALLOWED_TYPES.join(", ")}`,
            ];
          }

          // 5. Validate emoji
          const expectedEmoji = EMOJI_MAP[type];
          if (!rest.startsWith(`${expectedEmoji} `)) {
            return [
              false,
              `Type "${type}" must be followed by emoji "${expectedEmoji} ". Example: "${type}: ${expectedEmoji} ..."`,
            ];
          }

          // 6. Validate subject text
          const subject = rest.slice(expectedEmoji.length + 1).trim();
          if (!subject) {
            return [false, "Subject cannot be empty"];
          }

          if (subject.endsWith(".")) {
            return [false, "Subject must not have a trailing period"];
          }

          const firstChar = subject.charAt(0);
          if (firstChar !== firstChar.toLowerCase()) {
            return [false, "Subject must start with a lowercase letter"];
          }

          // 7. Validate breaking change footer
          if (breaking === "!") {
            const fullMessage = raw || `${header}\n\n${body || ""}\n\n${footer || ""}`;
            if (!fullMessage.includes("BREAKING CHANGE:")) {
              return [
                false,
                'Breaking changes ("type!:") must include a "BREAKING CHANGE:" footer',
              ];
            }
          }

          // 8. Body line length wrap
          if (body) {
            const lines = body.split("\n");
            for (const line of lines) {
              if (line.length > 72) {
                return [
                  false,
                  `Body lines must wrap at ~72 chars (found line with ${line.length} chars)`,
                ];
              }
            }
          }

          return [true];
        },
      },
    },
  ],
  rules: {
    "opus-commit-format": [2, "always"],
  },
};

export default config;
