# BP-004 — Controlled adaptive table

ADR-011. No new dependencies, API or business logic. BPTable receives one
page with stable IDs and column strings. Reject invalid page metadata,
duplicate IDs, mismatched column count and >50 rows or >8 columns.
Search, selection and page changes invoke caller callbacks; no fetching.
Wide rows and compact cards use shared BP components. Large text switches
to compact layout. Loading/error hide supplied stale rows; empty and retry
states are explicit. Synthetic nine-row example via table_catalog_main.dart.
The page size and column limits are provisional UI guardrails.

RED: missing bp_table.dart, row type and state enum.
GREEN: 7 new tests and 10 regressions (17/17); 360/800/1440 logical px,
page requests, disabled previous, selection, search, stale rows hidden,
loading/empty/error retry, invalid contracts, synthetic example, dark 200%
text at 1440. Analyzer and format clean.
Not a benchmark; no native build, real keyboard traversal, screen reader,
golden, selection multiple, sorting, API or actual browser runtime.
The widget copies its supplied page before rejecting invalid inputs; callers
must enforce bounded data before constructing UI. Loading placeholders are
generic, not exact geometry for every row. Separate catalog entrypoint build
and full accessibility remain pending. No production use.
