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
