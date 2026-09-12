# ลองเทรด v0.1 runbook

## Start

```bash
make build-web
go run ./web/backend
```

Open the printed localhost URL, choose **พอร์ตทดลอง**, and verify the amber
“ไม่มีเงินจริง” badge. A clean workspace automatically creates `paper-main` with
USD 10,000. Use **ลองเทรด** to preview a current simulated quote and explicitly
confirm it.

## Recovery

- `QUOTE_STALE`: wait for/refeed a fresh quote; do not confirm an old proposal.
- `EXPIRED`: prepare a new quote; proposals intentionally expire after 30 seconds.
- Local paper reset: POST `{}` to `/api/paper/account/reset` after confirming that
  history deletion is intended.
- Back up the workspace before moving machines. Paper data is SQLite-backed.

## XM Demo read-only mode

Follow `mt5_bridge/README.md`. The bridge is optional and localhost-only. Stop it
immediately if the terminal is not visibly connected to a Demo account.
