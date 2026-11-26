-- BP-008 subtarea 1 (#78): esquema de identidad (ADR-014).
-- Un usuario pertenece a una organización y actúa en empresas solo mediante
-- membresías. Las FK compuestas (organization_id, …) impiden en la base que un
-- usuario quede ligado a una empresa de otra organización (SEC-01).
-- Fuera de alcance: credenciales y sesiones (ADR-026, subtarea posterior) y
-- roles por ámbito (ADR-024, BP-009). Los usuarios se deshabilitan, no se borran.

-- +goose Up
ALTER TABLE companies ADD CONSTRAINT companies_organization_id_id_key UNIQUE (organization_id, id);

CREATE TABLE identity_users (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations (id),
    -- Normalized by the application: lowercase, trimmed, no whitespace.
    login           text NOT NULL CHECK (login <> '' AND login = lower(login) AND login !~ '\s'
                                         AND length(login) <= 64),
    display_name    text NOT NULL CHECK (btrim(display_name) <> ''),
    status          text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    disabled_at     timestamptz NULL,
    CHECK ((status = 'disabled') = (disabled_at IS NOT NULL)),
    UNIQUE (organization_id, login),
    UNIQUE (organization_id, id)
);

CREATE TABLE identity_memberships (
    organization_id uuid NOT NULL,
    user_id         uuid NOT NULL,
    company_id      uuid NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, company_id),
    FOREIGN KEY (organization_id, user_id) REFERENCES identity_users (organization_id, id),
    FOREIGN KEY (organization_id, company_id) REFERENCES companies (organization_id, id)
);
CREATE INDEX identity_memberships_by_company ON identity_memberships (company_id, user_id);

-- +goose Down
DROP TABLE identity_memberships;
DROP TABLE identity_users;
ALTER TABLE companies DROP CONSTRAINT companies_organization_id_id_key;
