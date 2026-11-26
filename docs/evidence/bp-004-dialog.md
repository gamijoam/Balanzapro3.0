# BP-004 — BPDialog

ADR-011; confirmation-only component, no business operation, no API.
Explicit confirm returns true; cancel, outside, Escape and system back
return false. Central width token 480; BPButton autofocus option used for
cancel. Shared card/tokens, scrollable content, no animation. Route label
and explicit children semantics. Double-pop guarded on active route.
Catalog example: flutter run -t lib/dialog_catalog_main.dart -d chrome.
Example only reports the choice; never changes inventory.

RED missing BPDialog. First GREEN attempt caught semantics route assertion;
corrected explicit child nodes. GREEN: 5 new tests + 25 regressions, 30/30;
360/800/1440, explicit confirm/cancel, Escape/outside, Enter with focus
on cancel, 200% scale dark 360x640, system back and synthetic example.
Analyze and format clean. Lockfile unchanged.

Limits: keyboard Tab cycle and focus restoration, real assistive technology,
very short landscape screens, native back/IME and browser runtime remain
unverified. No destructive styling variant yet; no durable operation
idempotency. Independent review and CI pending.
