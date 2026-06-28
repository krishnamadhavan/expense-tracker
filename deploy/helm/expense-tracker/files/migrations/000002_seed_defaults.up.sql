-- Default household + India-oriented system catalog for local single-tenant use.
-- App bootstrap (PR05) may attach the first user to this household or create another and copy catalog later.
-- Seed household id is stable for fixtures/docs: 11111111-1111-4111-8111-111111111111

INSERT INTO households (id, name, default_currency, fy_start_month, timezone)
VALUES (
  '11111111-1111-4111-8111-111111111111'::uuid,
  'Default Household',
  'INR',
  4,
  'Asia/Kolkata'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO household_settings (household_id, locale, features, psp_suffixes)
VALUES (
  '11111111-1111-4111-8111-111111111111'::uuid,
  'en-IN',
  '{}'::jsonb,
  NULL
)
ON CONFLICT (household_id) DO NOTHING;

-- Expense categories (system)
INSERT INTO categories (household_id, name, kind, is_system) VALUES
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Uncategorized', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Food & Dining', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Groceries', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Transport', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Fuel/Petrol', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Recharges & Telecom', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Utilities', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Rent', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Domestic Help', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'EMI & Loans', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Insurance', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Health', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Education', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Shopping', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Entertainment', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Fees & Charges', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Travel', 'expense', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Other', 'expense', true)
ON CONFLICT (household_id, kind, name) DO NOTHING;

-- Transfer category (kind=transfer)
INSERT INTO categories (household_id, name, kind, is_system) VALUES
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Transfers', 'transfer', true)
ON CONFLICT (household_id, kind, name) DO NOTHING;

-- Income categories (system)
INSERT INTO categories (household_id, name, kind, is_system) VALUES
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Uncategorized', 'income', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Salary', 'income', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Business', 'income', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Rental', 'income', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Other', 'income', true)
ON CONFLICT (household_id, kind, name) DO NOTHING;

-- Income streams
INSERT INTO income_streams (household_id, name, code) VALUES
  ('11111111-1111-4111-8111-111111111111'::uuid, 'PayCheck', 'paycheck'),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Business Income', 'business'),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Rental Income', 'rental'),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Other', 'other')
ON CONFLICT (household_id, code) DO NOTHING;

-- Default payment channels (no balances)
INSERT INTO accounts (household_id, name, type, currency, is_active) VALUES
  ('11111111-1111-4111-8111-111111111111'::uuid, 'UPI', 'upi', 'INR', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Debit Card', 'debit_card', 'INR', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Credit Card', 'credit_card', 'INR', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Cash', 'cash', 'INR', true),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'Bank Account', 'bank', 'INR', true)
ON CONFLICT (household_id, name) DO NOTHING;
