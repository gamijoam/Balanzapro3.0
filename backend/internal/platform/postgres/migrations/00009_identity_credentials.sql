-- BP-008 / ADR-026. Server-side credential verifiers only.
-- A PHC envelope check is a storage guard, NOT cryptographic validation.
-- The future hashing adapter must enforce bounded Argon2id parameters,
-- independently generated salts, constant-time verification and rate limits.
-- PIN may be absent before enrollment. TOTP and sessions are separate work.
-- +goose Up
CREATE TABLE identity_credentials (
    organization_id uuid NOT NULL,
    user_id uuid PRIMARY KEY,
    password_verifier text NOT NULL,
    pin_verifier text NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (organization_id, user_id) REFERENCES identity_users (organization_id, id),
    CHECK (length(password_verifier) <= 512 AND
        password_verifier ~ '^[$]argon2id[$]v=19[$]m=[0-9]+,t=[0-9]+,p=[0-9]+[$][A-Za-z0-9+/]+[$][A-Za-z0-9+/]+$'),
    CHECK (pin_verifier IS NULL OR (
        length(pin_verifier) <= 512 AND pin_verifier <> password_verifier AND
        pin_verifier ~ '^[$]argon2id[$]v=19[$]m=[0-9]+,t=[0-9]+,p=[0-9]+[$][A-Za-z0-9+/]+[$][A-Za-z0-9+/]+$'))
);

-- +goose Down
-- Never silently discard enrolled credentials.
-- +goose StatementBegin
DO $$
BEGIN
    LOCK TABLE identity_credentials IN ACCESS EXCLUSIVE MODE;
    IF EXISTS (SELECT 1 FROM identity_credentials) THEN
        RAISE EXCEPTION 'identity_credentials: rollback requires explicit credential recovery plan';
    END IF;
END;
$$;
-- +goose StatementEnd
DROP TABLE identity_credentials;
