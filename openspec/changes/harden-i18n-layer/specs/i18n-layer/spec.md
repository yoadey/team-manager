## ADDED Requirements

### Requirement: Locale switches are reflected across the whole mounted tree
Changing the active locale MUST update every rendered translated string in
the application, not only components that explicitly subscribe to the
locale.

#### Scenario: Locale changed while other screens are mounted
- **WHEN** the user switches the active locale via Settings
- **THEN** every currently mounted component's translated text reflects
  the new locale
- **AND** no component continues to show text in the previous locale
  after the switch

### Requirement: Plural interpolation does not require duplicating the count value
A translation call that selects a plural form via a `count` parameter
MUST NOT require the caller to separately pass the same value again for
interpolation.

#### Scenario: Plural call site passes only `count`
- **WHEN** a translation is called with `count` but without a separate
  `n` parameter
- **THEN** the rendered text interpolates the numeric value correctly
- **AND** no literal placeholder text appears in the rendered output

### Requirement: Translation keys are checked at compile time
`t()`'s key parameter MUST be type-checked against the translation
catalog, so a nonexistent or misspelled key fails to compile rather than
silently rendering the raw key string at runtime.

#### Scenario: A call site references a nonexistent key
- **WHEN** code calls `t()` with a key not present in the translation
  catalog
- **THEN** the TypeScript build fails
