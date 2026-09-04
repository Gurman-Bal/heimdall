# Heimdall

Heimdall is a self-hosted monitoring tool built for TrueNAS. It tails log files
(TrueNAS system logs, Minecraft server logs, whatever you point it at), matches
lines against configurable rules, and shows you what's going on through a web
dashboard. It also runs a local LLM (via Ollama) periodically to summarize
recent activity and flag anything worth looking at.

Backend is Go, no external dependencies beyond SQLite (embedded). Frontend is
plain HTML/CSS/JS for efficiency.

## Features

- Tails log files incrementally, tracks offsets so restarts don't replay history
- Pluggable log sources (TrueNAS, Minecraft, easy to add more)
- Regex-based rules, editable live from the UI, no rebuild needed
- Real-time event stream (SSE) + persisted event history
- LLM-generated periodic reports (local Ollama, no external API calls)
- Session-based login with configurable idle timeout
- Ops tab: restart/stop/start allowlisted Docker containers, change password,
  see what Heimdall itself is doing right now
- Activity log: Heimdall's own operational history, separate from monitored
  events, queryable by time window

## Requirements

- Docker + Docker Compose
- [Ollama](https://ollama.com) running somewhere reachable (bundled as a
  service in the provided `docker-compose.yml`)

## Setup

1. Clone the repo:
```bash
   git clone https://github.com/Gurman-Bal/heimdall.git
   cd heimdall
```

2. Create your `.env` (next to `docker-compose.yml`, not committed to git):
````
HEIMDALL_AUTH_USER=admin
HEIMDALL_AUTH_PASS=set-a-real-password-here
````

3. Build and start:
```bash
   docker compose up -d --build
```

4. First run only - pull the model:
```bash
   docker exec heimdall-ollama ollama pull qwen2.5:0.5b
```

5. Open `http://<your-server-ip>:8080`, log in, add your log sources through
   the Sources tab.

## Updating

```bash
git pull
docker compose up -d --build
```

## Local development

Copy `.env.example` to `.env`, then:
```bash
go run ./cmd/heimdall
```
Note: the Ops tab's container controls need `/var/run/docker.sock`, which
only exists when running in Docker on Linux, won't work from a local
`go run` on Windows/Mac. Test that part on the real deployment.

## Config

All config is env vars, see `.env.example` for the full list. Notable ones:

| Var | Default | What it does |
|---|---|---|
| `HEIMDALL_OLLAMA_URL` | `http://localhost:11434` | Where to reach Ollama |
| `HEIMDALL_LLM_MODEL` | `qwen2.5:0.5b` | Model used for reports |
| `HEIMDALL_REPORT_INTERVAL` | `1h` | How often reports auto-generate |
| `HEIMDALL_CONTROLLABLE_CONTAINERS` | `heimdall,heimdall-ollama` | Containers the Ops tab is allowed to touch |
| `HEIMDALL_SESSION_TIMEOUT` | `30m` | Idle timeout before re-login (also editable live from Ops) |

## Project layout
````
cmd/heimdall - entrypoint
internal/core - event bus, rule engine, activity log, status tracker
internal/ingest - generic file-tailing (shared by all source plugins)
internal/plugins - source-specific parsers (truenas, minecraft, ...)
internal/storage - SQLite, goose migrations
internal/auth - password hashing, sessions
internal/services - reporting (LLM), dockerctl
internal/api - HTTP handlers + embedded web UI
````

## Known limitations

- No auto-remediation - reports suggest fixes, nothing runs automatically.
  This was deliberate (see project history/decisions if you're curious why).
- Docker socket is mounted read-only into the container for the Ops tab.
  This still grants real control over the host's Docker daemon - only safe
  because Heimdall's own auth gates it and `dockerctl` only allows a fixed
  container allowlist.
- Small local LLM (qwen2.5:0.5b by default) means reports are decent, not
  brilliant. Bump the model in `.env` if you want better summaries and don't
  mind the extra RAM/CPU.

## Roadmap / not done yet

- Notifications (Discord/email) on critical events
- More source plugins (Docker container logs, Plex, etc.)
- Passive network/firewall log ingestion (design exists, not built)