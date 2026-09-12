# XM Demo MT5 read-only bridge

This Windows-only helper reads a running MetaTrader 5 **demo** terminal. It binds
to localhost and intentionally implements GET endpoints only; it has no order,
deal, login, or account mutation endpoint.

1. Install Python 3.11 x64 and XM MetaTrader 5, then log in to an XM Demo account.
2. Run `py -m pip install -r requirements.txt`.
3. Run `py app.py` and verify `http://127.0.0.1:8765/api/v1/health`.

Endpoints: `/api/v1/health`, `/account`, `/quote?symbol=XAUUSD`, `/positions`.
