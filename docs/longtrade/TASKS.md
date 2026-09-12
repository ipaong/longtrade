# แผนงานลองเทรด

เอกสารนี้เป็นคิวงานหลัก ผู้ใช้สามารถสั่งด้วยรหัส เช่น `ทำ LT-101`

สถานะ:

- `DONE`: ทำและ commit แล้ว
- `READY`: เริ่มทำได้
- `WAITING`: ต้องรอ task ก่อนหน้า
- `DEFERRED`: ยังไม่ทำในช่วงนี้

## Phase 0: Baseline และ Safety

| ID | สถานะ | งาน | ขึ้นกับ | เกณฑ์ตรวจรับ |
|---|---|---|---|---|
| LT-000 | DONE | สำรวจระบบและบันทึก API-first baseline | - | มี architecture map, คำสั่งรัน, Paper default, leverage off และ tests |
| LT-001 | DONE | ตรวจและล็อก safety invariants ทั้งระบบ | LT-000 | มีตารางทุก write path, ยืนยัน live tools ปิด และเพิ่ม test เมื่อพบช่องว่าง |

## Phase 1: Persistent Paper Trading

| ID | สถานะ | งาน | ขึ้นกับ | เกณฑ์ตรวจรับ |
|---|---|---|---|---|
| LT-101 | DONE | กำหนด paper account, position, order และ trade types | LT-000 | Types รองรับ XAUUSD, lot, side, SL/TP, P/L และมี validation tests |
| LT-102 | DONE | สร้าง SQLite paper store และ migrations | LT-101 | บันทึก account/positions/orders/trades, เปิดใหม่ข้อมูลไม่หาย และ tests ผ่าน |
| LT-103 | DONE | สร้างบัญชีเริ่มต้นและคำสั่ง reset | LT-102 | บัญชีใหม่มี 10,000 USD, reset ทำงานแบบ atomic และมี tests |
| LT-104 | DONE | สร้าง quote source interface และ fake quote | LT-101 | Engine รับราคาแบบ inject ได้, ตรวจ stale quote และทดสอบโดยไม่ใช้ network |
| LT-105 | DONE | เปิด simulated market position | LT-102, LT-104 | ตรวจ lot/balance/risk, บันทึก order กับ position ครั้งเดียว และมี idempotency test |
| LT-106 | DONE | คำนวณ Equity และ unrealized P/L | LT-105 | BUY/SELL คำนวณถูกต้องจาก bid/ask และ snapshot มี timestamp |
| LT-107 | DONE | ปิด position และบันทึก realized P/L | LT-106 | ปิดเต็มจำนวนได้, balance เปลี่ยนถูกต้อง และ trade history ตรวจสอบย้อนกลับได้ |
| LT-108 | DONE | เพิ่มและแก้ Stop Loss / Take Profit | LT-105 | validate ระดับราคา, บันทึกค่า และ trigger ปิด position ได้แบบ deterministic |
| LT-109 | DONE | Emergency close และ paper risk limits | LT-107, LT-108 | ปิดเฉพาะ position ที่มี, ป้องกัน order ซ้ำ และบันทึก audit event |

## Phase 2: Backend API (ดูรายละเอียดใน [PHASE_2_BACKEND_API.md](file:///home/user/Desktop/ลองเทรด/docs/longtrade/PHASE_2_BACKEND_API.md))

| ID | สถานะ | งาน | ขึ้นกับ | เกณฑ์ตรวจรับ |
|---|---|---|---|---|
| LT-201 | DONE | REST API สำหรับ account, positions และ trades | LT-106, LT-107 | JSON schema คงที่, มี mode/as_of/stale และ handler tests |
| LT-202 | DONE | API เตรียมและยืนยัน simulated trade | LT-105 | prepare คืน proposal ID, execute ตรวจซ้ำ และ proposal หมดอายุได้ |
| LT-203 | DONE | API ปิด position, แก้ SL/TP และ emergency close | LT-108, LT-109 | ทุก endpoint validate input, ใช้ service เดียวกัน และมี tests |
| LT-204 | DONE | ทำ error format และ API contract tests | LT-201, LT-203 | UI แยก validation/conflict/unavailable/internal error ได้ |

