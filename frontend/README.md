# BalanzaPro3.0 — Flutter development shell

This is a minimal web scaffold (not an inventory application or a release).
The app displays an explicit synthetic-data banner. It has no credentials,
business operations, API calls or local database.

SDK: Flutter 3.50.0 stable, Dart 3.14.0, revision
ab2912035e160d9cad6306df307576da6128ebdb. Candidate for BP-002;
not a ratification of the full dependency matrix or design system.

From frontend/:
    flutter pub get --enforce-lockfile
    dart format --output=none --set-exit-if-changed lib test
    flutter analyze --no-pub
    flutter test --no-pub --concurrency=1
    flutter build web --debug --no-pub --no-wasm-dry-run

Use the official SDK archive and verify its published checksum.
On VPS the SDK is outside the repository, under /opt/BalanzaPro3.0-dev/sdk/flutter.

Native platform runners, Shadcn, navigation and inventory fixture screens
are separate follow-up work. No public deployment. ADR-169 remains in force.

## Development component catalog

Separate entrypoint; no route is added to the inventory application.

    flutter run -t lib/catalog_main.dart -d chrome
    flutter test --no-pub --concurrency=1 test/component_catalog_test.dart

Requires an installed browser for interactive web execution. For a separately
built web preview use `flutter build web --debug -t lib/catalog_main.dart
--no-pub --no-wasm-dry-run`. Do not publish it as a production app.
The counter and loading switch are local examples, not commercial actions.
System brightness controls the theme. Extend this catalog as shared component
APIs grow; Widgetbook remains a candidate.

### Table catalog example

    flutter run -t lib/table_catalog_main.dart -d chrome

The example is synthetic and independent of the inventory entrypoint.
BPTable receives one page (up to 50 rows and eight columns); the caller
owns query, page requests, selection and error recovery. Use full remaining
viewport height. This is not a virtualized 100k SKU grid.
