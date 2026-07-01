-- +goose Up

-- Service-layer validation (validate.PositiveAmount) already rejects
-- non-positive amounts, but the database should not rely solely on
-- application code for this invariant — add matching CHECK constraints
-- as defense-in-depth against bad data from any other write path.
ALTER TABLE transactions  ADD CONSTRAINT transactions_amount_positive  CHECK (amount > 0);
ALTER TABLE penalties     ADD CONSTRAINT penalties_amount_positive     CHECK (amount > 0);
ALTER TABLE contributions ADD CONSTRAINT contributions_amount_positive CHECK (amount > 0);

-- +goose Down

ALTER TABLE transactions  DROP CONSTRAINT IF EXISTS transactions_amount_positive;
ALTER TABLE penalties     DROP CONSTRAINT IF EXISTS penalties_amount_positive;
ALTER TABLE contributions DROP CONSTRAINT IF EXISTS contributions_amount_positive;
