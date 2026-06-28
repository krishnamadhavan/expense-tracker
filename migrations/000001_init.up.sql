-- Authoritative v1 schema (design PR03). Cross-household FK equality is enforced in app (PR04), not DB.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE households (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name              TEXT NOT NULL,
  default_currency  CHAR(3) NOT NULL DEFAULT 'INR',
  fy_start_month    SMALLINT NOT NULL DEFAULT 4 CHECK (fy_start_month BETWEEN 1 AND 12),
  timezone          TEXT NOT NULL DEFAULT 'Asia/Kolkata',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE household_settings (
  household_id   UUID PRIMARY KEY REFERENCES households(id) ON DELETE CASCADE,
  locale         TEXT NOT NULL DEFAULT 'en-IN',
  features       JSONB NOT NULL DEFAULT '{}'::jsonb,
  psp_suffixes   JSONB,
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id   UUID NOT NULL REFERENCES households(id),
  email          CITEXT NOT NULL UNIQUE,
  password_hash  TEXT NOT NULL,
  role           TEXT NOT NULL DEFAULT 'admin',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_household ON users (household_id);

CREATE TABLE sessions (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash       TEXT NOT NULL UNIQUE,
  csrf_secret      TEXT NOT NULL,
  expires_at       TIMESTAMPTZ NOT NULL,
  idle_expires_at  TIMESTAMPTZ NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_expires ON sessions (expires_at);

CREATE TABLE api_tokens (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name          TEXT NOT NULL,
  token_prefix  TEXT NOT NULL,
  token_hash    TEXT NOT NULL UNIQUE,
  scopes        TEXT NOT NULL DEFAULT 'write'
                CHECK (scopes IN ('read', 'write')),
  expires_at    TIMESTAMPTZ NOT NULL,
  revoked_at    TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_tokens_user ON api_tokens (user_id);

CREATE TABLE accounts (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id  UUID NOT NULL REFERENCES households(id),
  name          TEXT NOT NULL,
  type          TEXT NOT NULL CHECK (type IN (
                  'upi', 'debit_card', 'credit_card', 'cash', 'bank', 'other')),
  currency      CHAR(3) NOT NULL DEFAULT 'INR',
  is_active     BOOLEAN NOT NULL DEFAULT true,
  metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (household_id, name)
);

CREATE INDEX idx_accounts_household ON accounts (household_id);

CREATE TABLE categories (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id  UUID NOT NULL REFERENCES households(id),
  parent_id     UUID REFERENCES categories(id),
  name          TEXT NOT NULL,
  kind          TEXT NOT NULL CHECK (kind IN ('expense', 'income', 'transfer')),
  is_system     BOOLEAN NOT NULL DEFAULT false,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (household_id, kind, name)
);

CREATE INDEX idx_categories_household ON categories (household_id);

CREATE TABLE income_streams (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id  UUID NOT NULL REFERENCES households(id),
  name          TEXT NOT NULL,
  code          TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (household_id, code)
);

CREATE INDEX idx_income_streams_household ON income_streams (household_id);

CREATE TABLE tags (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id  UUID NOT NULL REFERENCES households(id),
  name          TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (household_id, name)
);

CREATE TABLE import_batches (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id  UUID NOT NULL REFERENCES households(id),
  source        TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'pending',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE transactions (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id         UUID NOT NULL REFERENCES households(id),
  account_id           UUID NOT NULL REFERENCES accounts(id),
  transfer_account_id  UUID REFERENCES accounts(id),
  category_id          UUID REFERENCES categories(id),
  income_stream_id     UUID REFERENCES income_streams(id),
  direction            TEXT NOT NULL CHECK (direction IN ('income', 'expense', 'transfer')),
  amount               NUMERIC(19, 2) NOT NULL CHECK (amount >= 0),
  currency             CHAR(3) NOT NULL DEFAULT 'INR',
  txn_date             DATE NOT NULL,
  payee_raw            TEXT,
  payee_norm           TEXT,
  memo                 TEXT,
  external_ref         TEXT,
  source               TEXT NOT NULL DEFAULT 'manual',
  category_confidence  NUMERIC(5, 4),
  category_locked      BOOLEAN NOT NULL DEFAULT false,
  import_batch_id      UUID REFERENCES import_batches(id),
  voided_at            TIMESTAMPTZ,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT txn_transfer_accounts CHECK (
    (direction = 'transfer' AND transfer_account_id IS NOT NULL
      AND transfer_account_id <> account_id)
    OR (direction <> 'transfer' AND transfer_account_id IS NULL)
  ),
  CONSTRAINT txn_income_stream CHECK (
    (direction = 'income' AND income_stream_id IS NOT NULL)
    OR (direction <> 'income' AND income_stream_id IS NULL)
  ),
  CONSTRAINT txn_import_external UNIQUE (account_id, external_ref)
);

CREATE INDEX idx_txn_household_date ON transactions (household_id, txn_date DESC, id DESC);
CREATE INDEX idx_txn_payee_norm ON transactions (household_id, payee_norm);
CREATE INDEX idx_txn_category ON transactions (household_id, category_id);
CREATE INDEX idx_txn_account ON transactions (household_id, account_id);

CREATE TABLE transaction_tags (
  transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  tag_id         UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (transaction_id, tag_id)
);

CREATE TABLE category_rules (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id  UUID NOT NULL REFERENCES households(id),
  category_id   UUID NOT NULL REFERENCES categories(id),
  match_field   TEXT NOT NULL CHECK (match_field IN ('payee_norm', 'payee_raw', 'memo', 'account_id')),
  match_type    TEXT NOT NULL CHECK (match_type IN ('exact', 'prefix', 'contains')),
  pattern       TEXT NOT NULL,
  priority      INT NOT NULL DEFAULT 100,
  confidence    NUMERIC(5, 4) NOT NULL DEFAULT 0.8000,
  origin        TEXT NOT NULL CHECK (origin IN ('system', 'user', 'learned')),
  hit_count     INT NOT NULL DEFAULT 0,
  is_active     BOOLEAN NOT NULL DEFAULT true,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX category_rules_active_match
  ON category_rules (household_id, match_field, match_type, pattern)
  WHERE is_active;

CREATE INDEX idx_category_rules_household ON category_rules (household_id);

CREATE TABLE merchant_norms (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id         UUID NOT NULL REFERENCES households(id),
  norm_key             TEXT NOT NULL,
  display_name         TEXT,
  default_category_id  UUID REFERENCES categories(id),
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (household_id, norm_key)
);

CREATE TABLE categorization_events (
  id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id          UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  suggested_category_id   UUID REFERENCES categories(id),
  rule_id                 UUID REFERENCES category_rules(id),
  confidence              NUMERIC(5, 4),
  outcome                 TEXT NOT NULL DEFAULT 'pending'
                          CHECK (outcome IN ('pending', 'accepted', 'overridden')),
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_categorization_events_txn ON categorization_events (transaction_id);

CREATE TABLE moderation_events (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id    UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  user_id           UUID NOT NULL REFERENCES users(id),
  from_category_id  UUID REFERENCES categories(id),
  to_category_id    UUID NOT NULL REFERENCES categories(id),
  reason            TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_moderation_events_txn ON moderation_events (transaction_id);

CREATE TABLE review_queue (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id  UUID NOT NULL UNIQUE REFERENCES transactions(id) ON DELETE CASCADE,
  reason          TEXT NOT NULL CHECK (reason IN ('low_confidence', 'conflict', 'new_merchant', 'null_payee')),
  status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at     TIMESTAMPTZ
);

CREATE TABLE budgets (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id   UUID NOT NULL REFERENCES households(id),
  category_id    UUID NOT NULL REFERENCES categories(id),
  period_type    TEXT NOT NULL CHECK (period_type IN ('month', 'fy')),
  period_start   DATE NOT NULL,
  amount_limit   NUMERIC(19, 2) NOT NULL CHECK (amount_limit >= 0),
  currency       CHAR(3) NOT NULL DEFAULT 'INR',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (household_id, category_id, period_type, period_start)
);

CREATE TABLE idempotency_keys (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  key            TEXT NOT NULL,
  request_hash   TEXT NOT NULL,
  response_code  INT NOT NULL,
  response_body  JSONB NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, key)
);

CREATE INDEX idx_idempotency_keys_created ON idempotency_keys (created_at);
