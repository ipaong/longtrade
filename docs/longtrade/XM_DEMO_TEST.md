# XM Demo field verification

This check requires an operator-owned XM Demo account and Windows MT5 terminal;
CI cannot supply those external credentials.

1. Confirm the MT5 title/account properties say **Demo**, record server and masked login.
2. Start `mt5_bridge/app.py` and GET health, account, XAUUSD quote, and positions.
3. Confirm timestamps are current and values match the terminal.
4. Attempt POST on every path and confirm HTTP 405.
5. Search the bridge for `order_send` and confirm there are no matches.

| Date | Tester | Demo server | Health | Account | Quote | Positions | POST rejected |
|---|---|---|---|---|---|---|---|
| Pending external XM Demo access | — | — | — | — | — | — | — |
