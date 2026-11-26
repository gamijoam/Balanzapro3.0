-- BP-009 subtarea 1 (#82): esquema de módulos (ADR-023), configuración
-- versionada (ADR-027) y eventos de auditoría (docs/security/security.md).
-- Nada se borra: los módulos se desactivan, la configuración se versiona y la
-- auditoría es solo-append. Las reglas (dependencias, licencias, niveles
-- permitidos por clave, claves de seguridad no anulables) viven en el dominio.

-- +goose Up

-- Activation state per company. History of changes goes to audit_events.
CREATE TABLE company_modules (
    company_id   uuid NOT NULL REFERENCES companies (id),
    module_id    text NOT NULL CHECK (module_id ~ '^[a-z][a-z0-9_]{1,62}$'),
    status       text NOT NULL CHECK (status IN ('active', 'inactive')),
    version      bigint NOT NULL DEFAULT 1 CHECK (version >= 1),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (company_id, module_id)
);

-- Versioned configuration: every change is a new row; the current value of
-- (key, level, scope) is its highest version.
CREATE TABLE config_entries (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    key             text NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$' AND length(key) <= 128),
    scope_level     text NOT NULL CHECK (scope_level IN
                        ('global', 'organization', 'company', 'branch', 'terminal', 'user')),
    organization_id uuid NULL REFERENCES organizations (id),
    scope_id        uuid NULL,
    version         integer NOT NULL CHECK (version >= 1),
    value           jsonb NOT NULL,
    actor_user_id   uuid NULL,
    reason          text NOT NULL CHECK (btrim(reason) <> ''),
    created_at      timestamptz NOT NULL DEFAULT now(),
    CHECK ((scope_level = 'global') = (organization_id IS NULL)),
    CHECK ((scope_level = 'global') = (scope_id IS NULL)),
    CHECK (scope_level <> 'organization' OR scope_id = organization_id)
);
CREATE UNIQUE INDEX config_entries_version
    ON config_entries (key, scope_level, COALESCE(scope_id, '00000000-0000-0000-0000-000000000000'::uuid), version);

-- Audit trail: actor, action, entity, scope, sanitized before/after, reason,
-- time and correlation. Never stores secrets.
CREATE TABLE audit_events (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id uuid NULL REFERENCES organizations (id),
    company_id      uuid NULL,
    actor_user_id   uuid NULL,
    action          text NOT NULL CHECK (btrim(action) <> '' AND length(action) <= 128),
    entity_type     text NOT NULL CHECK (btrim(entity_type) <> ''),
    entity_id       text NOT NULL,
    before          jsonb NULL,
    after           jsonb NULL,
    reason          text NULL,
    correlation_id  uuid NOT NULL,
    occurred_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (company_id IS NULL OR organization_id IS NOT NULL),
    FOREIGN KEY (organization_id, company_id) REFERENCES companies (organization_id, id)
);
CREATE INDEX audit_events_by_company ON audit_events (company_id, occurred_at);
CREATE INDEX audit_events_by_correlation ON audit_events (correlation_id);

-- +goose StatementBegin
CREATE FUNCTION platform_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% is append-only (cannot %)', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER config_entries_append_only BEFORE UPDATE OR DELETE ON config_entries
    FOR EACH ROW EXECUTE FUNCTION platform_append_only();
CREATE TRIGGER config_entries_no_truncate BEFORE TRUNCATE ON config_entries
    FOR EACH STATEMENT EXECUTE FUNCTION platform_append_only();
CREATE TRIGGER audit_events_append_only BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION platform_append_only();
CREATE TRIGGER audit_events_no_truncate BEFORE TRUNCATE ON audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION platform_append_only();
-- Modules are deactivated, never removed (ADR-023).
CREATE TRIGGER company_modules_no_delete BEFORE DELETE ON company_modules
    FOR EACH ROW EXECUTE FUNCTION platform_append_only();
CREATE TRIGGER company_modules_no_truncate BEFORE TRUNCATE ON company_modules
    FOR EACH STATEMENT EXECUTE FUNCTION platform_append_only();

-- +goose Down
DROP TABLE audit_events;
DROP TABLE config_entries;
DROP TABLE company_modules;
DROP FUNCTION platform_append_only();
