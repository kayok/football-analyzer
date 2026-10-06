# Football Analyzer — Project Specification

Build a private football analysis web application as a local MVP with maintainable architecture and verified end-to-end behavior.

The application runs on localhost with email/password membership. Each account owns its picks and personal performance history. Public deployment, email verification, password recovery, and third-party sign-in are deferred.

## Tech Stack

- Backend: Go
- Frontend: Next.js + TypeScript
- Database: PostgreSQL
- Local infrastructure: Docker Compose
- API style: REST
- Architecture: Modular Monolith + Clean Architecture

Important:
- Do not build microservices in this version.
- Do not install system packages with apt unless absolutely necessary.
- Prefer Dockerized infrastructure for local dependencies.
- Do not install PostgreSQL directly on the host machine.
- Do not use real football APIs or real AI providers in the first implementation.
- Use mock providers first.

---

# 1. Product Goal

The application is a private football analysis dashboard.

The system analyzes football matches and produces recommendations, but the user decides manually whether to select a pick.

The system must support:

- today's matches
- football odds
- starting lineups
- injuries and suspensions
- probability calculation
- fair odds
- expected value (EV)
- recommendations
- manually selected user picks
- historical records
- match results
- model versioning

The system should eventually support both club football and international football.

The main application UI must be in Thai.

---

# 2. Core User Experience

The Today page must be very easy to scan quickly.

Example UI:

คู่แข่งขัน:
Liverpool vs Arsenal

เล่นอะไร:
Liverpool -0.5

ราคา:
1.95

โอกาสชนะ:
57%

ความคุ้มค่า EV:
+11.2%

สถานะ:
PLAY

Actions:

[ เลือกเล่น ]
[ ไม่เล่น ]
[ รายละเอียด ]

For a match without enough value:

Chelsea vs Tottenham

PASS

ยังไม่มีราคาที่คุ้ม

[ รายละเอียด ]

The four most important values are:

1. Selection
2. Odds
3. Model-estimated probability (label according to Section 6)
4. EV

The UI labels should be displayed in Thai.

---

# 3. Recommendation vs User Pick

This is a critical business rule.

A system recommendation and a user-selected pick must be separate entities.

The system generates:

- PLAY
- WATCH
- PASS

The user manually decides whether to select a recommendation.

Never automatically create a user pick from a recommendation.

When the user selects a pick, store a snapshot of:

- recommendation_id
- match_id
- market
- selection
- odds_at_pick
- probability_at_pick
- settlement_probabilities_at_pick
- ev_at_pick
- line (nullable for 1X2)
- bookmaker
- model_version
- stake_units (fixed at 1 for the MVP)
- picked_at

A user pick must never change automatically when future odds or predictions change.

Pick rules:

- The backend copies values from the stored recommendation and its referenced prediction/odds snapshots in one transaction. Do not trust odds, probability, or EV supplied by the frontend.
- Allow selection of PLAY or WATCH only, before kickoff, using the current recommendation and a valid odds snapshot within oddsMaxAge (default 15 minutes, as defined in Section 6). Inject a clock for deterministic checks/tests.
- Reject stale, superseded, PASS, or already-started recommendations with a clear error; require the user to review the current recommendation before retrying.
- Allow only one active pick per account, match, market, selection, and line. Repeated selection must not create duplicate active picks.
- The action `ไม่เล่น` dismisses the card for the current UI session only; it does not create a pick or change the recommendation.
- DELETE cancels an unsettled pick by setting `cancelled_at`; never physically delete its snapshot. Repeated cancellation is idempotent. Settled picks cannot be cancelled.
- My Picks shows active picks by default. History retains cancelled picks with a cancellation label and excludes them from performance statistics.

---

# 4. Backend Architecture

Use Modular Monolith + Clean Architecture.

Suggested modules:

- match
- odds
- lineup
- prediction
- recommendation
- userpick
- history

Suggested backend structure:

backend/
  cmd/
    api/
      main.go
    worker/
      main.go

  internal/
    match/
      domain/
      usecase/
      ports/
      delivery/

    odds/
      domain/
      usecase/
      ports/

    lineup/
      domain/
      usecase/
      ports/

    prediction/
      domain/
      usecase/
      ports/
      engine/

    recommendation/
      domain/
      usecase/
      ports/

    userpick/
      domain/
      usecase/
      ports/

    history/
      domain/
      usecase/
      ports/

  infrastructure/
    postgres/
    footballapi/
    ai/
    scheduler/

  migrations/

