# สนาม · Football Analyzer

เว็บวิเคราะห์ฟุตบอลส่วนตัวภาษาไทย ใช้ Go, Next.js และ PostgreSQL จริง รองรับข้อมูลจำลองสำหรับทดสอบ และตัวเชื่อมต่อ API-Football สำหรับข้อมูลจริง โดยค่าเริ่มต้นยังเป็น mock:

**Today → Recommendation → เลือกเล่น → My Picks → History**

รุ่นนี้ใช้บน localhost เข้าสู่ระบบด้วยอีเมลและรหัสผ่านเฉพาะเจ้าของบัญชี และยังไม่พร้อมเปิดสู่เครือข่ายสาธารณะ โหมด mock ใช้ expected goals จำลอง ส่วนโหมดจริงใช้โมเดลพื้นฐานจากผลย้อนหลัง ซึ่งยังไม่มีการยืนยันประสิทธิภาพการทำนาย

## มีอะไรให้ใช้บ้าง

| หน้า | ใช้ทำอะไร |
| --- | --- |
| เข้าสู่ระบบ `/login` | เข้าสู่ระบบด้วยอีเมลและรหัสผ่านของเจ้าของ |
| วันนี้ `/` | ดูคู่แข่งขัน ราคา โอกาสชนะ/ได้กำไร และสถานะ PLAY / WATCH / PASS |
| รายการที่เลือก `/picks` | ดูรายการของบัญชีตัวเอง และยกเลิกรายการที่ยังไม่สรุปผล |
| ประวัติและผลลัพธ์ `/history` | ดูผล กำไรจำลอง ROI และ Brier score พร้อมกรองวันที่ |
| รายละเอียด `/matches/:id` | ดู fair odds, expected goals, lineup และ snapshots ย้อนหลัง |

คำแนะนำระบบกับรายการที่คุณเลือกเก็บแยกกัน ระบบไม่เลือกให้โดยอัตโนมัติ และราคาใหม่ไม่เปลี่ยนค่าที่บันทึกไว้ตอนเลือก

## เริ่มใช้งาน

