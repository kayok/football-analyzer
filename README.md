# สนาม · Football Analyzer

เว็บวิเคราะห์ฟุตบอลส่วนตัวภาษาไทย ใช้ Go, Next.js และ PostgreSQL จริง ข้อมูลฟุตบอล ราคา รายชื่อผู้เล่น และคำอธิบายทั้งหมดเป็น mock สำหรับทดสอบ flow:

**Today → Recommendation → เลือกเล่น → My Picks → History**

รุ่นนี้ใช้บน localhost สำหรับผู้ใช้คนเดียว ยังไม่มีระบบ login และไม่พร้อมเปิดสู่เครือข่ายสาธารณะ โมเดล Poisson ใช้ expected goals จำลอง จึงไม่ได้แสดงประสิทธิภาพการทำนายการแข่งขันจริง

## เริ่มใช้งาน

ต้องมี Go 1.26+, Node.js 24 LTS, npm, Docker และ Docker Compose ไม่ต้องติดตั้ง PostgreSQL ลงเครื่อง

```bash
nvm install
nvm use
cp .env.example .env
cd frontend
npm ci
cd ..
make dev
```

หากมี `.env` อยู่แล้ว ให้ใช้ไฟล์เดิม ไม่ต้อง copy ทับ `make dev` เริ่ม PostgreSQL, รัน migrations, เพิ่มข้อมูลจำลอง แล้วเปิด API และ frontend ไม่ลบฐานข้อมูลหรือประวัติเดิม กด Ctrl+C เพื่อหยุด API/frontend; PostgreSQL และ volume ยังอยู่

เปิด **http://localhost:3000** (ใช้ hostname นี้ให้ตรงกับ FRONTEND_ORIGIN)

- API: http://127.0.0.1:8080
- PostgreSQL: 127.0.0.1:55432
- `make db-down` หยุด container โดยเก็บ volume ไว้

ติดตั้ง Go dependencies ครั้งแรกด้วย `cd backend && go mod download` หรือให้คำสั่ง `go run` ดาวน์โหลดให้อัตโนมัติ

## คำสั่งแยกแต่ละส่วน

```bash
make db-up
make migrate
make migrate-status
make seed
make backend       # terminal 1
make frontend      # terminal 2
make worker        # terminal 3: เพิ่ม snapshot ราคา/lineup/prediction/recommendation
```

seed มี 3 คู่วันนี้: PLAY, WATCH, PASS และ 2 คู่ที่จบแล้วพร้อม picks ตัวอย่างสำหรับ History ภายในวันเดียวกัน seed ซ้ำไม่สร้าง fixture หรือ source snapshot ซ้ำ แต่เพิ่ม prediction/recommendation รุ่นใหม่ของคู่ที่ยังไม่จบ และไม่แก้ picks เดิม Fixtures วันนี้ขึ้นกับ clock และ APP_TIMEZONE; ตัวอย่าง seed picks ใช้ flag `demo` และแสดงคำว่า “ตัวอย่าง” ใน UI

ราคามีอายุ 15 นาที: เมื่อหมดอายุให้รัน `make worker` แล้วกด “โหลดข้อมูลใหม่” แอปไม่ sync อัตโนมัติ การแสดงผลวันนี้ตรวจความพร้อมปัจจุบันอีกครั้ง แต่ประวัติแสดง recommendation ที่บันทึกจริงในแต่ละ generation

## จำลองผลการแข่งขันของ pick ที่เลือกเอง

คัดลอก match UUID จาก URL หน้ารายละเอียด หรือ `GET /api/v1/matches/today` แล้วรัน:

```bash
make worker ARGS='--finish-match MATCH_UUID --home 2 --away 0'
```

คำสั่งนี้ใช้กับ fixture ของ mock provider เท่านั้น บันทึกผลจำลองและ settle picks ที่ยังไม่ได้สรุปผลใน transaction หลังจากนั้นเปิด My Picks/History ดูผลและ ROI Worker หรือ seed ครั้งต่อไปจะไม่ย้อนสถานะคู่ที่จบแล้วกลับไปเป็น scheduled ไม่มีการเชื่อมต่อ provider จริง

