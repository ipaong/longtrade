# LongTrade MVP Baseline

This document records the starting point for the experimental "ลองเทรด" fork.
KhunQuant remains credited under its existing MIT license. Internal package names
stay unchanged during the MVP.

## Product direction

The MVP is API-first and web-first. The browser is the main user interface, and
all trading behavior is owned by deterministic backend services.

```text
ลองเทรด Web UI
        |
        | REST / WebSocket
        v
KhunQuant/ลองเทรด Go Backend
        |
        +-- Persistent paper trading
        +-- Risk and confirmation checks
        +-- MT5 provider
                |
                | REST
                v
        MT5 Bridge on Windows
                |
                v
        MetaTrader 5 -> XM Demo
```

MCP, ChatGPT plugins, real-money trading, strategy optimization, autonomous
strategies, and Bot League are not part of the MVP. Existing upstream MCP code
is left untouched to keep upstream changes easy to merge.

## Safety baseline

- New configurations start in paper-trading mode.
- Leverage is disabled by default.
- Live create, cancel, emergency-stop, transfer, and futures mutation tools
  remain disabled by default.
- Enabling a tool in the UI must never bypass backend validation, risk limits,
  confirmation, or account permissions.
- Real-money trading must not be added until paper trading and XM Demo behavior
  are stable and separately reviewed.

## Architecture map

```yaml
Frontend: web/frontend (React, TypeScript, Vite, TanStack Router and Query)
Backend: web/backend (launcher REST API and static UI) plus cmd/khunquant (gateway and agent runtime)
Trading providers: pkg/providers/broker interfaces and registry; adapters in pkg/exchanges
AI/Agent: pkg/agent with LLM adapters in pkg/providers and web chat in pkg/channels/pico
Tools: pkg/tools; the agent selects tools while deterministic handlers own trading behavior
Storage: config.json, .security.yml, workspace JSONL sessions, SQLite stores, snapshots and state files
Config: pkg/config with launcher-facing config endpoints in web/backend/api
Paper trading: pkg/tools/paper_trade.go; currently stateless and scheduled for replacement by a persistent provider
Best extension point for MT5: pkg/exchanges/mt5 through pkg/providers/broker; connect it to a local Windows bridge
```

## Minimum local commands

Prerequisites are Go 1.26.7 (from `go.mod`), a supported Node.js release (from
`web/frontend/package.json`), and pnpm.

```bash
cd web/frontend
pnpm install
cd ..
make dev
```

The web UI is served at `http://localhost:5173`; the launcher backend listens on
`http://localhost:18800`. To run them in separate terminals:

```bash
cd web
make dev-backend
```

```bash
cd web
make dev-frontend
```

Production-style local build:

```bash
make build-launcher
./build/khunquant-launcher
```

## Verified baseline

- Repository baseline: `aa71f2d` (`main`).
- Frontend dependency installation, lint, and production build succeed.
- The upstream release launcher serves its embedded UI and backend locally.
- A full backend source build exceeds the available memory in the current
  3.7 GiB development environment while compiling the CCXT Go package. Backend
  development therefore needs a higher-memory build runner or CI until that
  dependency constraint is addressed without a broad refactor.

## Next implementation slice

Replace the stateless paper-trade response with a persistent paper provider that
stores account balance, equity, positions, orders, and trade history. Expose that
state through backend REST endpoints before simplifying or rebranding the UI.