Rules:

- Domain must not depend on PostgreSQL, HTTP frameworks, API-Football, Sportmonks, Gemini, or other infrastructure.
- Business logic belongs in domain/usecase layers.
- HTTP handlers must remain thin.
- SQL must not exist in domain code.
- External services must be behind interfaces/ports.
- Use explicit dependency injection.
- Avoid global state.
- Use context.Context appropriately.
- Support graceful shutdown.
- Use structured logging.

---

# 5. Core Modules

## Match

Store:

- fixture
- competition / league
- home team
- away team
- kickoff time
- match status

## Odds

Support:

- 1X2
- Asian Handicap
- Over/Under
- bookmaker
- odds snapshots
- captured_at
- odds movement

## Lineup

Support:

- starting XI
- substitutes
- injuries
- suspensions
- lineup snapshots

## Prediction

Support:

- probability
- expected goals
- fair odds
- EV
- market implied probability
- Poisson model
- model versioning

The architecture should be ready for future support of:

- xG
- xGA
- home/away strength
- team rating
- lineup adjustment
- odds movement
- advanced bookmaker margin removal beyond the MVP proportional 1X2 method

Do not implement machine learning yet.

## Recommendation

Support:

- PLAY
- WATCH
- PASS
- recommended market
- recommended selection
- minimum acceptable odds
- probability
- EV
- reasons
- model_version
- generated_at

## User Pick

A manually selected user decision.

Must be stored independently from system recommendations.

## History

Support:

- historical recommendations
- user picks
- match results
- win/half-win/push/half-loss/loss/void
- ROI
- EV history
- model accuracy
- closing odds
- future CLV support

MVP settlement and statistics:

- Settle pre-match markets using the final score after regulation time plus stoppage time; exclude extra time and penalties.
- Each pick uses a fixed simulated stake of 1 unit. Store its settlement result, net profit in units, and settled_at separately from its immutable selection snapshot.
- Net profit per unit is odds_at_pick - 1 for win, (odds_at_pick - 1) / 2 for half-win, 0 for push/void, -0.5 for half-loss, and -1 for loss.
- ROI = total net profit / total stake of settled, non-cancelled, non-void picks. Include pushes in the denominator. Return null when that denominator is zero.
- Pending picks are excluded from realized profit and ROI. Cancelled or abandoned matches are void in the mock MVP; postponed matches remain pending until a final result or explicit cancellation.
- Report model accuracy as the multiclass Brier score for a complete 1X2 distribution, using the latest pre-kickoff prediction per completed match and model version. For each match, sum (predicted_probability - observed_indicator)^2 across home/draw/away, then average these sums across eligible matches (no division by 3). Lower is better; return null when no eligible results exist. Do not treat pushes or split settlements as binary wins/losses for accuracy.
- Closing odds and CLV are deferred statistics; the MVP preserves a labelled mock closing snapshot for later use.

---

# 6. Calculation Rules

Use decimal odds for the MVP.

For markets with only full win and full loss outcomes:

EV = probability * odds - 1

Example:

probability = 0.56
odds = 2.00

EV = 0.56 * 2.00 - 1
EV = 0.12
EV = +12%

Fair odds:

fairOdds = 1 / probability

For markets that allow pushes or split settlement, store the complete outcome distribution:

- p_win
- p_half_win
- p_push
- p_half_loss
- p_loss

All five probabilities must be finite, between 0 and 1, and sum to 1 within a documented numeric tolerance (1e-9). Decimal odds must be finite and greater than 1.

Expected net profit per unit:

EV = p_win * (odds - 1) + p_half_win * (odds - 1) / 2 - p_half_loss / 2 - p_loss

Define A = p_win + p_half_win / 2 and B = p_loss + p_half_loss / 2.

- EV = A * (odds - 1) - B
- fairOdds = 1 + B / A
- minimumAcceptableOdds = 1 + (B + playEVThreshold) / A

If A = 0, fair odds and minimum acceptable odds are unavailable (null), and the selection cannot be PLAY. Never serialize infinity or NaN. These formulas reduce to the binary formulas above when there are no pushes or half outcomes.

Asian Handicap settlement uses the selected team's regulation goal difference plus its signed handicap. Over/Under compares regulation total goals against its line. Whole and half lines settle directly; quarter lines split the stake equally between the adjacent whole and half lines (for example, -0.25 splits into 0 and -0.5, and Over 2.25 splits into Over 2 and Over 2.5). Use the same settlement rules for prediction distributions and final pick results.

