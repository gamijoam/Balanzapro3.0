# BP-003 — Flutter web development scaffold

- Ubuntu 24.04 x86_64, available memory 5.6 GiB after operator cleanup.
- Flutter 3.50.0 (stable, 2026-11-11), Dart 3.14.0. Official SDK archive SHA-256:
  6ea2d806fd57269ae783e6788e2a81fc26eaad782f1e0d0ff38623bb765ff9f5.
- SDK license BSD-3-Clause; direct dependencies only SDK flutter and
  flutter_test. SDK-resolved transitive dependencies locked in pubspec.lock.
  No UI library added. Full transitive license/advisory review remains BP-002.
- RED: missing BalanzaProApp and generated Material UI dependency not
  declared. First GREEN attempts exposed non-const WidgetsApp and missing
  page route builder; corrected implementation, unchanged acceptance test.
- GREEN: widget test 1/1, flutter analyze no issues (11.4s), dart format.
- Web debug build succeeded in 29.6s with --no-wasm-dry-run, using JS
  compilation. WASM and native builds are not validated. No browser E2E.
- Native scaffold generation was tested outside the repository. Native runner
  files not included in this PR to avoid mixing unverified platform builds.
- No new host packages. Native Linux apt simulation would update 13 packages
  and add 104; use isolated runner instead.
- Alternative: generated Material UI demo introduces additional packages;
  use SDK WidgetsApp until design system ADR-011 is ratified.
