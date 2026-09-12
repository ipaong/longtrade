# เป้าหมายโครงการลองเทรด

สถานะ: ACTIVE

## เป้าหมายหลัก

สร้าง "ลองเทรด" เวอร์ชัน MVP v0.1 จาก KhunQuant ให้เป็นผู้ช่วยทดลองเทรด
ในเครื่องที่มือใหม่ใช้งานได้ผ่านหน้าเว็บและภาษาไทย โดยไม่ต้องเข้าใจ API,
broker infrastructure หรือการตั้งค่าระบบเชิงลึก

สถาปัตยกรรมผลิตภัณฑ์ใช้ Web UI เรียก Go Backend API โดยตรง ระบบ AI มีหน้าที่
ตีความความต้องการและเรียกฟังก์ชันที่กำหนดไว้เท่านั้น ส่วน backend เป็นผู้ตรวจสอบ
ความเสี่ยง บันทึกสถานะ และดำเนินการทุกอย่างแบบ deterministic

## หลักการตัดสินใจ

1. ใช้ง่ายก่อน
2. ปลอดภัยและเสถียรก่อนเพิ่มความสามารถ
3. เริ่มด้วย Paper Trading เสมอ
4. ไม่อนุญาตให้ AI เขียนหรือข้ามกฎการส่งคำสั่งซื้อขาย
5. ใช้โครงสร้าง KhunQuant เดิมและเปลี่ยนให้น้อยที่สุด
6. ทำงานทีละ task พร้อม test และ commit
7. MVP ไม่ใช้ MCP และไม่รองรับเงินจริง

## เส้นทางผู้ใช้หลัก

1. เปิดเว็บแล้วเห็นชัดว่ากำลังอยู่ในโหมดทดลอง
2. ดู Balance, Equity, กำไรขาดทุน และ Position ที่เปิดอยู่
3. ทดลองเปิด XAUUSD และเห็นผลกระทบต่อบัญชีจำลอง
4. ปิด Position หรือแก้ Stop Loss และ Take Profit ได้
5. ถามภาษาไทยเกี่ยวกับพอร์ตและความเสี่ยงผ่านหน้า Chat
6. เชื่อม XM Demo ผ่าน MT5 เพื่ออ่านบัญชี ราคา และ Position

## Definition of Done สำหรับ MVP v0.1

- หน้าเว็บใช้ชื่อ "ลองเทรด" และใช้งานได้บนมือถือ
- การติดตั้งใหม่เริ่มใน Paper Trading และปิด leverage
- Paper account มี Balance, Equity, Daily P/L และข้อมูลคงอยู่หลัง restart
- เปิด ปิด และแก้ SL/TP ของ simulated position ได้
- ทุกคำสั่งเปลี่ยนสถานะผ่าน backend validation และ confirmation
- Dashboard แสดง Position และประวัติการเทรดล่าสุด
- Chat ตอบคำถามพอร์ตและเรียก paper-trading operations ผ่าน deterministic tools
- MT5 Bridge อ่าน connection, account, quote และ positions จาก XM Demo ได้
- ไม่มีเส้นทางใดส่งคำสั่งไปบัญชีเงินจริง
- มี automated tests, คู่มือเริ่มระบบ และ safety checklist

## นอกขอบเขต MVP

- Real-money trading
- MCP และ ChatGPT plugins
- Strategy optimizer และ AI-generated strategies
- Autonomous trading
- Walk-forward testing
- Bot League
- Edge-decay monitoring
- รองรับ broker หลายรายนอกเหนือจากแหล่งราคาจำลองและ XM Demo

## กติกาการส่งมอบ

ทุก task ต้องเริ่มจากอ่าน implementation เดิม ใช้ component เดิมเมื่อเหมาะสม
แก้ให้น้อยที่สุด รัน test ที่เกี่ยวข้อง ตรวจ diff และ commit แยกจาก task อื่น