Store handicap/total lines explicitly as numeric values in increments of 0.25; do not infer them from display text. All MVP markets are pre-match, full-regulation-time markets.

For binary markets, the UI probability is p_win, labelled `โอกาสชนะ`. For markets with pushes or split settlement it is p_win + p_half_win, labelled `โอกาสได้กำไร`; the detail page must show the full distribution. This number is a model estimate, not model accuracy or certainty, and must not replace the full distribution in EV calculations.

Use unrounded values for calculations and status decisions. Round only for display; round a displayed minimum acceptable odds upward to two decimal places.

Example with a push: p_win = 0.50, p_push = 0.10, p_loss = 0.40, odds = 2.00 gives EV = 0.10 and fairOdds = 1.80.

Recommendation rules (MVP defaults, configurable in the backend):

- playEVThreshold = 0.05; oddsMaxAge = 15 minutes.
- PLAY: valid complete inputs, match not started, odds age <= oddsMaxAge, and EV >= playEVThreshold.
- WATCH: the same eligibility checks, with 0 < EV < playEVThreshold.
- PASS: EV <= 0, or unavailable/invalid inputs, stale odds, or a started match. Store a reason code and Thai explanation; unavailable metrics must be null rather than fabricated zeros.
- Missing lineups alone do not invalidate the MVP model because lineup adjustment is deferred. Missing expected-goals inputs or the relevant odds do invalidate it.
- Evaluate available selections and emit one recommendation per match per generation. Choose the highest EV among eligible selections; break exact ties deterministically by market, selection, line, and bookmaker.
- PASS may retain the best evaluated selection for inspection, but must never be selectable. When no selection is evaluable, selection and its metrics are null.
- Persist odds_snapshot_id, prediction_id, the applied thresholds, and rule_version on each recommendation so its decision can be reproduced.

All probability, fair odds, and EV calculations must happen in the Go backend.

The frontend must never contain core probability or EV business logic.

---

# 7. Prediction Engine

Create a prediction engine interface.

Required in the MVP:

- A deterministic independent Poisson model using explicit expected_goals_home and expected_goals_away from mock fixtures; both must be finite and positive. Do not derive these inputs from the same offered odds being evaluated.
- Persist the expected-goals inputs and model version on each prediction. These mock inputs demonstrate the flow and do not establish real-world predictive performance.
- Derive 1X2, Asian Handicap, and Over/Under settlement distributions from the joint score distribution using Section 6.
- Truncate each Poisson distribution adaptively until omitted joint probability mass is <= 1e-9, then normalize the retained joint distribution. Test normalization and truncation behavior.
- Display raw market implied probability as 1 / decimal odds. For complete 1X2 quotes from the same bookmaker and snapshot, also display normalized probabilities after proportional margin removal: q_i = (1 / odds_i) / sum_j(1 / odds_j), across home/draw/away. Do not use 1 / odds as an unconditional win probability for push/split-settlement markets.

Deferred beyond the MVP:

- Learned or real-data expected-goals estimation, xG/xGA, team ratings, home/away strength estimation, lineup adjustments, and odds-movement adjustments.
- Margin-removal models for Asian Handicap and Over/Under, automated provider sync, automatic timed scheduling, and CLV computation.

Keep extension points simple; do not create implementations or extra infrastructure for deferred features.

Do not add machine learning in the MVP.

---

# 8. Football Data Provider

Create a FootballProvider interface.

Initial implementation:

MockFootballProvider

Future providers may include:

- API-Football
- Sportmonks
- other providers

Domain and use cases must not know provider names directly.

External provider IDs must not be used as internal primary keys.

Store:

- provider_name
- external_id

Use internal UUIDs for application entities.

---

# 9. AI Provider

Create:

AISummaryProvider interface

Initial implementation:

MockAISummaryProvider

Future use:

- summarize recommendation reasons in Thai
- explain lineup impact in Thai
- explain why the model prefers a specific market

AI must not calculate the primary probability.

AI must not be the source of truth for EV.

---

# 10. Worker

Create a separate worker process:

cmd/worker

The worker should eventually:

- sync matches
- sync odds
- sync lineups
- generate predictions
- generate recommendations
- save snapshots
- update results

Prepare scheduling support for:

- T-60 minutes
- T-30 minutes
- T-10 minutes
- closing snapshot

The MVP must provide a manually invoked worker that reads mock fixtures/odds/lineups/results, appends prediction and recommendation snapshots, and settles eligible picks. Automatic scheduling is deferred.

