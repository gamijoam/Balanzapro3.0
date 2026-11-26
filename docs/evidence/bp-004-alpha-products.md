# BP-004 — Interactive synthetic products Alpha

ADR-011 / ADR-169. Integrate shared BPTable, BPForm and BPEmptyState.
Search, single selection, pages of 10, create/edit metadata with local
required/length/duplicate fixture code validation, cancel draft.
Stock remains unchanged on edit and zero on create. Dashboard reflects
the same in-memory fixtures. No persistence, API, permissions or real
inventory actions. Changes disappear on restart/reload. Product codes
normalized uppercase for the fixture; this is not domain validation.
BPTable optional toolbar and clear-search action preserve reusable layout.
Movements still an explicitly unavailable protected-API placeholder.

RED: three new integration tests failed (BPTable missing).
GREEN: 4 new and 41 existing tests passed in four groups (45 total).
The single all-tests command timed out at the MCP limit; split runs
verified every test file. 360/800/1440 create, duplicate rejection,
edit, cancellation, previous navigation/search regressions. Analyzer
and format clean; unchanged Dart lockfile. No production changes.
Independent CI/review and actual device/browser testing pending.
No DB or migrations. Rollback revert PR. Fixture limit/paging UX and
real business semantics, offline synchronization, auth and native
compatibility remain unvalidated.
