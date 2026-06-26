<!-- AGENT INSTRUCTIONS — read and obey before writing any code. -->
# Goodwill Donor Receipt App

> **Global engineering contract:** https://github.com/rallytasks/agents/blob/main/AGENTS.md
> Every rule there applies to this repo. This file adds only project specifics and any `Override <ID>:`.
@../agents/AGENTS.md   <!-- local-checkout convenience; no-op in a standalone clone -->

A mobile-first webapp where in-kind donors to Goodwill scan a QR code, authenticate via phone/SMS, and generate e-receipts for tax purposes.

## Stack
Go · SQLite (WAL, inline migrations in `initDB()`) · Railway · vanilla HTML/CSS/JS (mobile-first) · auth via Twilio Verify (phone → SMS code). Flow: QR scan → phone entry → SMS code → login → create receipt. Scope key: `user_id` (all donor data).

## Commands
test: `go test -count=1 ./...` (zero failures; endpoints keep status codes + JSON shapes; migrations run clean on existing DBs)

## Top damage risks
1. Duplicate SMS on verify/resend → dedup or "already sent" flag; test "rapid calls do NOT resend" (`OUTBOUND`).
2. Duplicate receipts from a double submit → idempotency guard (`MUTATION`).
3. Cross-donor data leak → every query filters by `user_id` (`TENANT`).

## Project rules & overrides
- **E-receipt must contain (IRS in-kind):** charity name (Goodwill Industries) + EIN (per location), donor name, donation date, item description (in-kind, **no dollar values**), unique receipt number, and the statement "No goods or services were provided in exchange for this donation."
- **Refactoring fences:** do NOT remove/rename an endpoint or change its HTTP method; do NOT change a request/response JSON shape without a backward-compat test (`API`). Splitting Go files, extracting styles, adding tests are fine.

## Doc index
| Topic | File |
|---|---|
| (add as docs are created) | — |
