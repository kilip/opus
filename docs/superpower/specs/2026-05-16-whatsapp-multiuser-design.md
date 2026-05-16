# WhatsApp Multi-User Integration Design

**Topic:** Multi-user WhatsApp integration via Dash
**Date:** 2026-05-16
**Status:** Draft

## 1. Overview
The goal is to allow every user on the Opus platform to connect their own WhatsApp account. The system will support multiple concurrent WhatsApp sessions using `go.mau.fi/whatsmeow`. The frontend integration will be placed in the settings area of the dashboard, alongside a dedicated chat interface for messaging.

## 2. Architecture & State Management
- **Library:** `go.mau.fi/whatsmeow` will be used directly in the Go backend.
- **Persistence (Whatsmeow):** We will use `whatsmeow`'s built-in `sqlstore` utilizing the existing primary database connection (SQLite/Postgres). This isolates internal WhatsApp cryptography and session states from our primary domain models.
- **Persistence (Domain):** A new EntGo schema named `WaSession` will be created to track connection statuses per user. Additional schemas will be used for full data synchronization.
- **Session Manager:** A new service (`WhatsAppService`) will manage active `whatsmeow.Client` instances in-memory using a `sync.Map`, keyed by the user's `ID` (string).

## 3. Database Schema (EntGo)
**Entity: `WaSession`**
- `id` (string): Unique identifier for the record.
- `jid` (string): The connected WhatsApp ID / phone number (populated post-connection).
- `status` (string): Connection status (`UNAUTHENTICATED`, `CONNECTED`, `DISCONNECTED`).
- **Edges:**
  - `user`: One-to-One relationship with the `User` entity.

**Entities for Full Sync:**
- `WaContact`: Stores user contacts (JID, name, pushname). Edge to `WaSession`.
- `WaChat`: Stores conversation/group list (JID, chat name, unread count). Edge to `WaSession`.
- `WaMessage`: Stores message details (Message ID, sender, text content, timestamp, isFromMe flag). Edges to `WaChat` and `WaSession`.

## 4. API Endpoints
All endpoints are protected by the existing JWT authentication middleware.
Base path: `/whatsapp` (no `/api/v1` prefix).

- `GET /whatsapp/status`
  - Returns the current connection status and JID (if connected) for the authenticated user.
- `POST /whatsapp/connect`
  - Triggers the creation of a new Whatsmeow client (if not exists) and initiates the pairing process.
  - Returns a success response. The actual QR code string is delivered via SSE.
- `POST /whatsapp/disconnect`
  - Logs out the current WhatsApp session, clears the `sqlstore` for that device, and updates the `WaSession` status to `DISCONNECTED`.
- `POST /whatsapp/send`
  - Payload: `{ "target_jid": "62812xxx@s.whatsapp.net", "message": "hello" }`
  - Sends a text message to the target JID and saves the record to `WaMessage` with `isFromMe = true`.

## 5. Realtime Communication (SSE)
We will leverage the existing `/stream` endpoint to dispatch WhatsApp-specific events to the client.
- Event `wa_qr_update`: Contains the string payload for the QR code.
- Event `wa_connected`: Fired when the device successfully pairs and authenticates.
- Event `wa_disconnected`: Fired when the device disconnects or logs out.
- Event `wa_new_message`: Fired when a new message is received or sent, useful for real-time UI updates.

## 6. Frontend (Next.js)
- **Settings Path:** `app/(dashboard)/settings/whatsapp/page.tsx`
  - **Loading:** Fetching initial status.
  - **Connected:** Shows the connected JID and a "Disconnect" button.
  - **Unauthenticated/Disconnected:** Shows a "Connect" button. Clicking it calls `/whatsapp/connect` and listens to the SSE stream.
  - **Pairing:** Renders the QR code received from the `wa_qr_update` SSE event using a library like `qrcode.react`.
- **Chat UI Path:** `app/(dashboard)/whatsapp/chat/page.tsx`
  - **Chat List (Left):** Displays conversations retrieved from `WaChat`.
  - **Message Area (Right):** Displays message history (`WaMessage`). Includes an input box and "Send" button that calls `POST /whatsapp/send`. Updates in real-time via `wa_new_message` SSE event.

## 7. Initialization & Boot
When the backend server starts, the `WhatsAppService` will query the `WaSession` table for all users with a `CONNECTED` status and proactively initialize and connect their `whatsmeow` clients to ensure they remain online across server restarts.

## 8. Data Synchronization (Full Sync)
Opus will replicate the user's WhatsApp data to provide full context.
- **History Sync (Initial Connection):** Upon successful pairing, WhatsApp sends `events.HistorySync`. This parsing is executed in a separate goroutine to prevent blocking. Data is upserted into `WaContact`, `WaChat`, and `WaMessage`.
- **Active Sync (Real-time):** The `WhatsAppService` listens to `events.Message`. New incoming/outgoing messages are immediately inserted into `WaMessage`.
- **Media Handling:** Media files (images/videos/documents) are not downloaded automatically to save storage. Only metadata (e.g., caption text or message type) is stored initially.

## 9. Message Sending API & UI
Opus allows sending standard text messages through the connected session.
- Uses `client.SendMessage()` via the new `POST /whatsapp/send` API.
- The UI provides a familiar chat interface utilizing Next.js and Tailwind/Shadcn, reflecting real-time state via SSE.
