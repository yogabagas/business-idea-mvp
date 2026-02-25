# YouTube Live Spam Filter

Web app that shows **YouTube Live chat** in real time with **spam filtering**: blocklist, repetition detection, link detection, and custom regex. Built with a Go backend (Gin, YouTube Data API v3, Google OAuth2) and a React (Vite) frontend.

## Features

- **Sign in with Google** (YouTube read-only scope) to access live chat via the official API.
- **Enter any YouTube Live URL or video ID** to load that stream’s chat.
- **Spam filters**
  - **Blocklist**: case-insensitive words/phrases (one per line).
  - **Repetition**: same message N times from the same author (configurable threshold).
  - **Links**: flag messages containing URLs.
  - **Custom regex**: optional pattern to flag messages.
- **Hide or show spam**: either hide filtered messages or show them as flagged.

## Prerequisites

- **Go 1.21+**
- **Node.js 18+** (for the web frontend)
- **Google Cloud project** with:
  - **YouTube Data API v3** enabled
  - **OAuth 2.0 Client ID** (Web application) with:
    - Authorized JavaScript origins: `http://localhost:5173` (and your production origin if needed)
    - Authorized redirect URIs: `http://localhost:8080/api/auth/callback` (and production if needed)

## Setup

1. **Clone and backend env**

   ```bash
   cd business-idea-mvp
   cp .env.example .env
   # Edit .env and set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET from your OAuth client.
   ```

2. **Install Go dependencies**

   ```bash
   go mod tidy
   ```

3. **Run the backend**

   ```bash
   go run ./cmd/server
   ```

   Server listens on `http://localhost:8080` by default (override with `PORT`).

4. **Install and run the frontend**

   ```bash
   cd web
   npm install
   npm run dev
   ```

   Frontend runs at `http://localhost:5173` and proxies `/api` to the backend.

5. Open **http://localhost:5173**, click **Sign in with Google**, then enter a **YouTube Live** URL or video ID and adjust filters as needed.

## How to test

### 1. Google Cloud setup (one-time)

1. Go to [Google Cloud Console](https://console.cloud.google.com/) and create or select a project.
2. **Enable the API**: APIs & Services → Library → search **YouTube Data API v3** → Enable.
3. **Create OAuth credentials**: APIs & Services → Credentials → Create Credentials → OAuth client ID.
   - Application type: **Web application**.
   - Name: e.g. `YouTube Live Spam Filter`.
   - **Authorized JavaScript origins**: `http://localhost:5173`
   - **Authorized redirect URIs**: `http://localhost:8080/api/auth/callback`
   - Create → copy **Client ID** and **Client secret** into your `.env` as `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`.

### 2. Run the app

**Terminal 1 – backend:**

```bash
cd /Users/yogabagas/Project/Go/src/github/yogabagas/business-idea-mvp
go mod tidy
go run ./cmd/server
```

**Terminal 2 – frontend:**

```bash
cd /Users/yogabagas/Project/Go/src/github/yogabagas/business-idea-mvp/web
npm install
npm run dev
```

### 3. Test in the browser

1. Open **http://localhost:5173**.
2. Click **Sign in with Google** and complete the Google login (allow YouTube read-only access if prompted).
3. You should land back on the app, logged in.
4. **Load a live chat**: you need a **currently live** YouTube stream. Options:
   - Go to [YouTube Live](https://www.youtube.com/live) and pick any stream that is **live now**.
   - Copy the URL (e.g. `https://www.youtube.com/watch?v=VIDEO_ID`) or just the video ID.
5. Paste the URL or video ID into **YouTube Live URL or Video ID** and click **Load chat**.
6. Messages should appear and update automatically. Try the filters:
   - **Flag messages containing links** – turn on; messages with URLs get a “Spam (link)” badge.
   - **Blocklist** – add words (e.g. `spam`, `promo`), one per line; messages containing them are flagged.
   - **Repetition threshold** – set to `3`; if the same user sends the same message 3+ times, it’s flagged.
   - **Custom regex** – e.g. `(?:free|win|click)` to flag those words.
   - **Hide spam** – turn on to hide flagged messages instead of showing them with a badge.

### 4. If you don’t have a live stream

- The app only works with **active** YouTube Live streams. Regular (non-live) videos return “video is not a live stream or has no active chat”.
- To test without going live yourself: use any public stream that is live at [YouTube Live](https://www.youtube.com/live) (e.g. news or 24/7 streams).

### 5. Quick API check (optional)

With the backend running and after logging in once in the browser (so you have a session cookie):

```bash
# Replace VIDEO_ID with a live stream’s video ID
curl -v -b "session_id=YOUR_SESSION_COOKIE" "http://localhost:8080/api/live-chat/id?videoId=VIDEO_ID"
```

You should get JSON with `liveChatId` and `videoId`.

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GOOGLE_CLIENT_ID` | Yes | OAuth 2.0 Client ID (Web application). |
| `GOOGLE_CLIENT_SECRET` | Yes | OAuth 2.0 Client secret. |
| `PORT` | No | Server port (default `8080`). |
| `FRONTEND_ORIGIN` | No | Allowed CORS origin (default `http://localhost:5173`). |

## Project layout

- `cmd/server/` – Go server entrypoint.
- `internal/api/` – HTTP handlers and routes.
- `internal/auth/` – Google OAuth2 and YouTube service.
- `internal/config/` – Config from env.
- `internal/filters/` – Spam filter engine (blocklist, repetition, links, regex).
- `internal/store/` – In-memory session store.
- `internal/youtube/` – YouTube API client (live chat ID, messages).
- `web/` – React + Vite frontend.

## API (backend)

- `GET /api/auth/login` – Redirects to Google OAuth.
- `GET /api/auth/callback` – OAuth callback; sets session cookie and redirects to frontend.
- `GET /api/auth/me` – Returns `{ "loggedIn": true|false }`.
- `POST /api/auth/logout` – Clears session.
- `GET /api/live-chat/id?videoId=...` – Returns `{ "liveChatId", "videoId" }` (requires auth).
- `POST /api/live-chat/messages` – Body: `{ "liveChatId", "pageToken?", "filterConfig?" }`. Returns messages with spam flags and `nextPageToken`, `pollingIntervalMillis` (requires auth).

## License

MIT.
