# BP-004 — Shadcn adaptive inventory spike

Experimental only. ADR-011 remains pending ratification. Not a release or
production dependency decision. No real users, API, writes or database.

Candidate shadcn_ui 0.59.0, published 2026-11-09, MIT license verified
in downloaded package. pub.dev official archive SHA-256:
ec2d12a6692d962702a19bd8a23fab0b90102b32022975d0babc48e3f09c3c6a.
Flutter 3.50.0 / Dart 3.14.0 resolve the package and lockfile.
Alternative shadcn_flutter or SDK widgets; prefer wrappers around Shadcn
to avoid dependency leaking into future feature code. This spike confines
Shadcn to lib/ui; final token/component API not defined yet. Dependency
adds 53 resolved packages; audit transitives, advisories and font licenses
before adoption. Use bundled fonts, no external font service.

UX: wide >=1000 uses side navigation and column-aligned product rows;
compact uses top navigation and stacked cards; 48 logical pixel navigation
targets; status conveyed in text; selected nav semantics; named search.
Sample inventory only, movement screen explicitly disabled for real operations.
Light/dark palettes supplied with system mode; dark, screen-reader and
keyboard traversal acceptance still pending actual tests.

RED: three widget tests failed because scaffold lacked product view.
GREEN: three tests (360, 800, 1440 logical px) pass; filtering, empty state,
clear search and navigation round trip; analyzer clean and format clean.
Web compilation attempt exceeded 46s local command limit. CI and actual
browser execution pending. Native Windows/Linux/Android/iOS/macOS not built;
100k SKU, virtualized tables and performance remain separate acceptance.
