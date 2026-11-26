-- BP-009 subtarea 3 (#84): roles por ámbito (ADR-024) y autorizaciones de
-- supervisor (ADR-025). Las FK compuestas impiden en la base que un rol,
-- usuario, empresa, sucursal o almacén de otra organización o empresa entre en
-- una asignación (SEC-01). Una asignación exige membresía (ADR-014).
-- Las autorizaciones son inmutables salvo used_at, que se fija una sola vez
-- (SEC-02). Límites en NUMERIC(24,6), exactos.

-- +goose Up
CREATE TABLE permission_roles (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations (id),
    code            text NOT NULL CHECK (code ~ '^[a-z][a-z0-9_]{0,62}$'),
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, code),
    UNIQUE (organization_id, id)
);

CREATE TABLE permission_role_grants (
    organization_id uuid NOT NULL,
    role_id         uuid NOT NULL,
    permission      text NOT NULL CHECK (permission ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'
                                        AND length(permission) <= 128),
    -- NULL means unlimited.
    limit_value     numeric(24, 6) NULL CHECK (limit_value >= 0),
    PRIMARY KEY (role_id, permission),
    FOREIGN KEY (organization_id, role_id) REFERENCES permission_roles (organization_id, id)
);

CREATE TABLE permission_assignments (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id uuid NOT NULL,
    user_id         uuid NOT NULL,
    role_id         uuid NOT NULL,
    company_id      uuid NOT NULL,
    scope_level     text NOT NULL CHECK (scope_level IN ('company', 'branch', 'warehouse')),
    branch_id       uuid NULL,
    warehouse_id    uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CHECK ((scope_level = 'branch') = (branch_id IS NOT NULL)),
    CHECK ((scope_level = 'warehouse') = (warehouse_id IS NOT NULL)),
    FOREIGN KEY (organization_id, user_id) REFERENCES identity_users (organization_id, id),
    FOREIGN KEY (organization_id, role_id) REFERENCES permission_roles (organization_id, id),
    FOREIGN KEY (organization_id, company_id) REFERENCES companies (organization_id, id),
    FOREIGN KEY (user_id, company_id) REFERENCES identity_memberships (user_id, company_id),
    FOREIGN KEY (company_id, branch_id) REFERENCES branches (company_id, id),
    FOREIGN KEY (company_id, warehouse_id) REFERENCES warehouses (company_id, id)
);
CREATE UNIQUE INDEX permission_assignments_unique
    ON permission_assignments (user_id, role_id, company_id, scope_level, branch_id, warehouse_id) NULLS NOT DISTINCT;
CREATE INDEX permission_assignments_by_user ON permission_assignments (organization_id, user_id, company_id);

CREATE TABLE supervisor_authorizations (
    id                  uuid PRIMARY KEY,
    organization_id     uuid NOT NULL,
    company_id          uuid NOT NULL,
    requester_user_id   uuid NOT NULL,
    supervisor_user_id  uuid NOT NULL,
    permission          text NOT NULL CHECK (permission ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'
                                            AND length(permission) <= 128),
    target_branch_id    uuid NULL,
    target_warehouse_id uuid NULL,
    amount              numeric(24, 6) NULL,
    entity_ref          text NOT NULL CHECK (btrim(entity_ref) <> '' AND length(entity_ref) <= 256),
    terminal_id         uuid NOT NULL,
    session_id          uuid NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz NOT NULL,
    used_at             timestamptz NULL,
    CHECK (supervisor_user_id <> requester_user_id),
    FOREIGN KEY (organization_id, company_id) REFERENCES companies (organization_id, id),
    FOREIGN KEY (organization_id, requester_user_id) REFERENCES identity_users (organization_id, id),
    FOREIGN KEY (organization_id, supervisor_user_id) REFERENCES identity_users (organization_id, id),
    FOREIGN KEY (company_id, target_branch_id) REFERENCES branches (company_id, id),
    FOREIGN KEY (company_id, target_warehouse_id) REFERENCES warehouses (company_id, id)
);

-- +goose StatementBegin
CREATE FUNCTION supervisor_authorizations_single_use() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.used_at IS NOT NULL OR NEW.used_at IS NULL
       OR (to_jsonb(NEW) - 'used_at') IS DISTINCT FROM (to_jsonb(OLD) - 'used_at') THEN
        RAISE EXCEPTION 'supervisor_authorizations: only used_at may be set, once'
            USING ERRCODE = 'restrict_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER supervisor_authorizations_single_use BEFORE UPDATE ON supervisor_authorizations
    FOR EACH ROW EXECUTE FUNCTION supervisor_authorizations_single_use();
CREATE TRIGGER supervisor_authorizations_no_delete BEFORE DELETE ON supervisor_authorizations
    FOR EACH ROW EXECUTE FUNCTION platform_append_only();
CREATE TRIGGER supervisor_authorizations_no_truncate BEFORE TRUNCATE ON supervisor_authorizations
    FOR EACH STATEMENT EXECUTE FUNCTION platform_append_only();

-- +goose Down
DROP TABLE supervisor_authorizations;
DROP FUNCTION supervisor_authorizations_single_use();
DROP TABLE permission_assignments;
DROP TABLE permission_role_grants;
DROP TABLE permission_roles;
