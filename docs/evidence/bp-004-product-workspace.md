# BP-004 — Product workspace refinement

ADR-011 and ADR-169. User requests larger coherent frontend milestones
rather than reinstalling an APK after every small change.
Product list: controlled single-choice BPChoiceBar, combined name/code and
stock filters, count, clear filters and pagination reset. Dashboard metrics
open the corresponding filtered list, clearing the old search. Direct
per-row edit replaces select-then-edit in the product preview. BPTable
retains its existing selection mode for other callers and enforces exactly
one row action mode. Filtering state and search survive navigation;
cancelling an edit retains them. Save resets filter and searches saved code,
with explicit local feedback. Stock semantics remain synthetic fixtures.
ChoiceBar also has a catalogue example. No new dependencies.

RED: 3 tests failed for absent dashboard drilldown. Initial commands timed
out; explicit expectations reproduced missing controls. GREEN: 55/55
(6 new, 49 regression) in four groups, 360/800/1440, combined filters,
dashboard, direct edit, draft cancellation, invalid choice/action contracts,
catalogue with 200% dark text, previous flows. Three regression expectations
updated from "Limpiar búsqueda" to "Limpiar filtros"; edit test now uses
direct action, still verifies saved changes. Analyze and formatting clean.
No changes to lockfile, backend or production.

Limitations: visual review on real device and browser, CI build artifacts,
goldens, full keyboard/accessibility and performance remain pending.
Preview has no persistence, real movement entry, authentication or API.
No new version recommended for manual installation until a coherent
milestone is integrated and validated; CI continues on every PR.