Manual/seed execution must be repeatable without duplicating the same source snapshot or settling a pick twice. Fresh predictions/recommendations may be appended for a new generation; never overwrite an earlier generation. Label actual capture times and snapshot phases honestly; do not fabricate historical capture times during a current worker run.

API and worker must remain in the same codebase.

Do not turn them into microservices.

---

# 11. PostgreSQL

Use PostgreSQL through Docker Compose.

Do not install PostgreSQL directly on Ubuntu.

Use:

- persistent named volume
- healthcheck
- exposed port bound to 127.0.0.1
- environment variables

Example development values:

POSTGRES_DB=football
POSTGRES_USER=football
POSTGRES_PASSWORD=football_dev

Example:

DATABASE_URL=postgres://football:football_dev@localhost:5432/football?sslmode=disable

Requirements:

- provide .env.example
- never commit real .env files
- never hardcode secrets

---

# 12. Database Migrations

Use versioned migrations.

Do not rely on auto-migrate as the primary schema management mechanism.

Support:

- migrate up
- migrate down if appropriate
- migration status

All schema changes must be represented by migration files.

---

# 13. Database Tables

Initial tables:

- competitions
- teams
- matches
- odds_snapshots
- lineup_snapshots
- predictions
- recommendations
- user_picks
- match_results

Use UUID for internal primary keys.

Store external provider IDs separately.

Add appropriate:

- foreign keys
- indexes
- unique constraints

---

# 14. Historical Snapshots

The system must preserve historical state.

Do not overwrite historical odds, predictions, or recommendations.

Support phase labels and seed labelled mock examples for:

- T-60
- T-30
- T-10
- closing

The application should eventually be able to answer:

- what were the odds at T-60?
- what was the lineup at T-10?
- what probability did the model produce?
- what recommendation did the system make?
- what were the closing odds?

When a new prediction is generated, create a new record.

When a new recommendation is generated, create a new record.

Do not mutate historical records unnecessarily.

Automatic T-60/T-30/T-10/closing capture is deferred. Preserve input snapshot links and actual capture/generated times for manual generations. A closing snapshot means the last available pre-kickoff odds for the same market, selection, line, and bookmaker.

---

# 15. Model Versioning

Every prediction and recommendation must store:

- model_version
- generated_at

Examples:

- v1-market-poisson
- v2-lineup-adjustment

This must allow later comparison between model versions.

---

# 16. Time Handling

Store timestamps in PostgreSQL as UTC.

Use timestamptz when appropriate.

The frontend converts timestamps to the user's local timezone.

Avoid ambiguous local timestamps in the database.

Use APP_TIMEZONE=Asia/Bangkok by default. Today and date-based History queries accept an optional IANA `timezone` parameter; validate it and use APP_TIMEZONE when omitted.

The backend converts local midnight and the following local midnight independently into UTC and queries the half-open interval [start, end). Do not assume that a local day is always 24 hours. Inject a clock for determining today's date.

Today filters by kickoff time. History date filters use match kickoff date in the requested timezone, including picks and recommendations linked to those matches. The frontend sends the user's timezone so filtering and displayed dates agree.

---

# 17. Frontend

Use:

- Next.js
- App Router
- TypeScript strict mode

Separate:

- API client
- UI components
- page logic
- view models where useful

Do not hardcode backend URLs.

Support both desktop and mobile.

The application UI must primarily use Thai text.

---

# 18. UI Theme

Default theme:

Dark mode with green accent.

Style:

- professional analytics dashboard
- clean
- modern
- data-focused
- not visually similar to a flashy betting website
- no neon green
- readability first

Suggested palette:

Primary Green:
#16A34A

Dark Green:
#166534

Light Green:
#DCFCE7

Background:
#0B1220

Card Background:
#111827

Border:
#1F2937

Primary Text:
#F9FAFB

Secondary Text:
#9CA3AF

PLAY:
#22C55E

WATCH:
#F59E0B

PASS:
#6B7280

Negative EV / Loss / Error:
#EF4444

Rules:

- positive EV = green
- negative EV = red
- WATCH = amber
- PASS = gray
- recommendation card must be visually prominent
- use moderate border radius
- use generous spacing
- use readable typography
- avoid unnecessary gradients
- avoid excessive colors
- avoid neon effects

---

# 19. Pages

## Today

Show today's matches.

Each match card should show:

