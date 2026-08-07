-- +goose NO TRANSACTION

-- +goose Up

-- Replaces idx_contributions_team_month (dropped along with the `month`
-- column in 00018): FinancesContributions groups by due date now instead of
-- by month.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contributions_team_due_date ON contributions (team_id, due_date);

-- Partial: only the (comparatively rare) transactions that are actually
-- linked to a contribution need this index -- backs both the paid-amount
-- LATERAL join in finances.Repository.ListContributions/getContributionByID
-- and the ON DELETE SET NULL cascade lookup when a contribution is deleted.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_contribution_id ON transactions (contribution_id) WHERE contribution_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_transactions_contribution_id;
DROP INDEX IF EXISTS idx_contributions_team_due_date;
