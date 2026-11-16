#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "python-telegram-bot>=21.0",
# ]
# ///
"""Send a message or file to a Telegram chat via the Bot API."""

from __future__ import annotations

import argparse
import asyncio
import json
import os
import sys
from pathlib import Path

from telegram import Bot
from telegram.error import NetworkError, RetryAfter, TelegramError, TimedOut
from telegram.request import HTTPXRequest

SKILL_DIR = Path(__file__).resolve().parent.parent
SETTINGS_FILE = SKILL_DIR / "settings.json"
VALID_PARSE_MODES = ("HTML", "Markdown", "MarkdownV2")


def load_settings() -> dict[str, str]:
    if not SETTINGS_FILE.is_file():
        return {}
    try:
        data = json.loads(SETTINGS_FILE.read_text(encoding="utf-8"))
        return data if isinstance(data, dict) else {}
    except (OSError, json.JSONDecodeError):
        return {}


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="send.py",
        description="Send a message or file to a Telegram chat via the Bot API.",
    )
    parser.add_argument("-m", "--message", default="", help="Message text to send (max 4096 chars)")
    parser.add_argument("-f", "--file", dest="file_path", default="", help="Send a file as a document")
    parser.add_argument("-c", "--caption", default="", help="Caption for --file (max 1024 chars)")
    parser.add_argument(
        "-p",
        "--parse-mode",
        choices=VALID_PARSE_MODES,
        default=None,
        help="Parse mode: HTML, Markdown, or MarkdownV2",
    )
    parser.add_argument("-s", "--silent", action="store_true", help="Send silently without notification sound")
    parser.add_argument("-r", "--reply-to", type=int, default=None, help="Reply to a specific message ID")
    parser.add_argument("-t", "--chat-id", default="", help="Override TELEGRAM_CHAT_ID")
    parser.add_argument("positional", nargs="*", help="Message text (alternative to -m)")
    return parser.parse_args(argv)


async def send_with_retry(args: argparse.Namespace, token: str, chat_id: str, message: str) -> dict:
    max_retries = int(os.environ.get("TELEGRAM_MAX_RETRIES", "3"))
    request = HTTPXRequest(
        connect_timeout=15.0,
        read_timeout=30.0,
        write_timeout=120.0,
        pool_timeout=15.0,
    )
    delay = 2.0

    async with Bot(token=token, request=request) as bot:
        for attempt in range(1, max_retries + 1):
            try:
                if args.file_path:
                    caption = args.caption or message or None
                    file_path = Path(args.file_path)
                    with file_path.open("rb") as f:
                        msg = await bot.send_document(
                            chat_id=chat_id,
                            document=f,
                            filename=file_path.name,
                            caption=caption,
                            parse_mode=args.parse_mode,
                            disable_notification=args.silent,
                            reply_to_message_id=args.reply_to,
                        )
                else:
                    msg = await bot.send_message(
                        chat_id=chat_id,
                        text=message,
                        parse_mode=args.parse_mode,
                        disable_notification=args.silent,
                        reply_to_message_id=args.reply_to,
                    )
                return {"ok": True, "result": msg.to_dict()}

            except RetryAfter as exc:
                wait = float(exc.retry_after) + 0.5
                print(
                    f"Warning: Telegram rate limited (429), retrying in {wait:.1f}s "
                    f"(attempt {attempt}/{max_retries}).",
                    file=sys.stderr,
                )
                if attempt >= max_retries:
                    raise
                await asyncio.sleep(wait)

            except (TimedOut, NetworkError) as exc:
                print(
                    f"Warning: Telegram request failed ({type(exc).__name__}: {exc}) "
                    f"(attempt {attempt}/{max_retries}).",
                    file=sys.stderr,
                )
                if attempt >= max_retries:
                    raise
                await asyncio.sleep(delay)
                delay *= 2

    raise RuntimeError("Unreachable")


def main() -> int:
    args = parse_args(sys.argv[1:])
    settings = load_settings()

    token = os.environ.get("TELEGRAM_BOT_TOKEN") or str(settings.get("bot_token") or "")
    if not token:
        print(f"Error: bot_token is not set. Set TELEGRAM_BOT_TOKEN or configure {SETTINGS_FILE}.", file=sys.stderr)
        return 1

    chat_id = args.chat_id or os.environ.get("TELEGRAM_CHAT_ID") or str(settings.get("chat_id") or "")
    if not chat_id:
        print(
            f"Error: chat_id is not set. Set TELEGRAM_CHAT_ID, pass --chat-id, or configure {SETTINGS_FILE}.",
            file=sys.stderr,
        )
        return 1

    message = args.message or " ".join(args.positional)
    if not message and not args.file_path and not sys.stdin.isatty():
        message = sys.stdin.read()

    if not message and not args.file_path:
        print("Error: provide a message or --file.", file=sys.stderr)
        return 2

    if args.file_path:
        file_path = Path(args.file_path)
        if not file_path.is_file():
            print(f"Error: file not found: {file_path}", file=sys.stderr)
            return 1
        caption = args.caption or message
        if len(caption) > 1024:
            print(f"Error: caption exceeds Telegram 1024-character limit ({len(caption)} chars).", file=sys.stderr)
            return 2
    elif len(message) > 4096:
        print(
            f"Error: message exceeds Telegram 4096-character limit ({len(message)} chars). "
            "Split it or send as --file.",
            file=sys.stderr,
        )
        return 2

    try:
        payload = asyncio.run(send_with_retry(args, token, chat_id, message))
    except TelegramError as exc:
        print(f"Error: Telegram API error: {exc}", file=sys.stderr)
        return 1

    print(json.dumps(payload, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
