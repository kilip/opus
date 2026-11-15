---
name: telegram
description: >-
  Send messages or files to a user via the Telegram Bot API.
  Use this skill when the user asks to send a Telegram notification,
  alert, status update, summary, or file/report to their Telegram chat.
---

# Telegram Sender

Send text messages or document attachments to the user's Telegram chat using [send.sh](./scripts/send.sh).

## Prerequisites

The script requires two environment variables (do **not** print or echo their values):

- `TELEGRAM_BOT_TOKEN` — Bot token from `@BotFather`
- `TELEGRAM_CHAT_ID` — Target user/group chat ID (can also be overridden via `--chat-id`)

If either variable is missing, stop and ask the user to export them in their shell environment before retrying:

```bash
export TELEGRAM_BOT_TOKEN="<bot-token>"
export TELEGRAM_CHAT_ID="<chat-id>"
```

To discover a `chat_id` after messaging the bot once, the user can run:

```bash
curl -s "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getUpdates"
```

## Usage

Resolve the path to `scripts/send.sh` relative to this skill directory and run it via `bash`:

### 1. Send a plain text message

```bash
bash <skill-dir>/scripts/send.sh --message "Build finished: all tests passed."
```

Or pipe multiline text via stdin:

```bash
cat <<'EOF' | bash <skill-dir>/scripts/send.sh
Task completed:
- Lint: PASS
- Tests: PASS
EOF
```

### 2. Send formatted text (`HTML` or `MarkdownV2`)

Prefer `HTML` for formatting (fewer escaping pitfalls than `MarkdownV2`):

```bash
bash <skill-dir>/scripts/send.sh \
  --parse-mode HTML \
  --message "<b>Deploy complete</b>: <code>v1.2.3</code>"
```

### 3. Send a file or report (`sendDocument`)

Use `--file` for logs, diffs, or reports longer than Telegram's 4096-character message limit:

```bash
bash <skill-dir>/scripts/send.sh \
  --file ./report.md \
  --caption "Verification report"
```

### Options

- `-m, --message <text>` — Text message (max 4096 chars)
- `-f, --file <path>` — File to send as document
- `-c, --caption <text>` — Caption for `--file` (max 1024 chars)
- `-p, --parse-mode <mode>` — `HTML`, `Markdown`, or `MarkdownV2`
- `-s, --silent` — Deliver without notification sound
- `-r, --reply-to <id>` — Reply to a specific message ID
- `-t, --chat-id <id>` — Override `TELEGRAM_CHAT_ID`

## Rules & Verification

1. **Never leak secrets**: Never include tokens, API keys, `.env` contents, or credentials in Telegram messages, and never echo `$TELEGRAM_BOT_TOKEN` in terminal output.
2. **Verify delivery**: Confirm `send.sh` exits `0` and returns JSON containing `"ok":true`.
