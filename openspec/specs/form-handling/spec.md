# form-handling Specification

## Purpose
Defines how forms manage local state, validate input at runtime, and surface errors: each form owns its state via React Hook Form (no shared global form buffer or busy flag), and structural validation is generated from `openapi.yaml` rather than hand-written.

## Requirements
### Requirement: Per-form local state
Each form MUST manage its own local state via React Hook Form. There MUST NOT be a single shared global form buffer or a single shared submit-busy flag across all forms.

#### Scenario: Concurrent forms are independent
- **WHEN** two different form sheets are open
- **THEN** editing one does not affect the other's field values
- **AND** each form's submit control reflects only its own in-flight state

### Requirement: Runtime validation derived from the spec
Structural runtime validation MUST use schemas generated from `openapi.yaml`, kept drift-free. Only cross-field/semantic rules may be hand-written, as refinements on the generated schema.

#### Scenario: Generated schema regenerates cleanly
- **WHEN** the schema generation runs
- **THEN** the generated Zod module is produced with no diff against the checked-in version

#### Scenario: Cross-field rule enforced
- **WHEN** an event form has an end time not after its start time
- **THEN** validation fails with a field-level error on the end time

### Requirement: Per-field error display
Validation errors MUST be shown per field through the existing field component (label, error text, aria-invalid), not as a single global message.

#### Scenario: Missing required field
- **WHEN** a required field is left empty and the form is submitted
- **THEN** that specific field shows its own localized error and the form does not submit

