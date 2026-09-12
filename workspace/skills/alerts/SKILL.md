---
name: alerts
description: Create, list, and cancel price and indicator alerts. Alerts fire when market conditions are met, using the cron scheduler for persistent monitoring.
---

# Alerts

Set up automated price and indicator alerts so the agent notifies you when market conditions are met.

## Alert Management Workflow

```
list active alerts → suggest threshold → create alert → confirm → cancel when done
```

1. List existing alerts with `action=list` to avoid duplicates.
2. Check the current price with `get_ticker` before suggesting a threshold.
3. Create the alert with appropriate condition and threshold.
4. Always explain the alert to the user before creating it.
5. Cancel with `action=cancel` when no longer needed.

## Tools

### set_price_alert
Create, list, or cancel price alerts.

**Create** (`action=create`):
- `provider`, `account`, `symbol`: as per market-data skill
- `condition`: "above" (fire when price exceeds threshold) or "below" (fire when price drops below)
- `threshold`: price level
- `message`: custom notification message
- `recurring`: false (one-shot, default) or true (continuous)

**List** (`action=list`): Show all active alerts. No additional params.

**Cancel** (`action=cancel`):
- `alert_id`: the ID from the list

### set_indicator_alert
Create, list, or cancel indicator-based alerts. Same action pattern as `set_price_alert`.

**Create** (`action=create`):
- `provider`, `account`, `symbol`, `timeframe`: as above
- `indicator`: RSI, MACD, SMA, or EMA
- `period`: lookback period (integer, e.g. 9, 20, 50, 200, 350). Defaults: SMA=20, EMA=20, RSI=14. Ignored for MACD (uses 12/26/9).
- `condition`: "above" or "below"
- `threshold`: indicator value threshold
- `message`, `recurring`: as above

## Alert Examples

```
set_price_alert: BTC/USDT below 60000 → notify me
set_price_alert: ETH/USDT above 4000 → sell signal
set_indicator_alert: BTC/USDT RSI(1h) below 30 → potential oversold entry
set_indicator_alert: BTC/USDT EMA period=9 (4h) above 90000 → bullish cross
set_indicator_alert: ZEC/USDT EMA period=350 (1d) above 587.84 → long entry
set_indicator_alert: BTC/USDT SMA period=200 (1d) below 60000 → trend break
```

## Notes
- Alerts are persisted in the cron scheduler and survive agent restarts.
- One-shot alerts (recurring=false) auto-cancel after firing.
- Alerts poll every minute — they may fire up to 1 minute after the condition is met.
- The cron tool must be enabled for alerts to function.
- Alerts request `period * 3` candles (min 100, max 500). Exchanges with shallow history (e.g. Bitkub TradingView feed ~180 daily bars) will return a `not enough data: have X bars, need >= Y` error for very long periods on long timeframes.
