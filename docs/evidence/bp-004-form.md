# BP-004 — Shared BPForm

ADR-011; separate sibling of BPTable, based on component catalog #173.
No new dependency, API, persistence or business validation.
BPTextField supplies caller-owned controller, label and presentation validator.
BPForm owns focus, inline errors, first invalid focus, Next/Done keyboard action,
immutable submission snapshot and single in-flight callback per mounted form.
Fields disabled while sending. A true callback confirms success; false must
mean a confirmed rejection. Exceptions lock the mounted form until caller
reconciles outcome and remounts; no exception details rendered. This is not
durable idempotency or a guarantee against double submission across instances.
Form width central token 720 logical px, one-column layout, max 12 fields
(provisional guardrail). No use of raw Shadcn outside design system.
Run synthetic example: flutter run -t lib/form_catalog_main.dart -d chrome.
Explicitly no real product persistence.

RED missing form types. GREEN 18/18 (8 new, 10 existing), 360/800/1440,
validation/focus, Next/Done, single in-flight callback, known rejection retry,
unknown outcome lock, 200% text/dark, synthetic example, late completion
after unmount. Analyzer and format clean.
Limitations: dynamic schema/externally edited controllers and validators
throwing not fully tested. Validation callbacks must be pure and nonthrowing.
No durable operation tracking, select/number/date fields, server validation,
full Tab traversal, actual IME/reader, goldens, native or separate web build.
