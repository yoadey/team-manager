## REMOVED Requirements

### Requirement: Idempotent paid-state changes
**Reason**: Both settable paid-state operations this requirement covered are
removed by this change. A contribution's paid state was already made a
derived value (see the `membership-fees` capability) rather than an
independently settable one; a penalty assignment's paid state is now derived
the same way, from the sum of income transactions linked to it, so there is
no longer a settable "paid state change" for idempotence to apply to.
**Migration**: None — no client-visible replacement is needed; recording a
payment (for either a contribution or a penalty assignment) is done by
creating a transaction linked to it, which is naturally idempotent-safe the
same way any other transaction creation already is (a retried create is a
client concern, not a toggle-flip risk).
