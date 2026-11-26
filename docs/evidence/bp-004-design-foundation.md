# BP-004 — Design foundation prototype

Depends on #166 and ratification #167; no production adoption yet.
Design system centralizes the provisional primary palettes, radius,
spacing, touch target, desktop breakpoint and sidebar width.
BPApp adapts the provider, BPVisuals exposes semantic styles, BPButton,
BPInput and BPCard encapsulate Shadcn; BPSkeleton is static and excludes
individual placeholder semantics. The caller must announce actual loading.
No artificial loading added to synthetic inventory.

Component examples in test/design_system_test.dart serve as executable
catalog seed: enabled/disabled button, skeleton geometry, system theme.
The inventory uses the same components and has no direct Shadcn import.
No new dependencies relative to #166. Palette is provisional.
RED: missing design_system package/types. GREEN: 6 tests passed:
button/skeleton, system theme switch, primary pair contrast >=4.5,
360/800/1440 logical px inventory regression. Analyzer and format clean
before adding final theme/contrast tests; final analyzer to be verified.
Actual browser and native runtime, golden tests, keyboard/screen-reader
traversal, full contrast, text scaling, visual catalog and Widgetbook remain
pending. CI of stacked PR not equivalent to validation against main.
