# Phase 2: Backend REST API for LongTrade Paper Trading (LT-201 - LT-204)

เอกสารนี้ระบุรายละเอียดข้อกำหนดทางเทคนิค (Task Specifications) ของ **Phase 2: Backend API** สำหรับระบบ LongTrade (Paper Trading MVP) เพื่อใช้สั่งงานผ่าน Cloud Agent (เช่น Codex Cloud หรือ GitHub Actions) ให้ทำการตรวจสอบ ทดสอบ และผสานโค้ดเข้าสู่ repository

---

## 1. วัตถุประสงค์และสถาปัตยกรรม (Architecture & Invariants)

- **Execution Invariant**: ตรรกะการคำนวณกำไร/ขาดทุน (P/L), มาร์จิ้น, การตรวจสอบความเสี่ยง (Risk limits), สถานะบัญชี และการบันทึกฐานข้อมูล SQLite ต้องดำเนินการผ่าน `pkg/paper.Engine` และ `pkg/paper.Store` เท่านั้น
- **API Layer Role**: เลเยอร์ HTTP REST API (`web/backend/api/paper.go` และ `web/backend/api/router.go`) ทำหน้าที่เป็น Gateway:
  1. ถอดรหัส HTTP Request JSON และ validate input เบื้องต้น
  2. จัดการ Life-cycle ของ short-lived trade proposal
  3. เรียกใช้งาน `pkg/paper.Engine` แบบ thread-safe
  4. ตอบกลับด้วยโครงสร้าง JSON มาตรฐาน และส่ง HTTP status codes ที่ถูกต้อง
- **Zero Real-Money Path**: แยกส่วนออกจาก live exchange tool ทั้งหมดอย่างเด็ดขาด บัญชีเริ่มต้นคือ default paper account (`paper-main`) ใน SQLite

---

## 2. รายละเอียดงานย่อย (Task Breakdown)

### LT-201: REST API สำหรับ Account, Positions และ Trades
- **Endpoints**:
  - `GET /api/paper/account`: คืนค่า `paper.AccountSnapshot` ปัจจุบัน (Balance, Equity, Unrealized P/L, Daily Realized P/L, รายการ Positions ที่เปิดอยู่, รายการ Trades ล่าสุด, Mode="paper", AsOf timestamp)
  - `POST /api/paper/account/reset` (และ alias `/api/paper/reset`): รีเซ็ตบัญชีจำลองกลับสู่เงินตั้งต้น $10,000 USD แบบ atomic โดยล้าง positions, orders, trades, และบันทึก audit event
  - `GET /api/paper/positions`: คืนค่า JSON array ของ `paper.Position` ที่เปิดอยู่
  - `GET /api/paper/trades`: คืนค่า JSON array ของประวัติการเทรด `paper.Trade` (รองรับ query parameter `limit`, ค่าเริ่มต้น 50)
- **เกณฑ์ตรวจรับ**:
  - Schema สอดคล้องกับ types ใน `pkg/paper`
  - หาก quote stale จะส่งคืน HTTP 503 `QUOTE_STALE`
  - มี unit tests ครอบคลุม

### LT-202: Simulated Trade Proposal และ Execution (Two-Step Workflow)
- **Endpoints**:
  - `POST /api/paper/trade/prepare`:
    - **Request**:
      ```json
      {
        "symbol": "XAUUSD",
        "side": "BUY",
        "volume_lots": 0.05,
        "stop_loss": 2390.0,
        "take_profit": 2420.0
      }
      ```
    - **Logic**: ตรวจสอบ input, ดึง executable quote ล่าสุด, สร้าง `paper.Proposal` ชั่วคราว (TTL 30 วินาที) พร้อม `proposal_id` และ executable price
    - **Response**: คืนค่า Proposal JSON
  - `POST /api/paper/trade/confirm`:
    - **Request**:
      ```json
      {
        "proposal_id": "prop_xxxx",
        "client_request_id": "req_yyyy"
      }
      ```
    - **Logic**: ตรวจสอบว่า Proposal ยังไม่หมดอายุ จากนั้นส่งคำสั่งเปิด position ไปยัง `engine.OpenPosition` แบบ atomic และลบ proposal ออก (consumed)
    - **Response**: คืนค่า `{"position": {...}}`
    - **Idempotency**: หากใช้ `client_request_id` ซ้ำ ส่ง HTTP 409 `CONFLICT`
    - **Expiration**: หาก proposal หมดอายุหรือไม่มี ส่ง HTTP 410 `EXPIRED`