- competition
- teams
- kickoff time
- recommendation
- selection
- odds
- probability
- EV
- status

Thai labels should be used in the UI.

Actions:

- เลือกเล่น
- ไม่เล่น
- รายละเอียด

## My Picks

Show only manually selected user picks.

Fields:

- match
- market
- selection
- odds_at_pick
- probability_at_pick
- ev_at_pick
- result if available

## History

Allow filtering by date.

Show separately:

- system recommendations
- user-selected picks
- PASS recommendations
- final match results

## Match Detail

Show:

- odds
- probability
- fair odds
- EV
- expected goals
- lineup
- injuries/suspensions
- odds snapshots
- prediction snapshots
- recommendation snapshots
- reasons
- model version

---

# 20. REST API

Initial endpoints:

GET /health

GET /api/v1/matches/today
GET /api/v1/matches/:id
GET /api/v1/matches/:id/odds
GET /api/v1/matches/:id/lineup
GET /api/v1/matches/:id/prediction
GET /api/v1/matches/:id/recommendation

POST /api/v1/user-picks
GET /api/v1/user-picks
GET /api/v1/user-picks/:id
DELETE /api/v1/user-picks/:id

GET /api/v1/history
GET /api/v1/history/:date

Use consistent validation and error responses.

API behavior:

- POST /api/v1/user-picks accepts recommendation_id only; the backend derives the complete pick snapshot and fixed stake.
- DELETE /api/v1/user-picks/:id performs the cancellation defined in Section 3.
- Singular prediction/recommendation endpoints return the latest generation; odds and lineup endpoints return the latest captured state by default. Match Detail must also expose historical snapshots through a documented history query or its detail response.
- Date filters use Section 16. List endpoints use documented pagination.
- Return structured errors with code and message, using 400 for invalid input, 404 for missing entities, and 409 for stale recommendations, duplicate active picks, or disallowed cancellation.

---

# 21. Seed Data

Create seed data for local development.

Include:

- multiple leagues
- today's matches
- PLAY recommendation
- WATCH recommendation
- PASS recommendation
- odds snapshots
- lineup examples
- user pick examples
- completed matches for History

After seeding, the application should immediately demonstrate the main product flow.

Seed explicit mock expected-goals inputs and whole, half, and quarter lines, including push and half-settlement examples. Seed user picks as clearly identified demo fixtures; runtime generation must never auto-select picks.

Seeding is repeatable and must not erase user-created picks or historical snapshots. Generate today's mock fixtures relative to the injected clock and configured timezone; keep completed demo fixtures available for History.

---

# 22. Required MVP Vertical Slice

Do not build only a skeleton.

The MVP must support this real flow:

Today
→ Recommendation
→ User selects a pick
→ My Picks
→ History

Frontend must call the real Go backend.

Backend must read/write real PostgreSQL.

The football data source can remain mocked.

---

# 23. Docker and Local Development

Use Docker Compose for PostgreSQL.

Do not automatically delete the database volume.

Provide commands such as:

make db-up
make db-down
make db-reset
make migrate
make migrate-down
make migrate-status
make seed
make backend
make worker
make frontend
make dev
make test

db-reset must clearly be destructive.

Never make make dev automatically reset or destroy the database.

---

# 24. Testing

Backend unit tests must include:

- EV
- fair odds
- market probability
- proportional 1X2 bookmaker margin removal
- Poisson implementation
- recommendation rules
- user pick snapshot behavior
- push and half-settlement EV/fair odds, quarter-line settlement, and invalid inputs
- PLAY/WATCH/PASS boundaries, stale odds, and unavailable-input behavior
- pick cancellation, duplicate selection, and fixed-unit ROI
- timezone date boundaries and deterministic clock behavior
- 1X2 Brier score eligibility and aggregation

Add repository integration tests where reasonable.

Frontend tests should cover important behavior only:

- recommendation card
- user pick interaction
- PASS state

---

# 25. Code Quality

Go:

- gofmt
- idiomatic Go
- thin HTTP handlers
- business logic outside transport layer
- wrapped errors where appropriate
- context propagation
- no unnecessary abstractions
- no cyclic dependencies

Next.js:

- TypeScript strict
- avoid any
- no probability or EV business logic in React
- loading states
- error states
- empty states

---

# 26. Security and Configuration

Use environment variables.

Provide .env.example.

Never commit:

- API keys
- passwords
- access tokens
- .env files

Do not hardcode credentials.

Local-only membership requirements:

