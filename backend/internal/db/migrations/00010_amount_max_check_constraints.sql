-- +goose Up

-- validate.PositiveAmount now also rejects amounts above maxAmountCents, but
-- the database should not rely solely on application code for this
-- invariant (see 00007). Without an upper bound, a single amount near the
-- int64 range makes the ::BIGINT-cast SUM(amount) aggregates in
-- finances.SumTransactions / ListOpenPenaltiesByUser overflow and error out,
-- permanently breaking the team's finance overview for every reader until a
-- DBA deletes the offending row directly. NOT VALID + a separate VALIDATE
-- CONSTRAINT avoids an ACCESS EXCLUSIVE lock while scanning existing rows.
ALTER TABLE transactions  ADD CONSTRAINT transactions_amount_max  CHECK (amount <= 100000000) NOT VALID;
ALTER TABLE penalties     ADD CONSTRAINT penalties_amount_max     CHECK (amount <= 100000000) NOT VALID;
ALTER TABLE contributions ADD CONSTRAINT contributions_amount_max CHECK (amount <= 100000000) NOT VALID;

ALTER TABLE transactions  VALIDATE CONSTRAINT transactions_amount_max;
ALTER TABLE penalties     VALIDATE CONSTRAINT penalties_amount_max;
ALTER TABLE contributions VALIDATE CONSTRAINT contributions_amount_max;

-- +goose Down

ALTER TABLE transactions  DROP CONSTRAINT IF EXISTS transactions_amount_max;
ALTER TABLE penalties     DROP CONSTRAINT IF EXISTS penalties_amount_max;
ALTER TABLE contributions DROP CONSTRAINT IF EXISTS contributions_amount_max;
