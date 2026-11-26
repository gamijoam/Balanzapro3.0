# Evidencia TDD (2026-11-26)
Salidas reales de `go test` del núcleo de inventario, entorno: Linux, Go 1.26.0, PostgreSQL 16.15 local desechable.
- `red-qty.txt`, `red-domain.txt`, `red-inventory-integration.txt`: RED (stubs; las 16 pruebas de integración fallan con `not implemented` tras migrar correctamente).
- `green-qty.txt`, `green-domain.txt`, `green-all.txt`: GREEN.
- `green-race-x5.txt`: primera corrida `-race -count=5`; muestra un fallo de precondición en la propia prueba `TestReservationConsumeAndReleaseRaceHasOneWinner` (5 unidades para 10 rondas). Se corrigió la precondición y se reforzó la aserción del saldo final.
- `green-race-x10.txt`: `-race -count=10` en verde tras la corrección.
- `make-test.txt`: `make test` (gofmt, vet, integración con `-race`).
No incluye benchmarks ni govulncheck (no se pudo ejecutar localmente; corre en CI).

## Correcciones de la revisión de la PR #1
- `red-review-fixes.txt`: RED, 8 pruebas de regresión nuevas que fallan (stubs `Fits` y `SweepExpiredReservations`).
- `green-review-fixes-race-x5.txt`: `-race -count=5` en verde tras las correcciones.
