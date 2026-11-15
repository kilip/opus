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
MAX_RETRIES="${TELEGRAM_MAX_RETRIES:-3}"

post_telegram() {
  local endpoint="$1"
  shift
  local attempt=1
  local delay=2
  local raw_resp http_code body

  while true; do
    if raw_resp="$(curl -s --connect-timeout 10 --max-time 30 -w $'\n%{http_code}' -X POST "$@" "${API_BASE}/${endpoint}")"; then
      http_code="${raw_resp##*$'\n'}"
      body="${raw_resp%$'\n'*}"

      if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]] && [[ "$body" == *'"ok":true'* ]]; then
        echo "$body"
        return 0
      fi

      # Do not retry permanent 4xx client errors (except 429 rate limit)
      if [[ "$http_code" -ge 400 && "$http_code" -lt 500 && "$http_code" -ne 429 ]]; then
        echo "Error: Telegram API returned HTTP ${http_code}: ${body}" >&2
        return 1
      fi

      echo "Warning: Telegram ${endpoint} returned HTTP ${http_code} (attempt ${attempt}/${MAX_RETRIES})." >&2
    else
      echo "Warning: curl request to Telegram ${endpoint} failed (attempt ${attempt}/${MAX_RETRIES})." >&2
    fi

    if [[ "$attempt" -ge "$MAX_RETRIES" ]]; then
      echo "Error: Telegram ${endpoint} failed after ${MAX_RETRIES} attempts." >&2
      return 1
    fi

    sleep "$delay"
    attempt=$((attempt + 1))
    delay=$((delay * 2))
  done
}

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
    -F "chat_id=${CHAT_ID}"
    -F "document=@${FILE_PATH}"
  )
  [[ -n "$CAPTION" ]] && CURL_ARGS+=(--form-string "caption=${CAPTION}")
  [[ -n "$PARSE_MODE" ]] && CURL_ARGS+=(-F "parse_mode=${PARSE_MODE}")
  [[ "$SILENT" == "true" ]] && CURL_ARGS+=(-F "disable_notification=true")
  [[ -n "$REPLY_TO" ]] && CURL_ARGS+=(-F "reply_to_message_id=${REPLY_TO}")

  post_telegram "sendDocument" "${CURL_ARGS[@]}"
else
  if [[ ${#MESSAGE} -gt 4096 ]]; then
    echo "Error: message exceeds Telegram 4096-character limit (${#MESSAGE} chars). Split it or send as --file." >&2
    exit 2
  fi

  CURL_ARGS=(
    --data-urlencode "chat_id=${CHAT_ID}"
    --data-urlencode "text=${MESSAGE}"
  )
  [[ -n "$PARSE_MODE" ]] && CURL_ARGS+=(-d "parse_mode=${PARSE_MODE}")
  [[ "$SILENT" == "true" ]] && CURL_ARGS+=(-d "disable_notification=true")
  [[ -n "$REPLY_TO" ]] && CURL_ARGS+=(-d "reply_to_message_id=${REPLY_TO}")

  post_telegram "sendMessage" "${CURL_ARGS[@]}"
fi
