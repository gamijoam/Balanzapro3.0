-- Núcleo de inventario (BP-011/BP-012; ADR-005, 006, 016, 040, 059–068).
-- Cantidades exactas NUMERIC(24,6) en unidad base del SKU. El kardex es
-- solo-append (trigger) y los saldos se actualizan en la misma transacción.
-- Valoración (costos) queda fuera hasta ratificar ADR-044.

-- +goose Up
CREATE TABLE inventory_skus (
    id            uuid PRIMARY KEY,
    company_id    uuid NOT NULL REFERENCES companies (id),
    code          text NOT NULL CHECK (btrim(code) <> ''),
    name          text NOT NULL CHECK (btrim(name) <> ''),
    quantity_kind text NOT NULL CHECK (quantity_kind IN ('unit', 'weight', 'length', 'volume')),
    base_unit     text NOT NULL CHECK (btrim(base_unit) <> ''),
    precision     smallint NOT NULL CHECK (precision BETWEEN 0 AND 6),
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (company_id, id),
    UNIQUE (company_id, code)
);

CREATE TABLE inventory_presentations (
    id         uuid PRIMARY KEY,
    company_id uuid NOT NULL,
    sku_id     uuid NOT NULL,
    name       text NOT NULL CHECK (btrim(name) <> ''),
    factor     numeric(24, 6) NOT NULL CHECK (factor > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (company_id, id),
    UNIQUE (company_id, sku_id, id),
    UNIQUE (company_id, sku_id, name),
    FOREIGN KEY (company_id, sku_id) REFERENCES inventory_skus (company_id, id)
);

CREATE TABLE inventory_barcodes (
    company_id      uuid NOT NULL,
    code            text NOT NULL CHECK (code <> '' AND length(code) <= 64),
    sku_id          uuid NOT NULL,
    presentation_id uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (company_id, code),
    FOREIGN KEY (company_id, sku_id) REFERENCES inventory_skus (company_id, id),
    FOREIGN KEY (company_id, sku_id, presentation_id)
        REFERENCES inventory_presentations (company_id, sku_id, id)
);

CREATE TABLE inventory_balances (
    company_id   uuid NOT NULL,
    warehouse_id uuid NOT NULL,
    sku_id       uuid NOT NULL,
    physical     numeric(24, 6) NOT NULL DEFAULT 0,
    reserved     numeric(24, 6) NOT NULL DEFAULT 0,
    blocked      numeric(24, 6) NOT NULL DEFAULT 0,
    in_transit   numeric(24, 6) NOT NULL DEFAULT 0,
    version      bigint NOT NULL DEFAULT 0,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (company_id, warehouse_id, sku_id),
    FOREIGN KEY (company_id, warehouse_id) REFERENCES warehouses (company_id, id),
    FOREIGN KEY (company_id, sku_id) REFERENCES inventory_skus (company_id, id),
    CHECK (physical >= 0 AND reserved >= 0 AND blocked >= 0 AND in_transit >= 0),
    CHECK (reserved + blocked <= physical)
);

-- Idempotencia durable (ADR-006): misma clave + mismo hash → mismo resultado.
CREATE TABLE inventory_operations (
    company_id     uuid NOT NULL REFERENCES companies (id),
    operation_type text NOT NULL,
    operation_id   uuid NOT NULL,
    request_hash   text NOT NULL,
    result         jsonb NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (company_id, operation_type, operation_id)
);

CREATE TABLE inventory_movements (
    id                    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id            uuid NOT NULL,
    operation_type        text NOT NULL,
    operation_id          uuid NOT NULL,
    line_no               integer NOT NULL CHECK (line_no >= 0),
    movement_type         text NOT NULL CHECK (movement_type IN (
        'opening', 'adjustment_in', 'adjustment_out', 'issue', 'reservation_consume',
        'transfer_out', 'transfer_transit_in', 'transfer_transit_out', 'transfer_in')),
    bucket                text NOT NULL CHECK (bucket IN ('physical', 'in_transit')),
    direction             smallint NOT NULL CHECK (direction IN (-1, 1)),
    warehouse_id          uuid NOT NULL,
    sku_id                uuid NOT NULL,
    quantity              numeric(24, 6) NOT NULL CHECK (quantity > 0),
    balance_after         numeric(24, 6) NOT NULL CHECK (balance_after >= 0),
    presentation_id       uuid NULL,
    presentation_quantity numeric(24, 6) NULL,
    presentation_factor   numeric(24, 6) NULL,
    reason                text NULL,
    reference_id          uuid NULL,
    created_at            timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (company_id, operation_type, operation_id)
        REFERENCES inventory_operations (company_id, operation_type, operation_id),
    FOREIGN KEY (company_id, warehouse_id) REFERENCES warehouses (company_id, id),
    FOREIGN KEY (company_id, sku_id) REFERENCES inventory_skus (company_id, id),
    CHECK ((presentation_id IS NULL) = (presentation_factor IS NULL)),
    CHECK ((presentation_id IS NULL) = (presentation_quantity IS NULL))
);
CREATE INDEX inventory_movements_kardex_idx
    ON inventory_movements (company_id, warehouse_id, sku_id, id);

-- +goose StatementBegin
CREATE FUNCTION inventory_movements_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'inventory_movements is append-only (kardex cannot be % )', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER inventory_movements_no_update_delete
    BEFORE UPDATE OR DELETE ON inventory_movements
    FOR EACH ROW EXECUTE FUNCTION inventory_movements_append_only();

CREATE TRIGGER inventory_movements_no_truncate
    BEFORE TRUNCATE ON inventory_movements
    FOR EACH STATEMENT EXECUTE FUNCTION inventory_movements_append_only();

-- Un factor de presentación ya usado no se cambia (ADR-061): se crea otra.
-- +goose StatementBegin
CREATE FUNCTION inventory_presentation_factor_immutable() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.factor <> OLD.factor OR NEW.sku_id <> OLD.sku_id THEN
        RAISE EXCEPTION 'presentation factor and SKU are immutable'
            USING ERRCODE = 'restrict_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER inventory_presentations_factor_immutable
    BEFORE UPDATE ON inventory_presentations
    FOR EACH ROW EXECUTE FUNCTION inventory_presentation_factor_immutable();

CREATE TABLE inventory_reservations (
    id                  uuid PRIMARY KEY,
    company_id          uuid NOT NULL,
    warehouse_id        uuid NOT NULL,
    sku_id              uuid NOT NULL,
    quantity            numeric(24, 6) NOT NULL CHECK (quantity > 0),
    purpose             text NOT NULL CHECK (btrim(purpose) <> ''),
    owner_ref           text NOT NULL DEFAULT '',
    status              text NOT NULL CHECK (status IN ('active', 'released', 'consumed', 'expired')),
    expires_at          timestamptz NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    closed_at           timestamptz NULL,
    CHECK ((status = 'active') = (closed_at IS NULL)),
    FOREIGN KEY (company_id, warehouse_id) REFERENCES warehouses (company_id, id),
    FOREIGN KEY (company_id, sku_id) REFERENCES inventory_skus (company_id, id)
);
CREATE INDEX inventory_reservations_due_idx
    ON inventory_reservations (company_id, expires_at) WHERE status = 'active';

CREATE TABLE inventory_transfers (
    id                uuid PRIMARY KEY,
    company_id        uuid NOT NULL,
    from_warehouse_id uuid NOT NULL,
    to_warehouse_id   uuid NOT NULL,
    status            text NOT NULL CHECK (status IN ('shipped', 'received')),
    created_at        timestamptz NOT NULL DEFAULT now(),
    received_at       timestamptz NULL,
    CHECK (from_warehouse_id <> to_warehouse_id),
    UNIQUE (company_id, id),
    FOREIGN KEY (company_id, from_warehouse_id) REFERENCES warehouses (company_id, id),
    FOREIGN KEY (company_id, to_warehouse_id) REFERENCES warehouses (company_id, id)
);

CREATE TABLE inventory_transfer_lines (
    company_id  uuid NOT NULL,
    transfer_id uuid NOT NULL,
    line_no     integer NOT NULL,
    sku_id      uuid NOT NULL,
    quantity    numeric(24, 6) NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (company_id, transfer_id, line_no),
    FOREIGN KEY (company_id, transfer_id) REFERENCES inventory_transfers (company_id, id),
    FOREIGN KEY (company_id, sku_id) REFERENCES inventory_skus (company_id, id)
);

-- +goose Down
DROP TABLE inventory_transfer_lines;
DROP TABLE inventory_transfers;
DROP TABLE inventory_reservations;
DROP TRIGGER inventory_presentations_factor_immutable ON inventory_presentations;
DROP FUNCTION inventory_presentation_factor_immutable();
DROP TABLE inventory_movements;
DROP FUNCTION inventory_movements_append_only();
DROP TABLE inventory_operations;
DROP TABLE inventory_balances;
DROP TABLE inventory_barcodes;
DROP TABLE inventory_presentations;
DROP TABLE inventory_skus;
