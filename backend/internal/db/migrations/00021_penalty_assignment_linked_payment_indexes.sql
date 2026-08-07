-- +goose NO TRANSACTION

-- +goose Up

-- Partial: only the (comparatively rare) transactions actually linked to a
-- penalty assignment need this index -- backs both the paid-amount LATERAL
-- join in finances.Repository.ListAssignments/GetAssignmentByID and the
-- ON DELETE SET NULL cascade lookup when an assignment is deleted.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_penalty_assignment_id ON transactions (penalty_assignment_id) WHERE penalty_assignment_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_transactions_penalty_assignment_id;
