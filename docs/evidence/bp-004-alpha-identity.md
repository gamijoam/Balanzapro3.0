# BP-004 — Alpha visual identity

ADR-011 / ADR-169. Central type sizes/weights, semantic ink and muted
colors, consistent base Geist style, navigation distinct from action
buttons (ghost with soft selected icon background and bold active label).
Existing focus handling and selected semantics retained. No new font
or dependency: Geist already supplied by Shadcn. Identity catalogue:
flutter run -t lib/identity_catalog_main.dart -d chrome.

RED: 4 failures (heading 24 instead of 22, missing navigation active
marker at 360/800/1440). GREEN: 66/66 local tests (6 new, 60 regressions)
in four groups. Contrast pairs checked >=4.5:1 for text on light/dark
canvas/surface and active navigation. Dark 200% catalogue, shared touch
target and other navigation regressions passed. Analyzer/format clean.

## Actual widget-render evidence
- [Products 360x900](visual/alpha-identity/products-360.png)
- [Products 1440x900](visual/alpha-identity/products-1440.png)

Rendered with Flutter widget tester at DPR 1 on Linux, loading bundled
Geist/Lucide fonts from FontManifest. First capture failed because
manifest path escaped brackets; decode URI path. Initial render exposed
unstyled table text using test default font; central base text style
corrected and images regenerated. These are captures of the application
render tree, NOT Android hardware or browser. No golden comparison
enforced yet. To regenerate:
cd frontend && flutter test --no-pub --update-goldens tool/visual_capture_test.dart
Outputs tool/visual-evidence; review manually before replacing evidence.
Previous baseline captures had default test font artifacts and are not
published as a misleading device comparison. Analyzer initially flagged
dynamic font list; explicit cast fixed and capture rerun passed.

Visual review: desktop sidebar is vertically centered; compact row actions
and filter controls still consume significant space. Record these for the
integrated review after #197/#199; no overlapping edits in this PR.
Brand mark and final identity remain provisional. No new dependencies,
API, business data, deployment or production changes.
CI and independent review pending. Rollback: revert PR.
