-- BP-009 / ADR-174: allow role assignments at organization level.
-- Existing company/branch/warehouse assignments retain their membership FK.
-- The future authorization adapter must independently verify membership
-- before applying an organization assignment to a company target.
-- This migration alone grants no new permissions through the application.

-- +goose Up
ALTER TABLE permission_assignments
    DROP CONSTRAINT permission_assignments_scope_level_check,
    ALTER COLUMN company_id DROP NOT NULL,
    ADD CONSTRAINT permission_assignments_scope_level_check
        CHECK (scope_level IN ('organization', 'company', 'branch', 'warehouse')),
    ADD CONSTRAINT permission_assignments_organization_scope_check
        CHECK ((scope_level = 'organization') = (company_id IS NULL));

-- The existing branch/warehouse checks prohibit those IDs for organization
-- assignments; the existing NULLS NOT DISTINCT unique index also prohibits
-- duplicate organization-level assignments for the same user and role.
-- User/role composite FKs continue to enforce organization isolation.

-- +goose Down
-- Deliberately fail if organization assignments exist instead of deleting
-- authorized records. Reconcile them explicitly before rolling back.
ALTER TABLE permission_assignments
    ALTER COLUMN company_id SET NOT NULL,
    DROP CONSTRAINT permission_assignments_organization_scope_check,
    DROP CONSTRAINT permission_assignments_scope_level_check,
    ADD CONSTRAINT permission_assignments_scope_level_check
        CHECK (scope_level IN ('company', 'branch', 'warehouse'));
