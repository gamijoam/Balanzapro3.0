# BP-004 — Shared view states

ADR-011. BPEmptyState for confirmed empty results; BPErrorState for
recoverable reads/confirmed rejection; BPOfflineState for connectivity
check; BPLoadingState announces load and uses static skeletons or supplied
placeholder geometry. Optional actions only when provided. Shared card,
typography, spacing and buttons. BPTable consumes loading/error/empty
without exposing stale rows. No automatic network detection or write retry.
Example: flutter run -t lib/states_catalog_main.dart -d chrome.
Example actions only update a counter, never fabricate network recovery.

RED missing types. GREEN 30/30 (5 new, 25 regressions), including 360/800/1440,
200% text/dark, optional actions and synthetic example; catalog scrolling
test corrected to ensure target hit-testable. Analyze and format clean.
No dependencies changed; independent sibling of BPDialog.

Limitations: no data freshness or synchronization service, no live network,
no durable operation recovery, no reader or full keyboard traversal test,
no goldens/native/independent catalog build. Default skeleton geometry is
generic; callers with exact layouts must provide matching placeholders.
