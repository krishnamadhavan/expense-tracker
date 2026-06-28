-- Remove seed catalog for the stable default household only.
DELETE FROM accounts WHERE household_id = '11111111-1111-4111-8111-111111111111'::uuid;
DELETE FROM income_streams WHERE household_id = '11111111-1111-4111-8111-111111111111'::uuid;
DELETE FROM categories WHERE household_id = '11111111-1111-4111-8111-111111111111'::uuid;
DELETE FROM household_settings WHERE household_id = '11111111-1111-4111-8111-111111111111'::uuid;
DELETE FROM households WHERE id = '11111111-1111-4111-8111-111111111111'::uuid;
