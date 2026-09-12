# LongTrade v0.1 safety audit

## Enforced invariants

| Path | Capability | Enforcement |
|---|---|---|
| Paper REST API | Prepare/open/close/protection/reset | All mutations call `paper.Engine` or atomic `paper.Store` reset; proposal confirmation and idempotency are mandatory. |
| Paper persistence | Local SQLite only | Database is private to the workspace and starts at USD 10,000. |
| Web UI | Paper operations only | UI calls `/api/paper/*`; it contains no exchange credentials or broker-order endpoint. |
| MT5 bridge | Read only | Only HTTP GET health/account/quote/positions exist. No Python or Go order method exists. |
| XM | Demo inspection only | Operator must verify the terminal account is Demo before starting the localhost bridge. |

## Explicit exclusions

- No leverage, margin order, fund transfer, autonomous strategy, or real-money route.
- No MT5 `order_send`, trade request, login, or credential endpoint.
- The in-process default quote is visibly a simulated quote, not market data.
- A state-changing web action requires an explicit preview/confirmation step.

## Release gate

Run `make longtrade-check`. On Windows, additionally complete the XM Demo field
check in `docs/longtrade/XM_DEMO_TEST.md`. Never connect the bridge to a live
terminal.
