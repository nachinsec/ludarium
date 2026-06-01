# Ludarium

A self-hosted game library tracker — a Backloggd / HowLongToBeat alternative you
run yourself. Import your Steam library, track what you're playing, beaten and
backlogged, rate and annotate games, see your stats, and get AI recommendations
from a model **you** choose.

Privacy-first: no SaaS backend, no telemetry. Your data lives in a single SQLite
file, and the whole app ships as one ~21 MB container.

## Features

- **Your library, your way** — import from Steam in one click, or add any game
  by searching IGDB. Track status (playing · ongoing · cleared · backlog ·
  dropped · wishlist), rating, notes and playtime.
- **Multi-user** — email/password or Steam login, linkable. Every user only ever
  sees their own library (isolation enforced in every query).
- **Stats** — total hours, status breakdown, average rating, most-played games
  and top genres, drawn as dependency-free pixel bar charts.
- **Oracle** — a "this-or-that" taste quiz that feeds your picks to an LLM and
  recommends games you *don't* own yet, each verified against IGDB (real cover,
  no hallucinations).
- **Discover** — search the whole game database, browse trending and upcoming
  releases, add to your library in one click.
- **Bring your own AI** — works with any OpenAI-compatible endpoint (OpenAI,
  Ollama, Groq, OpenRouter…). Each user can set their own provider/model/key in
  Settings; keys are encrypted at rest (AES-GCM).
- **Pixel design system** — a hand-rolled retro/CRT UI, no Tailwind, no component
  library.

## Stack

- **Backend:** Go · chi · SQLite (`modernc.org/sqlite`, pure Go) · goose migrations
- **Frontend:** React · Vite · TypeScript · TanStack Query · CSS Modules
- **Auth:** bcrypt accounts and/or Steam OpenID · opaque session cookies
- **External:** Steam Web API · IGDB (Twitch) · any OpenAI-compatible LLM
- **Deploy:** a single Go binary with the frontend embedded (`go:embed`)

## Quick start (Docker)

```bash
cp -n .env.example .env       # set SESSION_SECRET (required) + any keys
docker compose up --build     # or: podman compose up --build
```

Open <http://localhost:3000>. SQLite persists in a named volume; the first user
to register becomes the instance admin.

> Using a local Ollama? It runs on the host, not in the container — set
> `AI_BASE_URL=http://host.docker.internal:11434/v1` in `.env`.

## Development

Requires Go ≥ 1.26 and Node ≥ 22 (pnpm).

```bash
make dev-api   # backend on :3000
make dev-web   # frontend on :5173 (proxies /api → :3000)
```

Open <http://localhost:5173>. To build the single production binary:

```bash
make build && ./bin/ludarium
```

## Configuration

All via environment variables (a `.env` in the working directory is loaded
automatically).

| Variable | Default | Purpose |
|---|---|---|
| `BASE_URL` | `http://localhost:3000` | Public URL; Steam OpenID realm. `https://…` enables Secure cookies. |
| `SESSION_SECRET` | random in dev | **Required in production.** Also derives the encryption key for stored API keys. |
| `DB_PATH` | `./data/ludarium.db` | SQLite file location. |
| `STEAM_API_KEY` | — | Enables Steam library import ([get one](https://steamcommunity.com/dev/apikey)). |
| `IGDB_CLIENT_ID` / `IGDB_CLIENT_SECRET` | — | Metadata, search, Discover ([Twitch app](https://dev.twitch.tv/console/apps)). |
| `AI_BASE_URL` | `https://api.openai.com/v1` | Default AI endpoint (OpenAI-compatible). |
| `AI_API_KEY` | — | Key for the default AI (any value for local Ollama). |
| `AI_MODEL` | `gpt-4o-mini` | Default model. |

Steam, IGDB and a default AI are all optional — the app runs without them, you
just won't have that feature until configured. Each user can also override the
AI from **Settings → AI provider**.

## License

MIT.
