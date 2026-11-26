-- Revisión correctiva PR #1: la unidad base debe pertenecer a la dimensión
-- del SKU (ADR-060/061; catálogo inicial ADR-171). Espejo de
-- domain.baseUnits: defensa en profundidad si alguien escribe sin el dominio.
--
-- Compatible con datos previos (v2 admitía texto libre): normaliza a
-- minúsculas sin espacios, añade la restricción NOT VALID (las filas nuevas
-- ya la cumplen obligatoriamente) y la valida solo si todas las filas
-- existentes la cumplen; si no, deja un WARNING y la restricción sin validar
-- hasta revisar esos SKUs (docs/runbooks/incident-response.md).

-- +goose Up
UPDATE inventory_skus SET base_unit = lower(btrim(base_unit))
WHERE base_unit <> lower(btrim(base_unit));

ALTER TABLE inventory_skus
    ADD CONSTRAINT inventory_skus_unit_dimension CHECK (
        (quantity_kind, base_unit) IN (
            ('unit', 'und'),
            ('weight', 'kg'), ('weight', 'g'),
            ('length', 'm'), ('length', 'cm'),
            ('volume', 'l'), ('volume', 'ml'))) NOT VALID;

-- +goose StatementBegin
DO $$
BEGIN
    ALTER TABLE inventory_skus VALIDATE CONSTRAINT inventory_skus_unit_dimension;
EXCEPTION WHEN check_violation THEN
    RAISE WARNING 'inventory_skus: hay SKUs con unidad base fuera de su dimensión; inventory_skus_unit_dimension queda NOT VALID hasta corregirlos';
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- La normalización a minúsculas no se revierte (no hay forma fiable de
-- recuperar el texto original y no cambia el significado).
ALTER TABLE inventory_skus DROP CONSTRAINT inventory_skus_unit_dimension;
