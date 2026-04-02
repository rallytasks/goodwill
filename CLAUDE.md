# Goodwill Donor Receipt App

A mobile-first webapp where in-kind donors to Goodwill scan a QR code, authenticate via phone/SMS, and generate e-receipts for tax purposes.

## Architecture

- **Backend**: Go, SQLite (WAL mode), deployed on Railway
- **Frontend**: Vanilla HTML/CSS/JS (mobile-first, no framework needed for this flow)
- **Auth**: Phone/SMS via Twilio Verify (donor enters cell, receives code, verifies)
- **Database**: SQLite with inline migrations in `initDB()`
- **Flow**: QR scan → phone entry → SMS code → login → create donation receipt

---

## Hard Rules

### Secret Keys

- Do NOT commit, log, or expose API keys, tokens, database credentials, or secrets
- All secrets live in environment variables (`.env`), listed in `.gitignore`
- `.env.example` must exist with placeholder values for every required secret
- If a secret is accidentally committed, rotate it immediately

### Notification & Communication Safety

- Do NOT send SMS without a dedup mechanism (idempotency key or "already sent" flag)
- Do NOT trigger outbound messages as a side effect of refactors, migrations, or data backfills
- ALWAYS test the negative path: "after verify, does NOT send again", "rapid calls do NOT produce duplicates"

### Idempotency & Mutation Safety

Every handler that mutates state must have:

1. **Idempotency guard** — calling it twice with the same input produces the same result, not a duplicate
2. **Transaction boundary** — multi-step mutations are atomic
3. **Server-side rate limit** — not bypassable by client bugs
4. **Negative-path test** — test that calling it twice does NOT produce duplicates

### Session Security

- HttpOnly cookies, 256-bit random tokens, 30-day expiry
- All donor data scoped to authenticated user

---

## Refactoring Constraints

### What IS allowed

- Splitting Go files into multiple files/packages
- Extracting shared styles, renaming internal functions, improving error handling
- Adding types/interfaces, reducing duplication, improving test coverage

### What is NOT allowed

- Do NOT remove, rename, or change the HTTP method of any existing endpoint
- Do NOT change the JSON request/response shape of any endpoint without a backward-compatibility test
- Do NOT drop or recreate any database table — use `ALTER TABLE ADD COLUMN` for migrations

### Database Migration Pattern

- `CREATE TABLE IF NOT EXISTS` for new tables
- `ALTER TABLE ADD COLUMN` for new columns — never drop and recreate
- WAL mode for SQLite
- Schema migrations must remain backward-compatible

---

## Donor Flow

1. **QR Scan** — Donor scans QR code at Goodwill location, opens webapp
2. **Phone Entry** — Donor enters cell phone number
3. **Code Verification** — Twilio Verify sends SMS code, donor enters it
4. **Authenticated** — Session created, donor sees their dashboard
5. **Create Receipt** — Donor fills out donation details (items, date, location)
6. **E-Receipt** — Receipt generated with unique ID, downloadable/printable for taxes

### Receipt Requirements

- Charity name: Goodwill Industries
- Charity EIN (will need to configure per location)
- Donor name (collected on first use)
- Date of donation
- Description of items donated (in-kind, no dollar values per IRS rules)
- Unique receipt number
- Statement: "No goods or services were provided in exchange for this donation"

---

## Testing Contract

After any change:

- `go test -count=1 ./...` passes with zero failures
- All HTTP endpoints respond with same status codes and JSON shapes
- Database migrations run cleanly on existing databases