## Phase 3: ลองเทรด Web UI และ Chat

| ID | สถานะ | งาน | ขึ้นกับ | เกณฑ์ตรวจรับ |
|---|---|---|---|---|
| LT-301 | DONE | เปลี่ยน branding ที่ผู้ใช้เห็นเป็น "ลองเทรด" | LT-000 | Title, sidebar, onboarding และ copy เปลี่ยน โดย license/upstream attribution อยู่ครบ |
| LT-302 | DONE | ลด navigation เหลือ Chat, Portfolio และ Try Trade | LT-301 | เมนูนักพัฒนาอยู่ใต้ Advanced Settings และ route เดิมยังเข้าถึงได้ |
| LT-303 | DONE | สร้าง paper dashboard | LT-201, LT-302 | แสดง Balance, Equity, Daily P/L, Positions, Recent Trades และ connection |
| LT-304 | DONE | สร้าง flow ทดลองเปิด/ปิด/แก้ SL/TP | LT-202, LT-203 | มี preview และ confirmation, loading/error/empty statesครบ |
| LT-305 | WAITING | เชื่อมคำถามภาษาไทยกับ deterministic tools | LT-201, LT-203 | intent ตัวอย่างหลักตอบได้ และ AI ไม่มี execution logic |
| LT-306 | READY | ตรวจ mobile, accessibility และ visual regression | LT-303, LT-304 | responsive/keyboard implementation และ build ผ่าน; รอ screenshot runner |

## Phase 4: MT5 และ XM Demo แบบ Read-only

| ID | สถานะ | งาน | ขึ้นกับ | เกณฑ์ตรวจรับ |
|---|---|---|---|---|
| LT-401 | DONE | เพิ่ม account/position/protection interfaces ที่ MT5 ต้องใช้ | LT-201 | เป็น additive interface, ไม่ผูก UI กับ MT5 และมี compile-time tests |
| LT-402 | DONE | กำหนด REST contract ของ MT5 Bridge และ fake server | LT-401 | ครบ health/account/quote/positions, versioned schema และ contract tests |
| LT-403 | DONE | สร้าง Python MT5 Bridge แบบ read-only | LT-402 | รันบน Windows, ไม่เปิด write endpoint และมี health diagnostics |
| LT-404 | DONE | เชื่อม Go MT5 provider กับ Bridge | LT-402, LT-403 | timeout/error และ fake integration tests ผ่าน |
| LT-405 | READY | ทดสอบกับ XM Demo terminal จริง | LT-404 | ต้องใช้ Windows, MT5 และข้อมูล XM Demo ของผู้ทดสอบภายนอก |
| LT-406 | DEFERRED | เพิ่ม order execution สำหรับ XM Demo | LT-405 | เริ่มได้หลัง read-only เสถียรและผ่าน safety review แยกต่างหาก |

## Phase 5: Release v0.1

| ID | สถานะ | งาน | ขึ้นกับ | เกณฑ์ตรวจรับ |
|---|---|---|---|---|
| LT-501 | WAITING | End-to-end demo journey | LT-305, LT-405 | ติดตั้งใหม่, เปิด/ปิด paper trade, restart และอ่าน XM Demo ผ่านครบ |
| LT-502 | WAITING | Packaging, runbook และ final safety audit | LT-501 | คำสั่งเริ่มสั้น, recovery ชัด และพิสูจน์ว่าไม่มี real-money path |

## ลำดับที่แนะนำตอนนี้

ทำ `LT-305` เพื่อเชื่อม Chat กับ deterministic tools จากนั้นใช้ Windows/XM Demo
ทำ field verification `LT-405` และปิด release gate `LT-501`–`LT-502`
