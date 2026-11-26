# BP-003 — Android and web Alpha packaging

Flutter 3.50.0 SDK verified by existing archive SHA/revision; Android runner
template from that SDK. Alpha application ID com.balanzapro.balanzapro_alpha,
min SDK 24, compile/target 36, NDK 28.2.13676358. Generated local properties,
IDE files, registrant, wrapper jar/scripts and signing keys not committed.
AGP 9.3.1, Kotlin 2.4.20, Gradle 9.5.0 from the pinned Flutter template.
Official Gradle 9.5 wrapper downloaded and verified against official checksum;
distribution checksum pinned. JDK17 and SDK36 on ubuntu-24.04 runner;
NDK installed in ephemeral CI, not VPS. No new Dart dependencies.
actions/upload-artifact v7.0.2 (2026-10-07), MIT, exact commit pinned;
GitHub advisories endpoint returned none. Alternative local packaging,
rejected to avoid host dependencies and manual distribution. CI read-only;
no secrets or deployment, public repository so artifacts contain fixtures only.

Validation: existing main had no Android runner or downloadable package (RED).
YAML parsed, bash syntax checked, manifest JSON/XML parsed, pinned action
references validated, official Gradle jar SHA matched (local static evidence).
Actual Android build/signature, web package upload and installation pending
CI; not GREEN until verified. Debug signing ephemeral, not release trust.
No public site, no actual devices or browser smoke, no commercial API.
Rollback: revert this PR; no changes to backend or production.
