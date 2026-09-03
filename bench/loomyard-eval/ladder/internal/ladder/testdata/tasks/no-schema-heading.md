# Task fixture — no output-schema heading

Type: fixture
Status: fixture only, never dispatched

## `<TASK TEXT>` (identical for A, B, C)

> Explain how the fixture loader in this package handles a task file that
> never declares an output schema at all.
>
> Say which function returns the error and what the error names.

## Notes for whoever scores this (ground truth — do not reveal to A/B/C)

This fixture intentionally carries no "## Output schema" heading anywhere,
so `LoadTaskFile` must return a hard error naming this file rather than an
empty schema.
