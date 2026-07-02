-- +goose Up

-- Store monetary amounts as integer cents (BIGINT) instead of NUMERIC(10,2)
-- euros. NUMERIC(10,2) was already exact at rest, but every Go call site
-- scanned it into a float64 and the OpenAPI contract exposed it as a JSON
-- number/double, reintroducing binary floating-point imprecision at the
-- API boundary. Storing cents as an integer removes that class of bug
-- end-to-end. The existing *_amount_positive CHECK constraints (see
-- 00007_amount_check_constraints.sql) are type-agnostic and need no change.
ALTER TABLE transactions  ALTER COLUMN amount TYPE BIGINT USING ROUND(amount * 100)::BIGINT;
ALTER TABLE penalties     ALTER COLUMN amount TYPE BIGINT USING ROUND(amount * 100)::BIGINT;
ALTER TABLE contributions ALTER COLUMN amount TYPE BIGINT USING ROUND(amount * 100)::BIGINT;

-- +goose Down

ALTER TABLE transactions  ALTER COLUMN amount TYPE NUMERIC(10,2) USING (amount::numeric / 100);
ALTER TABLE penalties     ALTER COLUMN amount TYPE NUMERIC(10,2) USING (amount::numeric / 100);
ALTER TABLE contributions ALTER COLUMN amount TYPE NUMERIC(10,2) USING (amount::numeric / 100);
