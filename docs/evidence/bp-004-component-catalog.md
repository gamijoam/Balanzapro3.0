# BP-004 — Internal component catalog

No new dependencies. ADR-011 and AGENTS.md govern the component examples.
Separate lib/catalog_main.dart, not imported by the inventory entrypoint.
Examples for BPButton primary/secondary/disabled, BPInput, BPCard and
BPSkeleton. Explicitly synthetic counter and manual loading switch. Loading
region has semantics and reserved geometry; no simulated network request.
Responsive column/row layout uses existing tokens.

RED: missing catalog entrypoint. Intermediate regression: enlarged text
(200%, dark, 360 logical px) revealed a provider button overflow. Fix centrally
in BPButton: constrain label to available space with allowance for decoration,
permit vertical growth, preserve 48 logical px minimum. Adjust the test to
scroll until the input is actually hit-testable, with the outer scrollable
explicitly selected. No disabling text scale or hiding overflow.
GREEN: full suite 10/10, including 360/800/1440 catalog interactions,
enlarged text/dark and keyboard backspace, and 6 existing regressions.
Analyzer and format clean. Existing dependency lock unchanged.

Limitations: keyboard input is not full Tab/Enter traversal; screen-reader
and actual hardware tests, golden baselines, separate catalog web build,
error/offline/empty components, and native builds are pending. System
brightness only, no runtime token editor. Visual catalog is provisional
under ADR-011, not Widgetbook adoption or production deployment.
