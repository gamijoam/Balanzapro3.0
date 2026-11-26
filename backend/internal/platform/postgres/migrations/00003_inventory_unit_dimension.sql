-- Revisión correctiva PR #1: la unidad base debe pertenecer a la dimensión
-- del SKU (ADR-060/061; catálogo inicial ADR-171). Espejo de
-- domain.baseUnits: defensa en profundidad si alguien escribe sin el dominio.

-- +goose Up
ALTER TABLE inventory_skus
    ADD CONSTRAINT inventory_skus_unit_dimension CHECK (
        (quantity_kind, base_unit) IN (
            ('unit', 'und'),
            ('weight', 'kg'), ('weight', 'g'),
            ('length', 'm'), ('length', 'cm'),
            ('volume', 'l'), ('volume', 'ml')));

-- +goose Down
ALTER TABLE inventory_skus DROP CONSTRAINT inventory_skus_unit_dimension;
