## Why

The statistics module and the event summary answer "was the member present?" with two different definitions. `stats/repository.go` counts only **explicit** attendance rows (`COUNT(*) FILTER (WHERE a.status = 'yes')`, `status IN ('yes','no','maybe')`), while the event summary computes an **effective** status (opt_out→yes, an overlapping absence→no). For an opt-out training series where most members never respond, the event detail shows "15 attending" but the stats show a 0% quote for the same members — a user-visible contradiction in a core club workflow. Additionally, event-level aggregations count ex-members' retained attendance rows while member-level aggregations filter to current members, so the two numbers in the same view no longer reconcile.

## What Changes

- Compute attendance stats from the **effective** status, using the same expression as the event summary, extracted into one shared place so the two cannot diverge again.
- Reconcile event-level vs member-level aggregations regarding ex-members (consistent inclusion/exclusion).
- Update/extend stats tests to cover opt-out defaults and absence-derived "no".

## Capabilities

### New Capabilities
- `attendance-statistics`: how attendance quotes are computed, ensuring they match the effective attendance shown on events.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: `internal/stats/repository.go` (+ service), a shared effective-status SQL expression (candidate: reuse/extract from `internal/events`), stats tests.
- No API/schema change; no migration.
