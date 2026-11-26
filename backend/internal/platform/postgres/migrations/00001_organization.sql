-- ADR-014: organización > empresa > sucursal > almacén. Ubicaciones dentro
-- del almacén llegan con BP-013/conteos. Las claves compuestas
-- (company_id, id) permiten que las tablas hijas exijan, por FK, que todo
-- pertenezca a la misma empresa (aislamiento a nivel de base de datos).

-- +goose Up
CREATE TABLE organizations (
    id         uuid PRIMARY KEY,
    name       text NOT NULL CHECK (btrim(name) <> ''),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE companies (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations (id),
    name            text NOT NULL CHECK (btrim(name) <> ''),
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE branches (
    id         uuid PRIMARY KEY,
    company_id uuid NOT NULL REFERENCES companies (id),
    name       text NOT NULL CHECK (btrim(name) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (company_id, id)
);

CREATE TABLE warehouses (
    id         uuid PRIMARY KEY,
    company_id uuid NOT NULL,
    branch_id  uuid NOT NULL,
    name       text NOT NULL CHECK (btrim(name) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (company_id, id),
    FOREIGN KEY (company_id, branch_id) REFERENCES branches (company_id, id)
);

-- +goose Down
DROP TABLE warehouses;
DROP TABLE branches;
DROP TABLE companies;
DROP TABLE organizations;
