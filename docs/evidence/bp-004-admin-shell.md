# BP-004 — Adaptive administration shell

ADR-011 and ADR-169. BPPageLayout and BPSidebar are controlled presentation
components using existing design tokens, semantic selected state and BPButton.
Navigation uses stable IDs and validates nonempty/unique items and selection.
Side navigation on wide layouts, compact wrapping scrollable navigation on
narrow or large-text layouts. Header and navigation are height-bounded for
short windows. Content owns its scroll; keyed pages prevent reusing offsets.
Existing synthetic inventory search preserved across page switches and resizing.
Inicio dashboard shows three explicitly synthetic metrics and a products shortcut.
Default entry remains products, movements remains a protected-API placeholder.
The application preview itself is the example, no real session, routes, writes,
authorization or network calls.

RED: four initial shell tests failed (layout/nav absent). GREEN: six new
tests, 25 existing tests, 31/31 in 28 seconds. 360/800/1440, search retained
across navigation and resizing; short landscape 640x360 dark/200% text;
360x640 200% dashboard shortcut and scroll to input; invalid navigation.
Analyze and format clean. No new dependencies.

Limits: compact menu may scroll at enlarged text; not a final mobile menu.
Header may scroll in very short windows. Full Tab traversal, focus restoration,
screen reader, golden, actual browser execution and native builds pending.
Dashboard values are hardcoded fixtures. Existing product body is retained,
not yet migrated to BPTable/BPForm; no company selector or effective permissions.
