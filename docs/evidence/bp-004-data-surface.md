# BP-004 — Adaptive inventory data surface

ADR-011 / ADR-169. BPTable chooses a continuous desktop surface by local
constraints and number of columns. Flexible columns emphasize the entity;
rows separated by rules, actions have their own width independent of sidebar.
Desktop icon actions have 48px targets, tooltip and contextual semantic
name; compact layout keeps visible short labels. Search and filters align
when space allows; BPTableHeader places primary action beside heading.
No new dependencies, no duplicated stock or permission logic.
RED: missing continuous surface. GREEN: 31 distinct focused tests (2 new,
29 previous), in two groups. Includes wide row <=72 for simple fixture,
48px action, 320/800/1440 resize, long names, 200% scale, semantics and
existing CRUD/consultation/filter/pagination/catalog flows. Intermediate
semantic test exposed missing actionable label; label moved inside button
and verified. Analyzer and format clean. Remaining full suite and CI,
browser/device evidence and independent review pending. No production changes.

## Local integration of milestone branches (not a merge to main)
Combined in detached worktree: #197 873352c, #199 a99c614, #201 d5622cf,
#203 7b77538 then 02d36d0, #205 77dc4cc. Local integration head
e07ea2eff5ff3189db1bd53be3c972ec753fbe49. First combined invocation
timed out; repeat in groups. 79 distinct tests passed across the two
integrated revisions (40 before rail polish; 40 after, resize repeated).
Final integrated analyzer clean. Final capture script from #201 extended
locally to widths 360,800,1440; three renders passed with bundled fonts.
- [Mobile](visual/adaptive-integration/products-360.png)
- [Intermediate](visual/adaptive-integration/products-800.png)
- [Desktop](visual/adaptive-integration/products-1440.png)
Reviewed render: menu fills height from top, aligned labels and readable
rail; table contiguous, search/filter and primary action aligned.
Synthetic fixtures explain unused vertical space in the wide render.
Images demonstrate the combined local code, not #205 in isolation.
No browser, device or hosted preview verified. Full CI on integrated main
and independent review remain pending. No automatic merges or deployment.