## Configuration

Makefile อ่าน `.env` ที่ root แล้วส่ง environment ให้ทั้ง Go และ Next.js เมื่อรัน frontend โดยตรงต้องส่ง NEXT_PUBLIC_API_URL ด้วย ค่าตัวแปรที่ขึ้นต้น NEXT_PUBLIC ถูกฝังใน frontend ตอน build; ต้อง build ใหม่เมื่อเปลี่ยน URL

| ตัวแปร | ความหมาย |
| --- | --- |
| POSTGRES_DB / POSTGRES_USER / POSTGRES_PASSWORD | การตั้งค่าฐานข้อมูล local; ค่าในตัวอย่างเป็น placeholders |
| POSTGRES_PORT | host port ของ PostgreSQL ค่าเริ่มต้นโปรเจกต์ 55432 |
| DATABASE_URL | URL ที่ Go ใช้เชื่อมต่อ; port ต้องตรงกับ POSTGRES_PORT |
| APP_TIMEZONE | เขตเวลาสำหรับ mock fixtures และค่าเริ่มต้น query: Asia/Bangkok |
| PLAY_EV_THRESHOLD | EV ขั้นต่ำสำหรับ PLAY: 0.05 |
| ODDS_MAX_AGE | อายุราคา เช่น 15m |
| API_PORT | port ของ Go API: 8080 |
| FRONTEND_ORIGIN | origin ที่อนุญาต เช่น http://localhost:3000 |
| NEXT_PUBLIC_API_URL | URL ของ backend ที่ browser เข้าถึงได้ |

API/frontend/database bind ที่ loopback เท่านั้น CORS อนุญาต origin เดียวตาม configuration `.env` ถูก ignore และไม่ควร commit credentials จริง ส่วน `.env.example` เก็บได้

## สถาปัตยกรรม

ใช้ Modular Monolith ใน codebase เดียว มี API process และ worker process ใช้ domain และ use cases เดียวกัน ไม่ใช่ microservices ไม่มี Redis, message broker หรือ Kubernetes

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
    application/ports/     Store, FootballProvider, AISummaryProvider, Clock, IDs
    application/usecase/   coordination, transactions, queries and decisions
    delivery/http/         thin REST handlers
  infrastructure/
    bootstrap/             composition root, explicit dependency injection
    postgres/              repository and versioned migration runner
    footballapi/           MockFootballProvider
    ai/                    MockAISummaryProvider
  migrations/              up/down SQL
frontend/
  app/                     Today, My Picks, History, Match Detail
  components/              UI and data-loading hook
  lib/                     typed API client and display formatting
  e2e/                     browser flow tests
