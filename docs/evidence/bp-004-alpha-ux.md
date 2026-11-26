# BP-004 — First physical Android feedback

Owner installed the first debug APK and provided a screenshot: excessive
vertical spacing, plain headers and cards, and undifferentiated stock states.
This incremental refinement uses ADR-011 and keeps ADR-169 intact.
Canvas and surfaces differ in light/dark; radius 10, card padding 16/12
logical px with flat surfaces. Bottom navigation at compact widths, side
navigation at wide widths; explicit labels and existing BPButton focus
behavior. Lucide icons already supplied by Shadcn, no new dependencies.
BPBadge semantic tones with explicit text, tested contrast >=4.5:1 in
both themes for all four variants. BPTable compact metadata is two-column,
first cell emphasized; optional cell builder enables status badge
without hardcoding inventory rules in the table. Existing create/edit
and draft flow unchanged. UX.md records the updated criteria; AGENTS.md
already requires following that standard.

RED: three tests failed on missing bottom navigation and badge.
GREEN: 49/49 tests in three groups (4 new, 45 regressions), including
360/800/1440, enlarged text, short landscape and both themes. Analyze
and formatting clean. No change to lockfile. All data remains synthetic.
Actual Android/web builds for this revision, new physical UX review,
goldens, full reader/keyboard review and production branding pending.
The first physical screenshot is evidence only of the previous build.
No backend, permissions, deploy, database or production changes.
Rollback: revert this PR.