หากต้องการรันทุกส่วนด้วย Docker โดยไม่ติดตั้ง Go/Node.js บน VPS ดูหัวข้อ [รันทั้งแอปด้วย Docker](#รันทั้งแอปด้วย-docker) ด้านล่าง

ต้องมี Go 1.26+, Node.js 24 LTS, npm, Docker และ Docker Compose ไม่ต้องติดตั้ง PostgreSQL ลงเครื่อง

```bash
nvm install
nvm use
# สำหรับการติดตั้งครั้งแรกเท่านั้น หากมี .env อยู่แล้วให้ข้ามบรรทัดนี้
cp -n .env.example .env
# แก้ OWNER_EMAIL ใน .env ให้ตรงกับบัญชีเจ้าของก่อนเริ่มระบบ
cd frontend
npm ci
cd ..
make dev
```

หากมี `.env` อยู่แล้ว ให้ใช้ไฟล์เดิม ไม่ต้อง copy ทับ `make dev` เริ่ม PostgreSQL, รัน migrations, เพิ่มข้อมูลจำลอง แล้วเปิด API และ frontend ไม่ลบฐานข้อมูลหรือประวัติเดิม กด Ctrl+C เพื่อหยุด API/frontend; PostgreSQL และ volume ยังอยู่

เปิด **http://localhost:3000** (ใช้ hostname นี้ให้ตรงกับ FRONTEND_ORIGIN)

ระบบเปิดให้ใช้งานได้เฉพาะบัญชีที่อีเมลตรงกับ `OWNER_EMAIL` ใน `.env` เท่านั้น ปิดการสมัครผ่านหน้าเว็บและ API รวมถึงปฏิเสธเซสชันเดิมของบัญชีอื่น เก็บบัญชีและประวัติเดิมทั้งหมดไว้ หากมีบัญชีอยู่แล้ว ให้ตั้งอีเมลเดิมและใช้รหัสผ่านเดิมได้ทันที

สำหรับฐานข้อมูลใหม่ ให้รันคำสั่งต่อไปนี้ก่อน `make dev` เพื่อสร้างบัญชีเจ้าของจากเครื่องที่ดูแลระบบ:

```bash
make db-up
make migrate
make seed
make owner
```

`make owner` อ่านอีเมลจาก `OWNER_EMAIL` และถามรหัสผ่านโดยไม่แสดงตัวอักษรหรือใส่รหัสผ่านใน command line ใช้ภาษาอังกฤษ ตัวเลข และอักขระพิเศษทั่วไป 9–72 ตัวอักษร ไม่มีช่องว่าง ตั้งชื่อที่แสดงผ่าน `OWNER_NAME` ได้ (ค่าเริ่มต้น Owner) คำสั่งไม่เปลี่ยนรหัสผ่านหรือเขียนทับบัญชีเดิม บัญชีแรกได้รับรายการเดิมที่ยังไม่มีเจ้าของ

เซสชันมีอายุ 7 วัน รหัสผ่านเก็บเป็น bcrypt hash และ cookie เป็น HttpOnly ระบบยังไม่มีการยืนยันอีเมลหรือกู้รหัสผ่าน Migration 002 ไม่รองรับ rollback เพื่อรักษาบัญชีและเจ้าของรายการ หากย้าย `.env` ไปเครื่องใหม่ ต้องตั้ง `OWNER_EMAIL` ให้ตรงกับบัญชีในฐานข้อมูลที่ย้ายไปด้วย

หลังอัปเดตโค้ด ให้หยุด `make dev` เดิมด้วย Ctrl+C แล้วรัน `make dev` ใหม่ เพื่อรัน migration และเริ่ม API รุ่นล่าสุด หาก dependencies เปลี่ยน ให้รัน `npm ci` ใน `frontend` อีกครั้ง

- API: http://127.0.0.1:8080
- PostgreSQL: 127.0.0.1:55432
- `make db-down` หยุด container โดยเก็บ volume ไว้

ติดตั้ง Go dependencies ครั้งแรกด้วย `cd backend && go mod download` หรือให้คำสั่ง `go run` ดาวน์โหลดให้อัตโนมัติ

## ลองใช้งานครั้งแรก

1. เข้าสู่ระบบด้วยบัญชีเจ้าของ แล้วเปิดหน้า **วันนี้**
2. อ่านช่อง **เล่นอะไร** บนการ์ด เช่น Liverpool, Barcelona -0.25 หรือ สูง 2.25 และตรวจสถานะคำแนะนำ
3. กด **เลือกเล่น** บนรายการ PLAY หรือ WATCH เพื่อบันทึกราคาและผลประเมิน ณ เวลานั้น
4. เปิด **รายการที่เลือก** เพื่อตรวจรายการที่บันทึก หากเปลี่ยนใจให้กดยกเลิกก่อนรายการสรุปผล
5. จำลองผลการแข่งขันตามคำสั่งด้านล่าง แล้วเปิด **ประวัติและผลลัพธ์**

หน้า History แบ่งรายการที่คุณเลือกตาม **วันที่เลือก** และแบ่งคำแนะนำระบบ PLAY / WATCH / PASS ตาม **วันที่สร้างคำแนะนำ** พร้อมหัววันที่และจำนวนรายการ ส่วนตัวกรองวันที่ด้านบนใช้ **วันที่แข่งขัน** จึงอาจต่างจากวันที่ของหัวกลุ่มได้

### อ่านค่าบนหน้าจอ

| ค่า | ความหมาย |
| --- | --- |
| PLAY | EV ตั้งแต่ 5% ขึ้นไปตามค่าเริ่มต้น และข้อมูลผ่านเกณฑ์การเลือก |
| WATCH | EV เป็นบวกแต่ยังต่ำกว่าเกณฑ์ PLAY เลือกได้หากผู้ใช้ตัดสินใจเอง |
| PASS | EV ไม่เป็นบวก หรือข้อมูลไม่พร้อม เช่น ราคาเก่าหรือเริ่มแข่งขันแล้ว เลือกไม่ได้ |
| ราคา | Decimal odds เช่น 2.08: หากชนะเต็มจะได้กำไร 1.08 units ต่อ 1 unit |
| EV | กำไรเฉลี่ยที่โมเดลคาดต่อทุน 1 unit เช่น +12% คือ +0.12 unit โดยเฉลี่ยตามผลประเมิน |
| Fair odds | ราคาคุ้มทุนตามความน่าจะเป็นของโมเดล โดยคำนึงถึงคืนทุนและผลครึ่งด้วย |
| Unit | หน่วยทุนจำลอง รุ่นนี้กำหนด 1 unit ต่อรายการ และยังไม่มีช่องใส่ทุนหรือแปลงเป็นบาท |
| ROI | กำไรสุทธิ ÷ ทุนของรายการที่สรุปผลแล้ว × 100 เช่น กำไร 1.16 units จากทุน 4 units ≈ +29% |
| Brier score | วัดว่าความน่าจะเป็น 1X2 ใกล้ผลจริงแค่ไหน ค่ายิ่งต่ำยิ่งดี ช่วง 0–2 และไม่ใช่เปอร์เซ็นต์ความแม่นยำ |

ROI ไม่รวมรายการรอผล ยกเลิก หรือ void แต่รวมรายการคืนทุน ส่วน Brier score ใช้ prediction ล่าสุดก่อนเริ่มแข่งขันของแต่ละโมเดล ตัวเลขในรุ่นนี้มาจากข้อมูลจำลอง EV และโอกาสชนะเป็นการประเมินของโมเดล ไม่ใช่ผลกำไรที่รับประกัน การกดเลือกบันทึกสถิติในแอปเท่านั้น

## คำสั่งแยกแต่ละส่วน

### ใช้งานจากมือถือ

หน้าเว็บรองรับเบราว์เซอร์มือถือ เมนูและข้อมูลบัญชีจัดเรียงใหม่บนจอเล็ก โดยทดสอบหน้าเข้าสู่ระบบและหน้า Today หลังเข้าสู่ระบบที่ความกว้าง 320, 390 และ 768 พิกเซล

`localhost:3000` บนมือถือหมายถึงตัวมือถือเอง จึงไม่เชื่อมกับแอปบนคอมพิวเตอร์ การเปิดจากทุกที่โดยไม่ติดตั้งแอปเพิ่มต้องจัดเตรียม HTTPS endpoint ก่อน รุ่นปัจจุบันยังรันบน localhost และยังไม่ได้เผยแพร่ ต้องเตรียมเซิร์ฟเวอร์หรือบริการ tunnel ที่เหมาะสม พร้อมตั้ง origin, secure session cookies และการจำกัดการเข้าถึงก่อนใช้งานจริง ข้อมูลบัญชีเจ้าของและการปิดสมัครยังต้องคงไว้

### เริ่มแต่ละ process แยกกัน

```bash
make db-up
make migrate
make migrate-status
make seed
make backend       # terminal 1
make frontend      # terminal 2
make worker        # terminal 3: เพิ่ม snapshot ราคา/lineup/prediction/recommendation
```

seed มี 3 คู่วันนี้: PLAY, WATCH, PASS และ 2 คู่ที่จบแล้วพร้อม picks ตัวอย่างสำหรับ History ภายในวันเดียวกัน seed ซ้ำไม่สร้าง fixture หรือ source snapshot ซ้ำ แต่เพิ่ม prediction/recommendation รุ่นใหม่ของคู่ที่ยังไม่จบ และไม่แก้ picks เดิม Fixtures วันนี้ขึ้นกับ clock และ APP_TIMEZONE; demo picks ที่สร้างหลังมีสมาชิกยังไม่มีเจ้าของ จึงไม่รวมในประวัติส่วนตัว; ตัวอย่าง seed picks ใช้ flag `demo` และแสดงคำว่า “ตัวอย่าง” ใน UI

ราคามีอายุ 15 นาที: เมื่อหมดอายุให้รัน `make worker` แล้วกด “โหลดข้อมูลใหม่” แอปไม่ sync อัตโนมัติ การแสดงผลวันนี้ตรวจความพร้อมปัจจุบันอีกครั้ง แต่ประวัติแสดง recommendation ที่บันทึกจริงในแต่ละ generation

## จำลองผลการแข่งขันของ pick ที่เลือกเอง

คัดลอก match UUID จาก URL หน้ารายละเอียด หรือ `GET /api/v1/matches/today` แล้วรัน:

```bash
make worker ARGS='--finish-match MATCH_UUID --home 2 --away 0'
```

คำสั่งนี้ใช้กับ fixture ของ mock provider เท่านั้น บันทึกผลจำลองและ settle picks ที่ยังไม่ได้สรุปผลใน transaction หลังจากนั้นเปิด My Picks/History ดูผลและ ROI Worker หรือ seed ครั้งต่อไปจะไม่ย้อนสถานะคู่ที่จบแล้วกลับไปเป็น scheduled ไม่มีการเชื่อมต่อ provider จริง

## เปิดใช้ API-Football กับข้อมูลจริง

1. สมัครหรือเข้าสู่ระบบที่ [API-Football Dashboard](https://dashboard.api-football.com/) แล้วยืนยันอีเมล
2. คัดลอก API key จาก **Account → My Access** มาใส่ `.env` เท่านั้น
3. ตั้งค่าดังนี้ โดยไม่ส่งคีย์ผ่านแชตหรือใส่ในตัวแปร NEXT_PUBLIC:

```dotenv
FOOTBALL_PROVIDER=api-football
API_FOOTBALL_KEY=ใส่คีย์จริงของคุณ
API_FOOTBALL_LEAGUES=39,140,135,78,61,5
API_FOOTBALL_MAX_REQUESTS=50
```

4. หยุด `make dev` เดิม แล้วรัน `make dev` ใหม่เพื่อให้ API และ frontend อ่าน configuration เดียวกัน
5. อีก terminal รัน `make worker` แล้วกด **โหลดข้อมูลใหม่** การเปิดหน้าเว็บไม่เรียก provider ซ้ำเอง

League IDs ข้างต้นคือพรีเมียร์ลีกอังกฤษ, ลาลีกา, เซเรียอา, บุนเดสลีกา และลีกเอิง ดูสิทธิ์ฤดูกาลและจำนวนคำขอใน dashboard ของผู้ให้บริการ แผนฟรีจำกัดฤดูกาลและโควตา จึงไม่ได้รับประกันว่าจะอ่านข้อมูลปัจจุบันได้ [ราคาและข้อจำกัด](https://www.api-football.com/pricing/)

Worker ดึงคู่วันนี้และอัปเดตคู่ที่เก็บไว้แต่ยังไม่จบ รวมถึงราคา pre-match, ผล regulation time และผลย้อนหลังที่ใช้กับโมเดล รายชื่อผู้เล่น/รายงานบาดเจ็บดึงภายใน 90 นาทีก่อนเริ่มเกมเมื่อมีข้อมูล ไม่เติมชื่อหรือราคา mock หาก API ไม่มีข้อมูล และไม่บันทึกข้อมูลครึ่งชุดเมื่อคำขอใดล้มเหลว

- เก็บประวัติ mock เดิมไว้ แต่แสดงเฉพาะ source ที่เลือก การกลับไป `FOOTBALL_PROVIDER=mock` จะเห็นข้อมูลเดิมอีกครั้ง
- รองรับเฉพาะ market เต็มเวลาที่อ่านได้อย่างชัดเจน: Match Winner, Asian Handicap และ Goals Over/Under ไม่อ่าน market ครึ่งแรกหรือ European Handicap เป็นตลาดเดียวกัน
- `captured_at` ของราคาใช้เวลา `update` จาก API ไม่ใช่เวลาที่เราเพิ่งดึงมา หากเก่าเกิน ODDS_MAX_AGE จะเป็น PASS การซิงก์ไม่ได้รับประกันว่าต้นทางจะมีราคาใหม่
- โมเดล `v1-recent-goals-poisson` ใช้ผลลีกย้อนหลัง 180 วัน โดยตัด 24 ชั่วโมงล่าสุดออก ต้องมีเกมเหย้าของทีมเจ้าบ้านและเกมเยือนของทีมเยือนอย่างน้อยฝั่งละ 3 นัด ปรับค่าเฉลี่ยด้วยผลลีกเทียบเท่า 2 นัด แล้วเฉลี่ยการยิงกับการเสียของคู่แข่ง นี่เป็น baseline จากสกอร์จริง ไม่ใช่ measured xG หรือโมเดลที่ผ่านการยืนยันความแม่นยำแล้ว
- หากราคา/ผลย้อนหลังไม่เพียงพอ แสดง PASS และไม่สร้าง probability/EV ปลอม ผลหลังต่อเวลาหรือดวลจุดโทษใช้เฉพาะ `score.fulltime` เพื่อสรุปผลรายการ
- จำกัดคำขอ **ต่อรอบ** ไม่ใช่ต่อวัน ไม่มี retry หรือ scheduler อัตโนมัติ ให้ตรวจโควตาใน dashboard ก่อนซิงก์ซ้ำ หากเกินงบต่อรอบให้ลดลีกหรือเพิ่มงบตามสิทธิ์แพ็กเกจ
- คำอธิบายยังเป็นข้อความตามกฎใน Go ไม่ได้เรียก AI API

การทดสอบตัวเชื่อมต่อใช้ HTTP responses จำลอง ตรวจ mapping, pagination, การไม่เผยคีย์, market ที่รองรับ และเวลาอัปเดตต้นทาง การยืนยันข้อมูลจริงต้องมี API key และสิทธิ์แพ็กเกจที่ใช้งานได้

### ดูการใช้โควตาหลังรัน worker

เมื่อใช้ API-Football terminal จะแสดง `API-Football usage` ทั้งตอนสำเร็จและเมื่อ sync ล้มเหลว:

- `requests_attempted_this_run`: จำนวนคำขอที่พยายามส่งในรอบนี้ ไม่ใช่จำนวนครั้งเปิดหน้าเว็บ
- `responses_this_run`: จำนวนคำขอที่ได้รับ HTTP response
- `daily_limit`, `daily_used`, `daily_remaining`: โควตารายวันตาม headers จาก response ล่าสุดของผู้ให้บริการ รวมการใช้คีย์นี้จากช่องทางอื่นด้วย
- `minute_limit`, `minute_remaining`: โควตารายนาที ใช้คนละค่าและคนละช่วงเวลากับรายวัน
- `by_endpoint`: จำนวนคำขอแยกตาม endpoint
- `usage_level`: low เมื่อใช้ต่ำกว่า 50%, moderate ตั้งแต่ 50%, high ตั้งแต่ 80%, exhausted เมื่อเหลือ 0, unknown เมื่อ headers ไม่ครบหรือไม่ถูกต้อง

หาก provider ไม่ส่ง headers จะเห็น `null` และ `unknown` ไม่เดาว่าเหลือครบ 100 ระบบหยุดส่งคำขอต่อเมื่อ headers บอกว่าโควตารายวันหรือรายนาทีเหลือ 0 โควตาที่ใช้ไปแล้วไม่คืนกลับเมื่อ sync ล้มเหลว รายงานแสดงใน terminal; ยังไม่มีตัวแสดงบนหน้าเว็บ

## Configuration

Browser เรียก `/api` บน hostname เดียวกับหน้าเว็บ และ Next.js proxy ไปยัง NEXT_PUBLIC_API_URL เพื่อให้ session cookie ทำงานได้แม้ Go API ใช้ 127.0.0.1

Makefile อ่าน `.env` ที่ root แล้วส่ง environment ให้ทั้ง Go และ Next.js เมื่อรัน frontend โดยตรงต้องส่ง NEXT_PUBLIC_API_URL ด้วย หากเปลี่ยน URL ให้ restart dev server หรือ build ใหม่สำหรับ production build เพื่ออัปเดต proxy configuration

| ตัวแปร | ความหมาย |
| --- | --- |
| POSTGRES_DB / POSTGRES_USER / POSTGRES_PASSWORD | การตั้งค่าฐานข้อมูล local; ค่าในตัวอย่างเป็น placeholders |
| POSTGRES_PORT | host port ของ PostgreSQL ค่าเริ่มต้นโปรเจกต์ 55432 |
| DATABASE_URL | URL ที่ Go ใช้เชื่อมต่อ; port ต้องตรงกับ POSTGRES_PORT |
| APP_TIMEZONE | เขตเวลาสำหรับ mock fixtures และค่าเริ่มต้น query: Asia/Bangkok |
| PLAY_EV_THRESHOLD | EV ขั้นต่ำสำหรับ PLAY: 0.05 |
| ODDS_MAX_AGE | อายุราคาต้นทางสูงสุด เช่น 15m |
| FOOTBALL_PROVIDER | mock หรือ api-football |
| API_FOOTBALL_KEY | API key จาก API-Football Dashboard ใช้เฉพาะ Go backend |
| API_FOOTBALL_LEAGUES | League IDs เช่น 39,140,135,78,61,5 สำหรับ 5 ลีกใหญ่ยุโรป และ UEFA Nations League (5) |
| API_FOOTBALL_MAX_REQUESTS | จำกัดคำขอต่อ worker run ค่าเริ่มต้น 50 (1–100) |
| API_PORT | port ของ Go API: 8080 |
| FRONTEND_ORIGIN | origin ที่อนุญาต เช่น http://localhost:3000 |
| OWNER_EMAIL | อีเมลบัญชีเดียวที่เข้าใช้งานได้ ต้องตั้งค่า มิฉะนั้น API ไม่เริ่มทำงาน |
| OWNER_NAME | ชื่อที่แสดงเมื่อสร้างบัญชีผ่าน make owner (ไม่จำเป็น ค่าเริ่มต้น Owner) |
| NEXT_PUBLIC_API_URL | URL ปลายทาง Go API ที่ Next.js ใช้ proxy เช่น http://127.0.0.1:8080 |

API/frontend/database bind ที่ loopback เท่านั้น CORS อนุญาต origin เดียวตาม configuration `.env` ถูก ignore และไม่ควร commit credentials จริง ส่วน `.env.example` เก็บได้

## สถาปัตยกรรม

ใช้ Modular Monolith ใน codebase เดียว แบ่งโมดูลตามหน้าที่ มี API process และ worker process ใช้ domain และ use cases เดียวกัน สื่อสารกับ frontend ผ่าน REST/JSON โดยไม่มี Redis, gRPC, message broker หรือ Kubernetes

```text
backend/
  cmd/api/                 HTTP server + graceful shutdown
  cmd/worker/              manual mock sync / settlement
  cmd/manage/              seed / migration commands
  internal/
    match/domain/          fixtures and results
    odds/domain/           market selections, implied odds, margin removal
    lineup/domain/         lineup snapshots
    prediction/domain/     distribution, EV, fair odds, settlement
    prediction/engine/     deterministic Poisson
    prediction/ports/      Engine interface
    recommendation/domain/ eligibility and EV thresholds
    userpick/domain/       independent immutable selection snapshots
    history/domain/        ROI, Brier score, timezone date bounds
    member/domain/         accounts, password policy and sessions
    application/ports/     Store, FootballProvider, AISummaryProvider, Clock, IDs
    application/usecase/   coordination, transactions, queries and decisions
    delivery/http/         thin REST handlers
  infrastructure/
    bootstrap/             composition root, explicit dependency injection
    postgres/              repository and versioned migration runner
    footballapi/           MockFootballProvider / APIFootball
    ai/                    MockAISummaryProvider
    security/              bcrypt passwords and random session tokens
  migrations/              versioned SQL (membership has no rollback)
frontend/
  app/                     Login, Today, My Picks, History, Match Detail
  components/              auth, recommendation cards and data-loading hook
  lib/                     typed API client, date groups and formatting
  e2e/                     browser flow tests
scripts/dev.sh             local process lifecycle
```

Dependency direction: HTTP/worker → application use cases → domain/ports; infrastructure implements ports. Domain ไม่ import HTTP หรือ PostgreSQL และ SQL อยู่ใน infrastructure เท่านั้น Calculation ทั้งหมดอยู่ใน Go; React แสดงค่าที่ backend ส่งมาและจัดรูปแบบตัวเลข

Repository ใช้ relational IDs, foreign keys, indexes และ JSONB สำหรับ payload ของ snapshot PostgreSQL read transactions ใช้ consistent snapshot; state transitions ใช้ transaction กับ advisory lock เพื่อกัน worker/การเลือก pick แข่งกัน Database unique index ป้องกัน active picks ซ้ำ Odds/predictions/recommendations เพิ่ม record ใหม่เสมอ ส่วน pick selection payload คงเดิม; cancellation/settlement อยู่คนละ columns

เป็นการออกแบบสำหรับ local MVP ที่มีสมาชิก: write transaction โหลด state เพื่อทำ transition ภายใน transactionเดียว ส่วน query วันที่กรอง matches และ snapshots ที่เกี่ยวข้องด้วย UTC bounds ใน SQL หากข้อมูลโตมากควรเพิ่ม repositories ที่อ่านและเขียนเฉพาะ entity แทนการโหลด state ทั้งหมด

## กฎคำนวณและประวัติ

- Poisson จำลองอิสระของ home/away ใช้ expected goals จาก fixtures และเก็บ model version
- รองรับ 1X2, Asian Handicap, Over/Under รวมทั้งเส้นเต็ม ครึ่ง และ quarter
- สูตร EV/fair odds ใช้ distribution ของชนะ/ชนะครึ่ง/คืนทุน/แพ้ครึ่ง/แพ้ ตาม PROJECT_SPEC.md
- ค่าใน UI เป็น model estimate; probability สำหรับ split outcomes หมายถึงโอกาสได้กำไร ไม่ใช่ accuracy
- PLAY: EV ≥ threshold; WATCH: 0 < EV < threshold; PASS: EV ≤ 0 หรือข้อมูลใช้ไม่ได้
- ผู้ใช้เลือก PLAY/WATCH เองเท่านั้น; POST รับ recommendation_id แล้ว backend copy snapshot ใน transaction
- ปฏิเสธ recommendation ที่ superseded, stale, PASS หรือแมตช์เริ่มแล้ว
- “ไม่เล่น” ซ่อน card ใน session ของ UI โดยไม่บันทึก decision
- DELETE ยกเลิก unsettled pick แบบ soft cancellation ประวัติยังอยู่ และ settled pick ยกเลิกไม่ได้
- เดิมพันจำลอง 1 unit; ROI ใช้ settled non-void non-cancelled picks รวม pushes ใน denominator
- Accuracy ใช้ multiclass 1X2 Brier score ล่าสุดก่อน kickoff แยกตาม model version
- เวลาจัดเก็บ UTC; Today/History กรองวันตาม IANA timezone และช่วง [เที่ยงคืน, เที่ยงคืนวันถัดไป)
- รองรับ labelled mock snapshots T-60/T-30/T-10/closing แต่ automatic scheduling และ CLV computation ยัง deferred

## REST API

| Method | Path | Response |
| --- | --- | --- |
| GET | /health | database health: 200 หรือ 503 |
| POST | /api/v1/auth/register | ปิดรับสมัคร คืน 403 REGISTRATION_CLOSED สำหรับคำขอที่ถูกต้อง |
| POST | /api/v1/auth/login | JSON email/password; บันทึก session cookie |
| GET | /api/v1/auth/me | บัญชีปัจจุบัน |
| POST | /api/v1/auth/logout | ถอน session ปัจจุบัน |
| GET | /api/v1/matches/today | page ของ match cards; optional timezone |
| GET | /api/v1/matches/:id | detail รวม historical snapshots |
| GET | /api/v1/matches/:id/odds | ราคาล่าสุดต่อ market/selection/line/bookmaker |
| GET | /api/v1/matches/:id/lineup | lineup ล่าสุด |
| GET | /api/v1/matches/:id/prediction | prediction ของ selection ที่แนะนำใน generation ล่าสุด |
| GET | /api/v1/matches/:id/recommendation | recommendation ล่าสุดพร้อม current eligibility |
| POST | /api/v1/user-picks | JSON {"recommendation_id":"UUID"}; 201 |
| GET | /api/v1/user-picks | active picks; include_cancelled=true ดูที่ยกเลิก |
| GET | /api/v1/user-picks/:id | pick snapshot พร้อม match |
| DELETE | /api/v1/user-picks/:id | soft cancellation; 204 |
| GET | /api/v1/history | recommendations/picks/results/stats; optional date, timezone |
| GET | /api/v1/history/:date | history ของวัน kickoff ใน timezone ที่ขอ |

snapshot endpoints รับ `history=true` เพื่อคืนประวัติทั้งหมด Lists ใช้ `offset=0&limit=50` (limit 1–200) คืน `{items,total,offset,limit}`; History paginate แต่ละ collection ด้วย offset/limit เดียวกัน แต่ stats คำนวณจากช่วงวันที่ทั้งหมด UI โหลดหน้าถัดไปเมื่อข้อมูลเกินขนาด page

API `/api/v1` ทุก endpoint ยกเว้น register/login ต้องส่ง session cookie; POST/DELETE ต้องมี `Origin` ตรงกับ FRONTEND_ORIGIN รวมถึงสมัครและเข้าสู่ระบบ ค่าเจ้าของ pick มาจาก session โดยตรง

Errors ใช้ `{ "error": { "code": "...", "message": "..." } }`: 400 invalid input, 401 ไม่ได้เข้าสู่ระบบหรือ session หมดอายุ, 403 origin ไม่ได้รับอนุญาต, 404 ไม่พบ entity, 409 invalid transition/duplicate/stale, 429 สมัคร/เข้าสู่ระบบเกิน 10 ครั้งต่อนาทีต่อ client IP, 500 internal

## ทดสอบและ build

```bash
make test
make verify
```

`make test` รัน Go unit tests และ frontend component tests `make verify` เพิ่ม go vet/backend build, frontend lint/typecheck/production build

Integration test ใช้ schema ชั่วคราวที่มีชื่อเฉพาะใน PostgreSQL แล้วลบเฉพาะ test schema เมื่อจบ ไม่ลบข้อมูลแอปหรือ volume:

```bash
make test-integration
```

ครอบคลุมสมัคร/เข้าสู่ระบบ, cookie, session expiry/logout, rate limit, CSRF, การแยกข้อมูลแต่ละบัญชี, migration repeatability, seed repeatability, PLAY/WATCH/PASS, concurrent duplicate selection, immutable snapshots, cancellation, settlement, stale/superseded recommendations, date queries และ transaction rollback

Browser tests ใช้ frontend ที่เปิดอยู่และ Chrome ที่ติดตั้งในเครื่อง:

```bash
make worker
cd frontend
npm run test:e2e
```

การทดสอบบัญชีเจ้าของจำลอง API เพื่อทดสอบการปิดปุ่มสมัคร/เข้าสู่ระบบ/ออกจากระบบ/มือถือ โดยไม่สร้างบัญชีจริง การทดสอบ flow และ hydration จะข้ามหากไม่ได้ตั้ง E2E_EMAIL/E2E_PASSWORD ของบัญชีทดสอบที่มีอยู่แล้ว ให้ใช้บัญชีทดสอบแยก เพราะ flow เลือก WATCH → My Picks → ยกเลิก → History → Match Detail จะสร้าง pick ที่ยกเลิกแล้วในบัญชีนั้น หากไม่มี Chrome ให้ติดตั้ง browser ผ่าน Playwright แล้วปรับ channel ใน config โดยไม่จำเป็นต้องติดตั้ง system packages ในการรันปกติ

## Migrations และการ reset

Schema อยู่ใน versioned `backend/migrations/*.up.sql` และ `*.down.sql` runner บันทึก version ใน schema_migrations ไม่ใช้ ORM auto-migrate

```bash
make migrate
make migrate-status
```

คำสั่งด้านล่างทำลายข้อมูล ใช้เฉพาะเมื่อคุณตั้งใจจะลบ:

```bash
make migrate-down CONFIRM=DROP_SCHEMA
make db-reset CONFIRM=DELETE_DATABASE
```

`migrate-down` rollback ได้เฉพาะ migration ที่มีไฟล์ down; migration 001 จะลบ application tables แต่ **migration 002 สมาชิกไม่รองรับ rollback** เพื่อรักษาบัญชี เซสชัน และเจ้าของรายการ ดังนั้นคำสั่งนี้จะไม่ rollback จาก schema รุ่นสมาชิก `db-reset` ลบ named volume ของโปรเจกต์รวมบัญชีและประวัติทั้งหมด แล้วสร้างฐานข้อมูล รัน migrations และ seed ใหม่ ไม่ถูกเรียกจาก `make dev`

## แก้ปัญหาเบื้องต้น

- ตรวจ `docker compose ps` และ `docker compose logs postgres` ควรมีสถานะ healthy
- ถ้า port ถูกใช้ เปลี่ยน POSTGRES_PORT และ port ใน DATABASE_URL ให้ตรงกัน แล้วรัน `make db-up`
- หากเปลี่ยน POSTGRES_USER/PASSWORD หลัง volume ถูกสร้าง ค่าใหม่ไม่เปลี่ยน credentials ใน volume เดิม ให้ใช้ค่าเดิมหรือเปลี่ยนผ่าน PostgreSQL โดยตั้งใจ; อย่า reset อัตโนมัติ
- ถ้า Docker permission denied ให้ตรวจสิทธิ์เข้าถึง Docker daemon ของผู้ใช้
- หาก backend บอก relation ไม่มี ให้รัน `make migrate` ก่อน `make seed`
- หาก browser แจ้ง origin denied ให้เปิด http://localhost:3000 ให้ตรง FRONTEND_ORIGIN หรือแก้ configuration แล้ว restart API
- หากราคาเก่าจนเลือกไม่ได้ ใช้ปุ่มซิงก์ข้อมูล หรือรัน `make worker` แล้วกดโหลดข้อมูลใหม่ การโหลดหน้าอย่างเดียวไม่สร้างราคาใหม่
- หากพบ hydration error ให้ดู attribute ที่ต่างกันในรายละเอียด error แล้วลองโปรไฟล์ Chrome ที่ไม่มีส่วนขยาย หากหาย ให้ปิดส่วนขยายที่แก้ HTML ก่อน React โหลด; ไม่ควรซ่อน error โดยไม่ตรวจสาเหตุ
- หาก sandbox ของเครื่องมือปิดกั้น localhost หรือ subprocess ของ build ให้รันคำสั่งเดียวกันใน terminal ปกติ

## เปลี่ยน mock providers ภายหลัง

Implement `FootballProvider.Fetch` ใน infrastructure ตัวใหม่แล้วเปลี่ยน composition root ที่ `bootstrap.Open` เก็บ provider_name/external_id แยกจาก internal UUID และ map ID ให้คงที่ รักษา timestamps และ source keys เพื่อ sync ซ้ำได้อย่างปลอดภัย Domain/use cases ไม่ควรรู้จักชื่อ provider หรือเรียก SDK โดยตรง

Implement `AISummaryProvider.Reasons` สำหรับคำอธิบายภาษาไทย แล้ว inject แทน MockAISummaryProvider เมื่อเพิ่ม provider จริง ควรส่ง structured evidence ที่จำเป็นให้ interface และมี fallback คำอธิบาย deterministic AI ไม่มีสิทธิ์คำนวณ probability หรือเป็น source of truth ของ EV

## งานที่ยังไม่อยู่ในรุ่นนี้

- AI provider จริง และโมเดลขั้นสูง เช่น measured xG, team ratings และ lineup adjustments
- การคำนวณ CLV
- การกำหนดทุนเป็นบาท และการเชื่อมต่อส่งเดิมพัน
- Google sign-in, ยืนยันอีเมล และกู้รหัสผ่าน
- การเปิดใช้งานสาธารณะ ซึ่งต้องเพิ่ม TLS และการเตรียมระบบสำหรับ deployment

รายละเอียดกฎธุรกิจและขอบเขตงานอยู่ใน [PROJECT_SPEC.md](PROJECT_SPEC.md) แนวทางสำหรับผู้พัฒนาและ agent อยู่ใน [AGENTS.md](AGENTS.md)

## ซิงก์จากหน้าเว็บและตามเวลา

หลังอัปเดตโค้ดให้รัน `make migrate` แล้ว restart API หน้า “วันนี้” มีปุ่ม **ซิงก์ข้อมูล** สำหรับเจ้าของ และแสดงผลสำเร็จล่าสุด/เวลาที่กดซ้ำได้ ปุ่ม **โหลดข้อมูลใหม่** ยังคงอ่านฐานข้อมูลเท่านั้น

```dotenv
SYNC_AUTO_ENABLED=true
SYNC_INTERVAL=24h
SYNC_COOLDOWN=30m
```

ตัวตั้งเวลาอยู่ใน Go API ไม่ต้องรัน worker อีกตัวตลอดเวลา API ต้องทำงานต่อเนื่องบน VPS เช่นผ่าน systemd การรัน API ครั้งแรกจะรอหนึ่งช่วงเวลา; ถ้าเคยซิงก์แล้วจะใช้เวลาครั้งล่าสุดที่เก็บในฐานข้อมูล ตรวจรอบที่ถึงกำหนดทุกนาที เปลี่ยน .env แล้ว restart API

วันละครั้งเป็นค่าเริ่มต้นเพื่อคุมโควตา ไม่เพียงพอสำหรับราคาที่ต้องสดภายใน 15 นาที หากเพิ่มความถี่ต้องประเมินแพ็กเกจ API และจำนวน requests ต่อรอบ ช่วงเวลาอย่างน้อย 15m; cooldown อย่างน้อย 1m ไม่มีการ retry อัตโนมัติทันทีเมื่อผิดพลาด

`make worker` ยังใช้ได้ แต่ร่วมล็อกกับปุ่มและตัวตั้งเวลา ป้องกันงานซ้อนข้าม process และใช้ cooldown เดียวกัน (409 เมื่อกำลังรัน, 429 เมื่อยังอยู่ในช่วงพัก) ทั้งรอบที่สำเร็จและล้มเหลวใช้โควตาได้ Cooldown ช่วยลดการกดซ้ำ แต่ไม่รับประกันว่าไม่เกินโควตารายวัน ให้ตรวจยอดที่ dashboard ผู้ให้บริการ

ระหว่างซิงก์เว็บแสดงข้อมูลเดิม หาก upstream ล้มเหลว/บันทึกไม่สำเร็จ ข้อมูลเดิมและประวัติไม่ถูกล้าง เมื่อสำเร็จหน้า Today โหลดค่าล่าสุดโดยอัตโนมัติ การเปลี่ยนวันหรือเลื่อน kickoff อาจทำให้คู่ย้ายออกจาก Today แต่ประวัติยังอยู่ API Free ที่ไม่รองรับฤดูกาลปัจจุบันยังคงซิงก์ไม่สำเร็จ การเพิ่มปุ่มไม่ได้ปลดข้อจำกัดแพ็กเกจ

สถานะเก็บใน PostgreSQL; restart แล้วยังคง cooldown และผลสำเร็จล่าสุด หาก process หยุดกลางงาน สถานะจะแจ้งว่าขัดจังหวะหลังหมดเวลา 2 นาที บันทึกข้อมูลฟุตบอลกับสถานะเป็นคนละ transaction; ถ้า process หยุดหลัง commit แต่ก่อนบันทึกสถานะ ข้อมูลอาจอัปเดตแล้วแม้สถานะยังไม่ยืนยัน ให้ใช้โหลดข้อมูลใหม่ได้

## รันทั้งแอปด้วย Docker

ใช้ Docker Engine และ Compose plugin บน Linux ได้ ไม่ต้องติดตั้ง Go, Node.js หรือ PostgreSQL บน host ไฟล์ `docker-compose.yaml` รัน PostgreSQL → migration → Go API → Next.js ตามลำดับ พร้อม healthchecks และ restart policy ส่วน `compose.yaml` เดิมยังเป็นฐานข้อมูลสำหรับ `make dev` ให้ระบุ `-f docker-compose.yaml` เสมอเมื่อรันทั้งแอป

```bash
# ทำครั้งแรกเท่านั้น ไม่ copy ทับไฟล์ที่ตั้งค่าแล้ว
cp -n .env.docker.example .env.docker
chmod 600 .env.docker
# แก้ OWNER_EMAIL, POSTGRES_PASSWORD และ DOCKER_DATABASE_URL ให้ตรงกัน
docker compose --env-file .env.docker -f docker-compose.yaml up -d --build
docker compose --env-file .env.docker -f docker-compose.yaml ps
```

`DOCKER_DATABASE_URL` ต้องใช้ host `postgres` และ port `5432` ไม่ใช้ localhost หากรหัสผ่านมีอักขระพิเศษ ให้ URL-encode เฉพาะส่วนรหัสผ่านใน URL; ค่า POSTGRES_PASSWORD ยังใช้รหัสจริง เปลี่ยน provider/key ใน `.env.docker` หากใช้ข้อมูลจริง ไม่ส่ง `.env.docker` เข้า Git หรือ Docker image

เว็บเปิดที่ **http://localhost:3000** บนเครื่องที่รัน Docker เผยแพร่เฉพาะพอร์ตเว็บบน loopback; API และ PostgreSQL ไม่มีพอร์ตเปิดบน host บน VPS จึงใช้ต่อกับ reverse proxy และ HTTPS หรือ SSH tunnel ได้ การเพิ่มไฟล์นี้ยังไม่ได้ติดตั้ง reverse proxy/ใบรับรองหรือเปิดเว็บสู่สาธารณะ หากเปลี่ยน WEB_PORT ให้ FRONTEND_ORIGIN ตรงกับ URL ที่ browser ใช้

สร้างเจ้าของสำหรับฐานข้อมูลใหม่ โดยรับรหัสผ่านจาก stdin ไม่ใส่รหัสใน command history:

```bash
read -r -s -p 'Owner password: ' owner_password
printf '\n'
printf '%s' "$owner_password" | docker compose --env-file .env.docker -f docker-compose.yaml run --rm -T --no-deps api /app/manage create-owner
unset owner_password
```

คำสั่งนี้ไม่เปลี่ยนรหัสของบัญชีที่มีอยู่แล้ว ให้ใช้ OWNER_EMAIL ตรงกับบัญชีเดิม หากใช้ mock และต้องการข้อมูลตัวอย่าง:

```bash
docker compose --env-file .env.docker -f docker-compose.yaml run --rm --no-deps api /app/manage seed
```

ระบบไม่ seed หรือเรียก API ฟุตบอลระหว่าง build/startup ตัวตั้งเวลาทำงานใน API ทุก 24h ตามค่าเริ่มต้น ไม่ต้องเปิด worker ค้างไว้ สามารถซิงก์ผ่านปุ่มเจ้าของ หรือรัน worker ครั้งเดียว:

```bash
docker compose --env-file .env.docker -f docker-compose.yaml run --rm worker
docker compose --env-file .env.docker -f docker-compose.yaml logs --tail=100 -f api frontend
```

ทั้งปุ่ม ตัวตั้งเวลาและ worker ร่วมล็อก/cooldown เดียวกัน หลังแก้โค้ดให้รัน `up -d --build` ใหม่; หลังแก้ค่าตัวแปรให้รัน `up -d` เพื่อสร้าง container ใหม่ การเปลี่ยน URL ของ API ภายในต้อง rebuild frontend เพราะ Next.js rewrites ถูกสร้างตอน build

ข้อมูลอยู่ใน named volume ของ Compose project `football-analyzer` แยกจากฐานข้อมูลเดิมของ `make dev` ไม่ย้ายข้อมูลเดิมให้อัตโนมัติ หากต้องการบัญชีและประวัติเดิมบน VPS ต้อง backup/restore PostgreSQL ก่อนใช้ระบบจริง เปลี่ยนรหัสใน env ไม่ได้เปลี่ยนรหัสใน volume ที่มีอยู่แล้ว

หยุดระบบโดยเก็บข้อมูลไว้:

```bash
docker compose --env-file .env.docker -f docker-compose.yaml down
```

อย่าเพิ่ม `--volumes` หรือ `-v` หากต้องการเก็บบัญชีและประวัติ การตั้งค่า HTTPS, Secure cookies หลัง reverse proxy และการสำรองข้อมูลต้องเตรียมก่อนเปิดใช้งานสาธารณะ
