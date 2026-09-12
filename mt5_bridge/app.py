"""Read-only localhost bridge for a running MetaTrader 5 Demo terminal."""
from datetime import datetime, timezone
import os
from flask import Flask, jsonify, request
import MetaTrader5 as mt5

app = Flask(__name__)

def connected():
    return mt5.initialize()

@app.get("/api/v1/health")
def health():
    ok = connected()
    terminal = mt5.terminal_info() if ok else None
    return jsonify(status="ok" if ok else "unavailable", version="v1", terminal=getattr(terminal, "name", "")), 200 if ok else 503

@app.get("/api/v1/account")
def account():
    if not connected(): return jsonify(error="MT5 terminal unavailable"), 503
    value = mt5.account_info()
    if value is None: return jsonify(error="Demo account unavailable"), 503
    return jsonify(login=value.login, server=value.server, currency=value.currency, balance=value.balance, equity=value.equity, margin=value.margin, free_margin=value.margin_free)

@app.get("/api/v1/quote")
def quote():
    symbol = request.args.get("symbol", "XAUUSD")
    if not connected(): return jsonify(error="MT5 terminal unavailable"), 503
    value = mt5.symbol_info_tick(symbol)
    if value is None: return jsonify(error="Symbol unavailable"), 404
    return jsonify(symbol=symbol, bid=value.bid, ask=value.ask, as_of=datetime.fromtimestamp(value.time, timezone.utc).isoformat())

@app.get("/api/v1/positions")
def positions():
    if not connected(): return jsonify(error="MT5 terminal unavailable"), 503
    values = mt5.positions_get() or []
    return jsonify([dict(ticket=p.ticket, symbol=p.symbol, side="buy" if p.type == mt5.POSITION_TYPE_BUY else "sell", volume=p.volume, open_price=p.price_open, current_price=p.price_current, stop_loss=p.sl, take_profit=p.tp, profit=p.profit) for p in values])

if __name__ == "__main__":
    app.run(host="127.0.0.1", port=int(os.getenv("MT5_BRIDGE_PORT", "8765")))
