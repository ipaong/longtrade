# Code Map สำหรับลองเทรด

เอกสารนี้ชี้ว่าโค้ดส่วนใดรับผิดชอบอะไร และ task ใหม่ควรต่อเข้าที่ใด
โดยหลีกเลี่ยงการรื้อ KhunQuant เดิม

## ภาพรวม runtime ปัจจุบัน

```text
Browser
  +-- REST -----------------> web/backend/api
  +-- Chat WebSocket -------> /pico/ws proxy
                                  |
                                  v
                            khunquant gateway
                                  |
                         pkg/agent -> pkg/tools
                                  |
                         pkg/providers/broker
                                  |
                            pkg/exchanges/*
```

`web/backend` เป็น launcher/API process ส่วน `cmd/khunquant` เป็น
gateway/agent runtime ทั้งสองส่วนใช้ config และ workspace เดียวกัน

## จุดเริ่มโปรแกรม

| Path | หน้าที่ |
|---|---|
| `cmd/khunquant/main.go` | CLI entry point |
| `cmd/khunquant/internal/gateway` | ประกอบ agent, tools, channels และ runtime |
| `web/backend/main.go` | Web launcher entry point |
| `web/backend/api/router.go` | ลงทะเบียน REST routes |
| `web/backend/api/pico.go` | ตั้งค่าและ proxy WebSocket chat |
| `web/frontend/src/routes` | Frontend routes |
| `web/frontend/src/components/app-sidebar.tsx` | Navigation หลัก |

## AI และ deterministic tools

| Path | หน้าที่ |
|---|---|
| `pkg/agent` | Agent loop และ tool orchestration |
| `pkg/providers` | LLM provider adapters |
| `pkg/tools` | Tool definitions และ deterministic handlers |
| `pkg/tools/paper_trade.go` | Paper tool เดิมซึ่งยังไม่เก็บสถานะ |
| `pkg/tools/order_create.go` | Live order flow และ paper-mode redirect |
| `pkg/channels/pico` | Web chat channel |

หลักการคือ AI เลือก tool ได้ แต่ห้ามคำนวณ balance, P/L, risk หรือ execution
แทน domain service

## Trading providers

| Path | หน้าที่ |
|---|---|
| `pkg/providers/broker/provider.go` | Provider capability interfaces |
| `pkg/providers/broker/registry.go` | Provider registration |
| `pkg/providers/broker/permissions.go` | Account permissions |
| `pkg/providers/broker/losstracker.go` | Daily-loss tracking |
| `pkg/providers/broker/ratelimit.go` | Order rate limits |
| `pkg/exchanges/*` | Exchange-specific adapters |
| `pkg/exchanges/mt5` | MT5 stub และ extension point |

Interface ปัจจุบันใช้โครงสร้าง CCXT เป็นหลัก งาน MT5 ต้องเพิ่ม capability
แบบ additive เฉพาะข้อมูลบัญชีและ position โดยไม่เปลี่ยน provider เดิมทั้งหมด

## Config และ storage

| Path | หน้าที่ |
|---|---|
| `pkg/config/defaults.go` | ค่าเริ่มต้น รวม Paper mode และ leverage |
| `pkg/config/config.go` | Config schema และ loader |
| `web/backend/api/config.go` | Config REST API |
| `pkg/dca/store.go` | ตัวอย่าง SQLite store, WAL และ migration |
| `pkg/deltaneutral/store.go` | ตัวอย่าง persistent strategy store |
| `pkg/session` | JSONL conversation sessions |
| `pkg/snapshot` | Portfolio snapshots |
| `pkg/state` | Runtime state |

## ตำแหน่งใหม่ที่วางแผนไว้

| Path | Owner |
|---|---|
| `pkg/paper/types.go` | Account, Position, Order, Trade และ Proposal types |
| `pkg/paper/store.go` | SQLite schema, migrations และ transactions |
| `pkg/paper/engine.go` | Open, close, valuation และ SL/TP behavior |
| `pkg/paper/risk.go` | Lot, balance, loss, stale quote และ position limits |
| `pkg/paper/quote.go` | Quote interface และ adapters |
| `pkg/paper/*_test.go` | Unit/store/concurrency tests |
| `web/backend/api/paper.go` | Paper REST handlers |
| `web/frontend/src/api/paper.ts` | Typed API client |
| `web/frontend/src/features/trading` | Dashboard และ trade workflow |
| `bridge/mt5` | Python read-only Windows bridge |
| `pkg/exchanges/mt5` | Go client/adapter สำหรับ bridge |

ตำแหน่งเหล่านี้เป็นแผนเริ่มต้น Task `LT-101` ต้องตรวจ pattern เดิมอีกครั้ง
ก่อนสร้างไฟล์จริง

## Data flow ที่ต้องได้

### อ่าน Paper Portfolio

```text
Web/API or Chat tool
  -> pkg/paper service
  -> paper SQLite store + current quote
  -> AccountSnapshot
  -> REST JSON or tool result
```

### ทดลองเปิด Position

```text
prepare request
  -> validate input and quote freshness
  -> deterministic risk check
  -> short-lived Proposal
  -> explicit confirmation
  -> atomic execute
  -> order + position + audit event
```

### อ่าน XM Demo

```text
Web/API or Chat tool
  -> Go MT5 provider
  -> private REST
  -> Python bridge on Windows
  -> MetaTrader5 package
  -> XM Demo terminal
```

## Test map

| ชั้น | Test |
|---|---|
| `pkg/paper` | Table-driven unit tests, SQLite reopen, transaction rollback และ concurrency |
| `pkg/tools` | Tool input/output และยืนยันว่า tool เรียก service ไม่ทำ execution เอง |
| `web/backend/api` | Handler, JSON contract, validation และ HTTP status |
| `web/frontend` | TypeScript build, lint, component behavior และ responsive screenshots |
| `pkg/exchanges/mt5` | Fake bridge contract, timeout และ error mapping |
| `bridge/mt5` | Python unit tests โดย mock MetaTrader5 |
| End-to-end | Browser paper journey และ Windows XM Demo smoke test |

## ขอบเขตที่ไม่แตะใน MVP

- ไม่ rename internal Go packages
- ไม่รื้อ agent loop หรือ LLM providers
- ไม่เปลี่ยน DCA และ delta-neutral strategy code
- ไม่ผูก frontend เข้ากับ MT5 โดยตรง
- ไม่สร้าง real-money execution path
- ไม่ใช้ MCP เป็นส่วนหนึ่งของผลิตภัณฑ์

## เอกสารที่เกี่ยวข้อง

- `docs/longtrade-mvp-baseline.md`: baseline และคำสั่งรัน
- `docs/longtrade/GOAL.md`: เป้าหมายและ Definition of Done
- `docs/longtrade/TASKS.md`: task IDs และลำดับการทำงาน