- Bind the API, frontend development server, and published PostgreSQL port to loopback by default. Keep services bound to localhost; public deployment is deferred.
- Allow browser origins only for the configured local frontend; do not use wildcard CORS.
- Email/password authentication, per-account pick ownership and authorization are implemented locally. TLS and deployment hardening remain required before public deployment.
- Example development credentials are placeholders for local use only and must be supplied through environment configuration.

---

# 27. README

Create README.md containing:

- project overview
- architecture
- Modular Monolith explanation
- Clean Architecture dependency direction
- directory structure
- prerequisites
- Docker setup
- database setup
- environment variables
- migrations
- seed
- running backend
- running worker
- running frontend
- testing
- database reset
- PostgreSQL troubleshooting
- how to replace MockFootballProvider later
- how to replace MockAISummaryProvider later

Expected prerequisites:

- Go
- Node.js
- Docker
- Docker Compose

PostgreSQL should not need to be installed directly on the host.

---

# 28. Implementation Order

Before modifying files:

1. Inspect the existing workspace.
2. Read AGENTS.md and PROJECT_SPEC.md.
3. Report the current structure.
4. Identify conflicts with this specification.
5. Produce a short implementation plan.

Do not modify files until the initial workspace review is complete.

Then implement in this order:

1. domain entities/value objects
2. ports/interfaces
3. use cases
4. unit tests for core logic
5. PostgreSQL migrations
6. repositories
7. REST API
8. mock football provider
9. seed data
10. worker
11. Next.js frontend
12. frontend/backend integration
13. vertical slice testing
14. formatter/tests/build fixes

---

# 29. Final Verification

Before declaring an application implementation complete, run:

1. docker compose up -d
2. docker compose ps
3. confirm PostgreSQL health
4. run migrations
5. run seed
6. gofmt
7. backend tests
8. backend build
9. frontend lint
10. frontend typecheck
11. frontend build

Fix errors before finishing.

Do not stop after creating files.

For documentation-only changes, review consistency and the diff; application infrastructure/tests/builds are not required when no application code exists or changes.

---

# 30. Do Not

Do not:

- create microservices
- add Kubernetes
- add a message broker without a real need
- add Redis in the MVP unless clearly required
- over-engineer
- use real football providers initially
- call Gemini initially
- place probability logic in frontend
- let AI generate the primary probability
- automatically create user picks
- overwrite historical snapshots
- install PostgreSQL through apt
- install system packages through apt unless absolutely necessary

If a local dependency can reasonably run in Docker, prefer Docker.

When a small design detail is unspecified, choose the simplest maintainable solution consistent with Modular Monolith + Clean Architecture.

The final goal is that this repository can be opened locally and the developer can run:

Today
→ Recommendation
→ Select Pick
→ My Picks
→ History

using a real PostgreSQL database in Docker and mock football data.


# 31. Membership (requested extension)

- Register with display name, email, and password; log in and log out without Google or an email provider.
- Normalize email to lowercase; New registrations require 9–72 visible ASCII characters (English letters, digits and standard punctuation); reject whitespace, Thai, emoji and other non-ASCII characters. Existing account passwords remain usable at login. Store bcrypt hashes only (cost 12).
- Persist random opaque sessions as SHA-256 hashes with seven-day expiry. Use HttpOnly, SameSite=Lax cookies; Secure when served via HTTPS.
- Require authentication on all /api/v1 application endpoints except register/login. Health remains public. Require the configured frontend Origin on mutation requests; reject unauthorized CORS origins.
- Proxy browser API calls through Next.js /api to keep cookies same-origin when the frontend uses localhost and the Go API uses 127.0.0.1. Never put session tokens in localStorage.
- Resolve the account from the server session; never trust a user ID supplied by the browser. Scope pick creation, uniqueness, cancellation, details, card selection state and personal ROI/history to that account. Fixtures, system recommendations and model Brier scores remain shared.
- Serialize initial registration with worker/pick transactions; assign existing ownerless local picks to the first registered account without changing their saved prediction/odds snapshots. Later seed demo picks remain ownerless and do not appear in personal records.
- Limit register/login attempts to ten per minute per direct client IP. The MVP does not trust forwarded IP headers and remains local.
- Migration 002 preserves legacy data. Membership rollback is deliberately unsupported because it would lose account/session data and merge account ownership.
- Test authentication errors, session expiry/revocation, CSRF/origin rejection, legacy ownership assignment, and cross-account read/cancel isolation with PostgreSQL.
