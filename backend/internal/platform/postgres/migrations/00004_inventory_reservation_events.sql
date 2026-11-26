-- Revisión correctiva PR #1: trazabilidad de reservas (ADR-016/068).
-- Cada cambio del bucket "reserved" queda en un historial solo-append con la
-- operación que lo causó: creación, liberación, consumo o vencimiento (lote
-- del barrido). La base garantiza un "created" y como máximo un evento
-- terminal por reserva. El actor (usuario/terminal) llega con Fase 03.
--
-- Backfill: las reservas anteriores reciben su "created" (y su evento
-- terminal si están cerradas) con operation_type 'migration.backfill' y un
-- operation_id fijo. Su reserved_after es el saldo reservado al migrar (la
-- historia previa no existe); occurred_at usa created_at / closed_at.

-- +goose Up
ALTER TABLE inventory_reservations ADD CONSTRAINT inventory_reservations_company_id_key UNIQUE (company_id, id);

CREATE TABLE inventory_reservation_events (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id     uuid NOT NULL,
    reservation_id uuid NOT NULL,
    event_type     text NOT NULL CHECK (event_type IN ('created', 'released', 'consumed', 'expired')),
    warehouse_id   uuid NOT NULL,
    sku_id         uuid NOT NULL,
    quantity       numeric(24, 6) NOT NULL CHECK (quantity > 0),
    reserved_after numeric(24, 6) NOT NULL CHECK (reserved_after >= 0),
    operation_type text NOT NULL,
    operation_id   uuid NOT NULL,
    occurred_at    timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (company_id, reservation_id) REFERENCES inventory_reservations (company_id, id),
    FOREIGN KEY (company_id, warehouse_id) REFERENCES warehouses (company_id, id),
    FOREIGN KEY (company_id, sku_id) REFERENCES inventory_skus (company_id, id)
);
INSERT INTO inventory_reservation_events (company_id, reservation_id, event_type, warehouse_id, sku_id,
    quantity, reserved_after, operation_type, operation_id, occurred_at)
SELECT r.company_id, r.id, 'created', r.warehouse_id, r.sku_id, r.quantity, COALESCE(b.reserved, 0),
       'migration.backfill', '00000000-0000-4000-8000-000000000004', r.created_at
FROM inventory_reservations r
LEFT JOIN inventory_balances b USING (company_id, warehouse_id, sku_id)
ORDER BY r.created_at, r.id;

INSERT INTO inventory_reservation_events (company_id, reservation_id, event_type, warehouse_id, sku_id,
    quantity, reserved_after, operation_type, operation_id, occurred_at)
SELECT r.company_id, r.id, r.status, r.warehouse_id, r.sku_id, r.quantity, COALESCE(b.reserved, 0),
       'migration.backfill', '00000000-0000-4000-8000-000000000004', COALESCE(r.closed_at, now())
FROM inventory_reservations r
LEFT JOIN inventory_balances b USING (company_id, warehouse_id, sku_id)
WHERE r.status <> 'active'
ORDER BY r.closed_at, r.id;

CREATE UNIQUE INDEX inventory_reservation_events_one_created
    ON inventory_reservation_events (reservation_id) WHERE event_type = 'created';
CREATE UNIQUE INDEX inventory_reservation_events_one_terminal
    ON inventory_reservation_events (reservation_id) WHERE event_type <> 'created';
CREATE INDEX inventory_reservation_events_by_reservation
    ON inventory_reservation_events (company_id, reservation_id, id);

-- +goose StatementBegin
CREATE FUNCTION inventory_reservation_events_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'inventory_reservation_events is append-only (cannot %)', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER inventory_reservation_events_no_update_delete
    BEFORE UPDATE OR DELETE ON inventory_reservation_events
    FOR EACH ROW EXECUTE FUNCTION inventory_reservation_events_append_only();
CREATE TRIGGER inventory_reservation_events_no_truncate
    BEFORE TRUNCATE ON inventory_reservation_events
    FOR EACH STATEMENT EXECUTE FUNCTION inventory_reservation_events_append_only();

-- +goose Down
DROP TABLE inventory_reservation_events;
DROP FUNCTION inventory_reservation_events_append_only();
ALTER TABLE inventory_reservations DROP CONSTRAINT inventory_reservations_company_id_key;
