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
