-- +goose Up
CREATE TABLE identity_attempt_budgets (
    organization_id uuid NOT NULL REFERENCES organizations(id),
    budget_key bytea NOT NULL CHECK (octet_length(budget_key) = 32),
    kind text NOT NULL CHECK (kind IN ('password', 'pin', 'totp')),
    max_attempts integer NOT NULL CHECK (max_attempts BETWEEN 1 AND 100),
    window_us bigint NOT NULL CHECK (window_us BETWEEN 1000000 AND 86400000000),
    cooldown_us bigint NOT NULL CHECK (cooldown_us BETWEEN 1000000 AND 86400000000),
    used integer NOT NULL DEFAULT 0 CHECK (used >= 0 AND used <= max_attempts),
    window_start timestamptz,
    last_reserved timestamptz,
    blocked_until timestamptz,
    PRIMARY KEY (organization_id, budget_key, kind),
    CONSTRAINT attempt_state_coherent CHECK (
        (used = 0 AND window_start IS NULL AND last_reserved IS NULL AND blocked_until IS NULL)
        OR
        (used > 0 AND window_start IS NOT NULL AND last_reserved IS NOT NULL
          AND isfinite(window_start) AND isfinite(last_reserved)
          AND last_reserved >= window_start
          AND last_reserved < window_start + window_us * interval '1 microsecond'
          AND (
            (used < max_attempts AND blocked_until IS NULL)
            OR
            (used = max_attempts AND blocked_until IS NOT NULL
              AND isfinite(blocked_until)
              AND blocked_until = greatest(
                window_start + window_us * interval '1 microsecond',
                last_reserved + cooldown_us * interval '1 microsecond'
              ))
          ))
    )
);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    LOCK TABLE identity_attempt_budgets IN ACCESS EXCLUSIVE MODE;
    IF EXISTS (SELECT 1 FROM identity_attempt_budgets) THEN
        RAISE EXCEPTION 'cannot roll back recorded authentication attempt budgets';
    END IF;
END;
$$;
-- +goose StatementEnd
DROP TABLE identity_attempt_budgets;