scripts/dev.sh             local process lifecycle
```

Dependency direction: HTTP/worker → application use cases → domain/ports; infrastructure implements ports. Domain ไม่ import HTTP หรือ PostgreSQL และ SQL อยู่ใน infrastructure เท่านั้น Calculation ทั้งหมดอยู่ใน Go; React แสดงค่าที่ backend ส่งมาและจัดรูปแบบตัวเลข

Repository ใช้ relational IDs, foreign keys, indexes และ JSONB สำหรับ payload ของ snapshot PostgreSQL read transactions ใช้ consistent snapshot; state transitions ใช้ transaction กับ advisory lock เพื่อกัน worker/การเลือก pick แข่งกัน Database unique index ป้องกัน active picks ซ้ำ Odds/predictions/recommendations เพิ่ม record ใหม่เสมอ ส่วน pick selection payload คงเดิม; cancellation/settlement อยู่คนละ columns

เป็นการออกแบบสำหรับ local single-user MVP: write transaction โหลด state เพื่อทำ transition ภายใน transactionเดียว ส่วน query วันที่กรอง matches และ snapshots ที่เกี่ยวข้องด้วย UTC bounds ใน SQL หากข้อมูลโตมากควรเพิ่ม repositories ที่อ่านและเขียนเฉพาะ entity แทนการโหลด state ทั้งหมด

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

Errors ใช้ `{ "error": { "code": "...", "message": "..." } }`: 400 invalid input, 404 ไม่พบ entity, 409 invalid transition/duplicate/stale, 500 internal การเรียกจาก browser origin อื่นคืน 403

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

ครอบคลุม migration repeatability, seed repeatability, PLAY/WATCH/PASS, concurrent duplicate selection, immutable snapshots, cancellation, settlement, stale/superseded recommendations, date queries และ transaction rollback

Browser tests ใช้ API/frontend ที่เปิดอยู่และ Chrome ที่ติดตั้งในเครื่อง:

```bash
make worker
cd frontend
npm run test:e2e
```

ทดสอบเลือก WATCH → My Picks → ยกเลิก → History → Match Detail และ mobile layout พร้อมตรวจ PASS ที่ไม่มีปุ่มเลือก Tests จะสร้าง pick ที่ยกเลิกแล้วในข้อมูล demo local หากไม่มี Chrome ให้ติดตั้ง browser ผ่าน Playwright แล้วปรับ channel ใน config โดยไม่จำเป็นต้องติดตั้ง system packages ในการรันปกติ

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

`migrate-down` rollback migration ล่าสุด (migration แรกจะลบ application tables) `db-reset` ลบ named volume ของโปรเจกต์และสร้างใหม่ ไม่ถูกเรียกจาก `make dev`

## PostgreSQL troubleshooting

- ตรวจ `docker compose ps` และ `docker compose logs postgres` ควรมีสถานะ healthy
- ถ้า port ถูกใช้ เปลี่ยน POSTGRES_PORT และ port ใน DATABASE_URL ให้ตรงกัน แล้วรัน `make db-up`
- หากเปลี่ยน POSTGRES_USER/PASSWORD หลัง volume ถูกสร้าง ค่าใหม่ไม่เปลี่ยน credentials ใน volume เดิม ให้ใช้ค่าเดิมหรือเปลี่ยนผ่าน PostgreSQL โดยตั้งใจ; อย่า reset อัตโนมัติ
- ถ้า Docker permission denied ให้ตรวจสิทธิ์เข้าถึง Docker daemon ของผู้ใช้
- หาก backend บอก relation ไม่มี ให้รัน `make migrate` ก่อน `make seed`
- หาก browser แจ้ง origin denied ให้เปิด http://localhost:3000 ให้ตรง FRONTEND_ORIGIN หรือแก้ configuration แล้ว restart API
- หาก sandbox ของเครื่องมือปิดกั้น localhost หรือ subprocess ของ build ให้รันคำสั่งเดียวกันใน terminal ปกติ

## เปลี่ยน mock providers ภายหลัง

Implement `FootballProvider.Fetch` ใน infrastructure ตัวใหม่แล้วเปลี่ยน composition root ที่ `bootstrap.Open` เก็บ provider_name/external_id แยกจาก internal UUID และ map ID ให้คงที่ รักษา timestamps และ source keys เพื่อ sync ซ้ำได้อย่างปลอดภัย Domain/use cases ไม่ควรรู้จักชื่อ provider หรือเรียก SDK โดยตรง

Implement `AISummaryProvider.Reasons` สำหรับคำอธิบายภาษาไทย แล้ว inject แทน MockAISummaryProvider เมื่อเพิ่ม provider จริง ควรส่ง structured evidence ที่จำเป็นให้ interface และมี fallback คำอธิบาย deterministic AI ไม่มีสิทธิ์คำนวณ probability หรือเป็น source of truth ของ EV

งานที่ยังไม่อยู่ใน MVP: real football/AI providers, authentication/authorization/multi-user deployment, learned xG/team ratings/lineup adjustments, automatic timed scheduler และ CLV ไม่มีการติดตั้ง PostgreSQL ลง host และไม่มีการส่งข้อมูลไป provider จริง