### LT-203: Position Management, แก้ SL/TP และ Emergency Close
- **Endpoints**:
  - `POST /api/paper/positions/{id}/close`:
    - **Request**:
      ```json
      {
        "client_request_id": "close_xxx",
        "reason": "manual"
      }
      ```
    - **Logic**: ปิด position ด้วยราคาตลาดปัจจุบันผ่าน `engine.ClosePosition` และบันทึก realized P/L ลง ledger
    - **Response**: คืนค่า `paper.Trade` ที่ปิดสำเร็จ
    - **Not Found**: หากไม่พบ position ส่ง HTTP 404 `NOT_FOUND`
  - `POST /api/paper/positions/{id}/protection`:
    - **Request**:
      ```json
      {
        "stop_loss": 2395.0,
        "take_profit": 2430.0
      }
      ```
    - **Logic**: ปรับระดับ Stop Loss / Take Profit ผ่าน `engine.UpdateProtection`
    - **Response**: คืนค่า `paper.Position` ที่อัปเดตแล้ว
  - `POST /api/paper/emergency-close`:
    - **Logic**: สั่งปิด position ทั้งหมดทันทีผ่าน `engine.EmergencyClose`
    - **Response**: คืนค่า `{"closed_trades": [...], "count": N}`

### LT-204: Error Format Envelope และ Contract Test Suite
- **Standard JSON Error Envelope**:
  ```json
  {
    "error": {
      "code": "VALIDATION_ERROR" | "NOT_FOUND" | "CONFLICT" | "QUOTE_STALE" | "EXPIRED" | "INTERNAL_ERROR",
      "message": "Human readable error description",
      "field": "field_name_if_applicable"
    }
  }
  ```
- **HTTP Status Code Mapping**:
  - `400 Bad Request`: `VALIDATION_ERROR` (เช่น ล็อตผิด, ค่า SL/TP ไม่ถูกต้อง, JSON เสีย)
  - `404 Not Found`: `NOT_FOUND` (ไม่พบ position ID)
  - `409 Conflict`: `CONFLICT` (idempotency key ชน)
  - `410 Gone`: `EXPIRED` (proposal หมดอายุ)
  - `503 Service Unavailable`: `QUOTE_STALE` (ราคาตลาดเก่าเกินเกณฑ์)
  - `500 Internal Server Error`: `INTERNAL_ERROR`
- **Unit & Contract Tests**:
  - มีไฟล์ทดสอบ `web/backend/api/paper_test.go` ครอบคลุมทุก endpoint และ error cases
  - ใช้ `paper.FakeQuoteSource` และ SQLite temp store ในการทดสอบ เพื่อให้รันได้แบบ deterministic และไม่ต้องต่อ network ภายนอก

---

## 3. รายการไฟล์ที่เกี่ยวข้อง (Code Map)

1. `pkg/paper/engine.go`: เพิ่มเมธอด `CurrentQuote` และ `PrepareProposal`
2. `pkg/paper/types.go`: โครงสร้าง `Proposal`, `AccountSnapshot`, `Position`, `Trade`, `Side`, `CloseReason`
3. `web/backend/api/router.go`: เพิ่มฟิลด์ `paper*` ใน `Handler` และลงทะเบียน `h.registerPaperRoutes(mux)`
4. `web/backend/api/paper.go`: REST Handlers ทั้งหมดของ paper trading
5. `web/backend/api/paper_test.go`: ชุด unit tests ทั้งหมดของ paper API

---

## 4. คำสั่งทดสอบสำหรับ Cloud Runner / Codex

เมื่อสั่งงานใน Cloud ให้รันคำสั่ง:
```bash
# ตรวจสอบว่าแพ็กเกจ paper เดิมทำงานผ่านทั้งหมด
go test -v -count=1 ./pkg/paper/...

# ทดสอบ API endpoints ของ paper ทั้งหมด
go test -v -count=1 ./web/backend/api/... -run "TestPaper"

# ตรวจสอบการคอมไพล์และทดสอบทั้งหมดใน web backend
go test -v ./web/backend/api/...
```
