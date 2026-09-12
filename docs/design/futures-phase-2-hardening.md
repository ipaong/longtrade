# Futures Phase 2 Hardening

## Summary

Phase 1 added the Binance and OKX futures foundation: exchange adapters, futures tools, CCXT contract symbol support, position reads, funding reads, and basic live entry flow. Phase 2 should make futures trading safer and more complete for AI-assisted control.

The goal is to move from "confirmed futures entry plus read-only monitoring" to "controlled futures operations with risk preflight, close/reduce workflows, and clear failure recovery."

## Goals

- Require explicit leverage-trading opt-in before any live futures mutation.
- Make market orders pass the same notional and risk checks as limit orders.
- Validate futures market metadata before execution, including newer OKX stock and pre-market perps.
- Verify fills and resulting positions after every live futures entry.
- Give users operational control after entry: close, reduce, move stop, cancel protection, and emergency flatten.
- Expose liquidation, margin health, and funding-cost estimates in user-facing tools.

## Key Gaps To Address

- Enforce `trading_risk.allow_leverage=true` for all live futures actions.
- Estimate market-order notional from ticker, mark price, or order-book midpoint before execution.
- Validate that requested instruments are active linear swap contracts, not spot or inactive markets.
- Use market precision, min amount, min cost, and leverage tier data where CCXT exposes it.
- Detect partial fills and fetch the resulting position before reporting success.
- Treat failed stop-loss or take-profit placement as a critical unprotected-position state.
- Inspect Binance/OKX account mode where possible so one-way and hedge-mode orders are handled intentionally.
- Estimate next funding payment from current funding rate, position notional, and next funding timestamp.
- Report liquidation distance, margin mode, collateral, margin ratio, and unrealized/realized PnL in a single risk view.

## Proposed Tools

- `futures_validate_market`: load market metadata and confirm the symbol is tradeable for futures.
- `futures_risk_summary`: summarize all open futures positions, margin health, liquidation distance, and PnL.
- `futures_estimate_funding_fee`: estimate next funding payment for a symbol or open position.
- `futures_close_position`: close a specific futures position with reduce-only order semantics.
- `futures_reduce_position`: reduce an open position by amount or percentage.
- `futures_modify_protection`: create, replace, or move stop-loss/take-profit protection orders.
- `futures_cancel_orders`: cancel futures open orders by symbol, type, or ID.
- `futures_emergency_flatten`: cancel futures orders and close all configured futures exposure after explicit confirmation.

## Acceptance Criteria

- Live futures mutations fail unless `trading_risk.allow_leverage=true`.
- Every live futures entry computes estimated notional and applies max-order checks, including market entries.
- Entry with protection either verifies stop-loss/take-profit orders or reports a critical unprotected-position warning.
- Users can close or reduce any futures position opened through the agent.
- Users can ask whether they are near liquidation and receive actionable margin-health output.
- Users can estimate upcoming funding fees for Binance/OKX positions.
- OKX symbols such as `TSLA/USDT:USDT`, `AMD/USDT:USDT`, and `ANTHROPIC/USDT:USDT` validate through public market metadata before trade.

## Test Plan

- Unit-test leverage opt-in enforcement for every live futures mutation tool.
- Unit-test market-order notional estimation using mocked ticker/mark price responses.
- Unit-test market validation for active swap contracts, inactive markets, spot symbols, and unknown symbols.
- Unit-test partial-fill and protection-failure result formatting.
- Unit-test close/reduce request mapping for long and short positions.
- Add integration-style public CCXT checks for OKX and Binance futures market metadata and funding rates, gated so they do not require credentials.
- Keep authenticated live-order tests opt-in only, never enabled by default in CI.

## Assumptions

- Binance TH and Bitkub remain spot-only and must not implement futures tools.
- Binance futures support continues to target USDT-M swaps first.
- OKX futures support uses CCXT unified symbols with settlement suffixes, for example `BTC/USDT:USDT`.
- Live position-closing tools should require explicit confirmation, even when order notional is small.
