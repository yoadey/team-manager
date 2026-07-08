-- +goose Up

-- events_end_after_start_time (00012) and absences_span_within_limit (00016)
-- were both added NOT VALID so existing rows wouldn't block deployment, but
-- neither migration followed up with VALIDATE CONSTRAINT -- unlike 00010's
-- equivalent amount-max constraints, which validate in the same migration.
-- Functionally harmless (both constraints are already enforced against every
-- write since their creation), but pg_constraint.convalidated stays false
-- for these two indefinitely, which a compliance/db-hygiene audit would
-- flag, and any pre-existing row that predates 00012/00016 and violates
-- either constraint would never be caught. VALIDATE CONSTRAINT takes a
-- SHARE UPDATE EXCLUSIVE lock (not ACCESS EXCLUSIVE) and does not block
-- concurrent reads/writes while it scans.
ALTER TABLE events   VALIDATE CONSTRAINT events_end_after_start_time;
ALTER TABLE absences VALIDATE CONSTRAINT absences_span_within_limit;

-- +goose Down

-- Validating a constraint is not reversible in any meaningful sense (the
-- constraint itself already existed and was already being enforced against
-- new writes before this migration; this only marks it as checked against
-- existing rows too). Nothing to undo.
