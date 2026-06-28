# Design Document Summary (revision 3)

**Output:** `/tmp/grok-design-doc-db2bd394.md` (rev 3)  
**Review:** `/tmp/grok-design-review-db2bd394.md`  
**Repository:** `/Users/krishnamadhavan/Documents/xAI/expense-tracker` (greenfield; remote `https://github.com/krishnamadhavan/expense-tracker.git`)  
**Date:** 2026-06-27

## What was produced

Enterprise-pragmatic system design for a **single-household** personal finance app:

- **Go modular monolith** (`internal/` hexagonal), **chi v5**, **PostgreSQL 16**, **golang-migrate**
- **API-first REST** `/api/v1` + OpenAPI 3.1; mandatory error envelope; **Idempotency-Key** on POST transactions
- **SPA:** React + TS + Vite + **Recharts**; **embed in Go** at PR13 (Profile A complete)
- **Auth:** Argon2id sessions + CSRF (**session `csrf_secret` is authority**; cookie delivers same value) + Bearer tokens (90d, `read`/`write`)
- **Domain:** INR `NUMERIC(19,2)`; FY Apr–Mar; transfers = single row + `transfer_account_id`, **excluded from P&L**; CC = spend channel only; **no balances (NG10)**
- **Categorization:** rules + learn-from-moderation; **first rule mismatch = no rule change**; **≥2 conflicts** deactivate learned/system + insert (user rules protected); ConfidenceFn with **tier = priority integer**
- **Platform:** **MVP-functional** = PR01–11 on `make run` or **PR05 minimal compose** + Vite; **MVP-packaged** = +**PR13** Profile A embed; Minikube B / Argo C stretch
- **Key Decisions:** **KD1–KD40**; module `github.com/krishnamadhavan/expense-tracker`
- **PR plan:** **18 PRs**; effort ~4–7 weeks functional MVP, ~5–8 weeks packaged
- **Open preferences only:** PQ1–PQ4 (license default MIT, bootstrap email, chart pin timing, Argo timing)

## MVP alignment (rev 3)

| Tier | Cut line | Runtime |
| --- | --- | --- |
| MVP-functional | PR01–PR11 | `make run` or PR05 `db`+`migrate`+`api` + Vite SPA |
| MVP-packaged (Profile A complete) | + PR13 | One-command compose with embedded SPA |

## Diagrams & sections

System context, categorization/learning, GitOps (Profile C), ERD; full table inventory + normative DDL; appendices (env vars, SPA fallback); decisions-locked checklist.

## Implementation gates

- **PR01–PR03:** ready  
- **PR05 auth/CSRF/idempotency:** ready  
- **PR06–07 learning:** ready (mismatch &lt; N branch specified)  
- **MVP milestone language:** aligned across Overview, KD31, Effort, PR13, milestones  
