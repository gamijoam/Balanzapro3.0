# BP-004 — Adaptive administrative shell
Stacked on #199 to retain keyboard handling. One stateful page layout;
BPAdaptive selects compact/rail/expanded by width and text scale. Sidebar
fills available height from top, proportional capped width, manual collapse
in expanded mode; rail retains labels and touch targets. Desktop destinations
use horizontal icon/label, compact bottom navigation retained. Workspace
key and caller-owned state persist across width changes. No platform checks.
AGENTS and UX updated. No new dependencies.
RED: prior sidebar y=379.5 at 1440x900. GREEN: 34 focused tests
(1 new resize/contract test, 33 regression) across 3 groups; 320,599,600,
800,1023,1024,1440,1920, draft/search/collapse and keyboard simulation.
Analyze and format clean. Existing navigation tests updated for rail mode.
Actual browser/device render, independent review and integrated CI remain
pending. Table refinement is a separate unit. No backend/production changes.
