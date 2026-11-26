# BP-008 — Secret file loading (ADR-026)

The dedicated 32-byte random key must be supplied by the installer/operator
and stored outside the source tree, application database and logs. All server
replicas need the same stable key; back it up separately with restricted access.
Do not regenerate on startup, missing file, backup restore or configuration
reload. A missing or invalid key must prevent starting authentication.

Linux: require an absolute path inside a trusted operator-controlled directory
whose parent components cannot be modified by untrusted users; service-owned
regular file, mode 0400 or 0600, one link, no final symlink. Store raw 32 bytes,
not a text representation. File loading is not key provisioning or encryption
at rest. No production secret created in this PR. Windows needs an audited
ACL and secure storage adapter before authentication is enabled there.

Rotation needs a separate decision and coordinated migration: simply replacing
the secret changes every budget identifier and effectively loses its limits.
Also separate this key from PIN, TOTP, license-signing and token secrets.

RED: missing implementation. GREEN: ten repetitions with race detector,
invalid length, zero key, permissions, missing path, final symlink, directory
and stable repeated read. Vet, format, AST and Windows package cross-build
successful; Windows runtime deliberately returns ErrUnavailable.
