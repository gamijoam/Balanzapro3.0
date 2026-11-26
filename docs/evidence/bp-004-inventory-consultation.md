# BP-004 — Inventory consultation

Stacked on #191; ADR-011 and ADR-169. Read-only product details, explicit
fixture stock/status, access to example movement history and return to the
same product/search. Preserve direct editing. BPTable adds an optional
secondary detail callback, illustrated in the internal table catalog.
Movement history has six static example records, search by current
product name/code or fixture reference, type filter, detail and empty
state. Links use stable product IDs, so editing product metadata does not
lose the linked fixtures. Historical labels show current product metadata,
not historical snapshots. The stock figures are pre-supplied fixtures;
no stock calculation, ledger, transfer, valuation or real writes.

RED: three missing detail action failures; first cold test invocation
timed out and repeated test confirmed exact missing controls.
GREEN: 60/60 local tests (5 new and 55 previous) in four groups,
360/800/1440, product detail/history/back, type filter, rename linkage,
empty state, global history and catalogue action; regressions include
200% text on existing shared components. Analyze and format clean.
The old placeholder wording assertion now checks explicit warning about
real movements requiring protected API, preserving navigation checks.
No new dependencies, no production or backend changes.

Limits: no persistent history, no operation creation, no actual kardex,
authentication or permissions; no browser/physical visual review of this
revision. CI does not run main-only previews on stacked base; retarget
after #191 merges, reconcile and revalidate. Separate milestone delivery
still pending. Full keyboard, screen reader and golden testing pending.
Rollback: revert PR after integration.
