# BP-004 — Inventory dashboard refinement

ADR-011 / ADR-169. Reusable BPMetricCard and BPDistribution with semantic
tones and explicit counts; the distribution bar is decorative, never the
only signal. Current synthetic product session supplies counts. Wide
three-column layout, compact total plus two cards, single column on
narrow/large-text layouts. Primary shortcut at top opens new product
or resumes the existing draft; filter drilldown disabled during draft
to avoid unexpected replacement. Existing products/history shortcuts
and current search remain available. No real stock computation.
Separate metrics catalogue: flutter run -t lib/metrics_catalog_main.dart -d chrome.

RED: two initial cold test invocations timed out; repeated run confirmed
3 missing distribution failures at 360/800/1440. A subsequent test exposed
lazy list reachability for the shortcut at 360; moved primary action to
the top, tests scroll to verify other content. GREEN: 65/65 (5 new and
60 regressions) in four groups. Draft retained when navigating, metric
counts update on fixture creation, negative counts rejected, empty
distribution and dark 200% catalogue. Existing enlarged text dashboard
navigation verified. Analyze and format clean. No dependency changes.

Limits: figures are synthetic and temporary; no authentic metrics, API,
permissions or persistence. No actual browser/device visual evidence,
goldens or full accessibility review yet. No production changes.
CI builds and independent review pending. Do not recommend another
manual install until the grouped visual milestone is integrated and tested.
Rollback: revert PR.
