# Format Drift Workflow

Trigger this workflow when the task mentions format drift, schema drift,
unknown fields, unknown record types, extra fields, `carn.log` drift
warnings, or `known_schema_extras`.

1. Read `VOCABULARY.md` first so drift terms stay aligned with the project
   vocabulary.
2. Read the provider-owned drift files under `internal/source/<provider>/`,
   especially `drift*.go`, `raw_values.go`, and any existing
   `known_schema_extras.go`, `known_schema_extras.json`, and
   `known_schema_extras_test.go`.

## Known Schema vs. Known Schema Extras

- **Known Schema** — fields and values the app actively models or depends
  on.
- **Known Schema Extras** — observed provider-owned fields or values that
  are intentionally tolerated but not yet modeled.

## Adding a Known Schema Extra

- Update the provider catalog entry with `status`, `path`, `record_types`,
  `description`, `future_use`, `first_seen`, and `example`.
- Keep examples small but real enough to contain the declared path and
  document the raw shape that triggered the warning.
- Add or update tests that prove documented extras suppress drift warnings
  and do not duplicate the compile-time known schema.

## Promoting an Extra

If the app starts parsing or depending on an extra field or value, move it
from the known schema extras catalog into the provider's compile-time
known schema maps and update tests accordingly.

## Vocabulary

When a new drift-related term or workflow appears, update `VOCABULARY.md`
before adding new package, file, or helper names.
