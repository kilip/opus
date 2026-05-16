# WhatsApp Multi-User Integration Design

**Topic:** Multi-user WhatsApp integration via Dash
**Date:** 2026-05-16
**Status:** Draft

## 1. Overview
The goal is to allow every user on the Opus platform to connect their own WhatsApp account. The system will support multiple concurrent WhatsApp sessions using `go.mau.fi/whatsmeow`. The frontend integration will be placed in the settings area of the dashboard.

## 2. Architecture & State Management
- **Library:** `go.mau.fi/whatsmeow` will be used directly in the Go backend.
- **Persistence (Whatsmeow):** We will use `whatsmeow`'s built-in `sqlstore` utilizing the existing primary database connection (SQLite/Postgres). This isolates internal WhatsApp cryptography and session states from our primary domain models.
- **Persistence (Domain):** A new EntGo schema named `WaSession` will be created to track connection statuses per user.
- **Session Manager:** A new service (`WhatsAppService`) will manage active `whatsmeow.Client` instances in-memory using a `sync.Map`, keyed by the user's `ID` (string).

## 3. Database Schema (EntGo)
**Entity: `WaSession`**
- `id` (string): Unique identifier for the record.
- `jid` (string): The connected WhatsApp ID / phone number (populated post-connection).
- `status` (string): Connection status (`UNAUTHENTICATED`, `CONNECTED`, `DISCONNECTED`).
- **Edges:**
  - `user`: One-to-One relationship with the `User` entity.

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

## 5. Realtime Communication (SSE)
We will leverage the existing `/stream` endpoint to dispatch WhatsApp-specific events to the client.
- Event `wa_qr_update`: Contains the string payload for the QR code.
- Event `wa_connected`: Fired when the device successfully pairs and authenticates.
- Event `wa_disconnected`: Fired when the device disconnects or logs out.

## 6. Frontend (Next.js)
- **Path:** `app/(dashboard)/settings/whatsapp/page.tsx`
- **UI States:**
  - **Loading:** Fetching initial status.
  - **Connected:** Shows the connected JID and a "Disconnect" button.
  - **Unauthenticated/Disconnected:** Shows a "Connect" button. Clicking it calls `/whatsapp/connect` and listens to the SSE stream.
  - **Pairing:** Renders the QR code received from the `wa_qr_update` SSE event using a library like `qrcode.react`.

## 7. Initialization & Boot
When the backend server starts, the `WhatsAppService` will query the `WaSession` table for all users with a `CONNECTED` status and proactively initialize and connect their `whatsmeow` clients to ensure they remain online across server restarts.
