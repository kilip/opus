#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SETTINGS_FILE="${SKILL_DIR}/settings.json"

usage() {
  cat <<'EOF'
Usage: send.sh [OPTIONS] [MESSAGE]

Send a message or file to a Telegram chat via the Bot API.

Configuration (checked in order):
  1. Environment variables: TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID
  2. Skill config file:     <skill-dir>/settings.json (bot_token, chat_id)

Options:
  -m, --message <text>      Message text to send (max 4096 chars)
  -f, --file <path>         Send a file as a document
  -c, --caption <text>      Caption for --file (max 1024 chars)
  -p, --parse-mode <mode>   Parse mode: HTML, Markdown, or MarkdownV2
  -s, --silent              Send silently without notification sound
  -r, --reply-to <id>       Reply to a specific message ID
  -t, --chat-id <id>        Override TELEGRAM_CHAT_ID
  -h, --help                Show this help message
EOF
}

MESSAGE=""
FILE_PATH=""
CAPTION=""
PARSE_MODE=""
SILENT="false"
REPLY_TO=""
CHAT_ID="${TELEGRAM_CHAT_ID:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -m|--message)
      MESSAGE="${2:-}"
      shift 2
      ;;
    -f|--file)
      FILE_PATH="${2:-}"
      shift 2
      ;;
    -c|--caption)
      CAPTION="${2:-}"
      shift 2
      ;;
    -p|--parse-mode)
      PARSE_MODE="${2:-}"
      shift 2
      ;;
    -s|--silent)
      SILENT="true"
      shift
      ;;
    -r|--reply-to)
      REPLY_TO="${2:-}"
      shift 2
      ;;
    -t|--chat-id)
      CHAT_ID="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      if [[ $# -gt 0 ]]; then
        MESSAGE="$*"
      fi
      break
      ;;
    -*)
      echo "Error: unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      if [[ -z "$MESSAGE" ]]; then
        MESSAGE="$1"
      else
        MESSAGE="$MESSAGE $1"
      fi
      shift
      ;;
  esac
done

if ! command -v curl >/dev/null 2>&1; then
  echo "Error: curl is required but not installed." >&2
  exit 1
fi

read_setting() {
  local key="$1"
  [[ -f "$SETTINGS_FILE" ]] || return 0
  if command -v jq >/dev/null 2>&1; then
    jq -r --arg k "$key" '.[$k] // empty' "$SETTINGS_FILE" 2>/dev/null || true
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); print(d.get(sys.argv[2],"") or "")' "$SETTINGS_FILE" "$key" 2>/dev/null || true
  fi
}

TOKEN="${TELEGRAM_BOT_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(read_setting "bot_token")"
fi
if [[ -z "$TOKEN" ]]; then
  echo "Error: bot_token is not set. Set TELEGRAM_BOT_TOKEN or configure ${SETTINGS_FILE}." >&2
  exit 1
fi

if [[ -z "$CHAT_ID" ]]; then
  CHAT_ID="$(read_setting "chat_id")"
fi
if [[ -z "$CHAT_ID" ]]; then
  echo "Error: chat_id is not set. Set TELEGRAM_CHAT_ID, pass --chat-id, or configure ${SETTINGS_FILE}." >&2
  exit 1
fi

if [[ -n "$PARSE_MODE" && "$PARSE_MODE" != "HTML" && "$PARSE_MODE" != "Markdown" && "$PARSE_MODE" != "MarkdownV2" ]]; then
  echo "Error: invalid --parse-mode '$PARSE_MODE' (expected HTML, Markdown, or MarkdownV2)." >&2
  exit 2
fi

if [[ -z "$MESSAGE" && -z "$FILE_PATH" ]]; then
  if [[ ! -t 0 ]]; then
    MESSAGE="$(cat)"
  fi
fi

if [[ -z "$MESSAGE" && -z "$FILE_PATH" ]]; then
  echo "Error: provide a message or --file." >&2
  usage >&2
  exit 2
fi

API_BASE="https://api.telegram.org/bot${TOKEN}"

if [[ -n "$FILE_PATH" ]]; then
  if [[ ! -f "$FILE_PATH" ]]; then
    echo "Error: file not found: $FILE_PATH" >&2
    exit 1
  fi
  if [[ -z "$CAPTION" && -n "$MESSAGE" ]]; then
    CAPTION="$MESSAGE"
  fi
  if [[ ${#CAPTION} -gt 1024 ]]; then
    echo "Error: caption exceeds Telegram 1024-character limit (${#CAPTION} chars)." >&2
    exit 2
  fi

  CURL_ARGS=(
    -s
    -w $'\n%{http_code}'
    -X POST
    -F "chat_id=${CHAT_ID}"
    -F "document=@${FILE_PATH}"
  )
  [[ -n "$CAPTION" ]] && CURL_ARGS+=(--form-string "caption=${CAPTION}")
  [[ -n "$PARSE_MODE" ]] && CURL_ARGS+=(-F "parse_mode=${PARSE_MODE}")
  [[ "$SILENT" == "true" ]] && CURL_ARGS+=(-F "disable_notification=true")
  [[ -n "$REPLY_TO" ]] && CURL_ARGS+=(-F "reply_to_message_id=${REPLY_TO}")

  RAW_RESP="$(curl "${CURL_ARGS[@]}" "${API_BASE}/sendDocument")" || {
    echo "Error: curl request to Telegram sendDocument failed." >&2
    exit 1
  }
else
  if [[ ${#MESSAGE} -gt 4096 ]]; then
    echo "Error: message exceeds Telegram 4096-character limit (${#MESSAGE} chars). Split it or send as --file." >&2
    exit 2
  fi

  CURL_ARGS=(
    -s
    -w $'\n%{http_code}'
    -X POST
    --data-urlencode "chat_id=${CHAT_ID}"
    --data-urlencode "text=${MESSAGE}"
  )
  [[ -n "$PARSE_MODE" ]] && CURL_ARGS+=(-d "parse_mode=${PARSE_MODE}")
  [[ "$SILENT" == "true" ]] && CURL_ARGS+=(-d "disable_notification=true")
  [[ -n "$REPLY_TO" ]] && CURL_ARGS+=(-d "reply_to_message_id=${REPLY_TO}")

  RAW_RESP="$(curl "${CURL_ARGS[@]}" "${API_BASE}/sendMessage")" || {
    echo "Error: curl request to Telegram sendMessage failed." >&2
    exit 1
  }
fi

HTTP_CODE="${RAW_RESP##*$'\n'}"
BODY="${RAW_RESP%$'\n'*}"

if [[ "$HTTP_CODE" -lt 200 || "$HTTP_CODE" -ge 300 ]] || [[ "$BODY" != *'"ok":true'* ]]; then
  echo "Error: Telegram API returned HTTP ${HTTP_CODE}: ${BODY}" >&2
  exit 1
fi

echo "$BODY"
