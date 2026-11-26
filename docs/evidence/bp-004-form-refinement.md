# BP-004 — Product form refinement

ADR-011 and ADR-169. BPForm supports optional title, description, section,
required marker with fallback validation, help and optional cancel.
Single scroll area; actions stacked on narrow or large-text layouts,
side-by-side when space permits. Shared semantic error tone. Cancel
disabled while a submission is busy or uncertain; this is a mounted
widget guard, not a persistent operation reconciliation mechanism.
Product editor uses BPDialog when its current values differ from initial
values; cancellation of dialog retains input; confirmed discard returns
to previous product view. Unchanged drafts exit without confirmation.
Compact shell reserves keyboard inset and hides bottom navigation while
keyboard is open. Catalogue demonstrates required fields/help/reset.

RED: 3 failures for missing identification section at 360/800/1440.
GREEN: 65/65 (5 new, 60 regressions) across four groups. New tests
cover dirty draft confirmation, simulated 300px keyboard inset and
scroll to save, required fallback and cancellation lock during busy/
uncertain submission. Existing cancellation test confirms discard
explicitly. Existing scale/dark tests and keyboard Next/Done pass.
Analyzer and format clean. No new dependencies or lock changes.

Limits: no real keyboard/IME or visual device/browser validation of this
revision, no goldens, no server validation/persistence, no business rules.
The draft is only in memory; app shutdown may lose it. Navigation outside
BPForm is not a durable operation lock. No real operations exposed.
CI and independent review pending. No production changes.
Rollback: revert PR.
