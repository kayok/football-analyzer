# Agent Instructions

Before making changes:

- Read `PROJECT_SPEC.md` completely.
- Inspect the existing workspace before editing files.
- Report the implementation plan before large structural changes.
- Preserve existing user code unless a change is necessary.

Architecture:

- Follow Modular Monolith + Clean Architecture.
- Do not introduce microservices.
- Keep dependency direction toward domain/use cases.
- Do not place business logic in HTTP handlers or React components.
- Keep `recommendation` and `user_pick` as separate concepts.

Backend:

- Core probability, fair odds, and EV calculations must live in the Go backend.
- Use settlement-aware EV and fair odds for markets with pushes or half outcomes, as defined in `PROJECT_SPEC.md`.
- Keep recommendation thresholds and eligibility checks in backend domain/use cases.
- Create user picks from persisted snapshots; cancellation must preserve their history.
- Preserve historical snapshots instead of overwriting them.
- Use explicit dependency injection.
- Avoid global state.
- Use `context.Context` appropriately.
- Inject clocks for time-dependent behavior; filter local dates using validated IANA timezones and UTC query bounds.

Infrastructure:

- Do not install system packages with `apt` unless absolutely necessary.
- Prefer Dockerized infrastructure for local dependencies.
- Do not install PostgreSQL directly on the host.
- Do not connect to real football APIs or real AI providers in the first implementation.
- Keep the unauthenticated, single-user MVP bound to localhost; follow the explicit MVP/deferred scope in `PROJECT_SPEC.md`.

Safety:

- Do not delete database volumes automatically.
- Do not run destructive commands without explicit approval.
- Do not commit secrets, API keys, passwords, or `.env` files.

Quality:

- Keep changes small and maintainable.
- Do not over-engineer.
- Run relevant tests and builds before finishing application changes; review consistency and diffs for documentation-only changes.
- Fix build, typecheck, and test failures caused by your changes.
