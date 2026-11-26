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

## Revisión correctiva del PR #1 (segunda ronda)
| Hallazgo | RED | GREEN |
|---|---|---|
| 24 vulnerabilidades stdlib (Go 1.26.0) | `govulncheck-ci-run2-go1.26.0.txt` (CI run 45729977712) | `govulncheck-go1.26.11.txt` (No vulnerabilities found) |
| Unidad base no ligada a la dimensión | `red-units.txt` | `green-units.txt` |
| Reservas sin historial trazable | `red-reservation-trace.txt` | `green-reservation-trace.txt` |
| Pruebas ejecutables contra cualquier BD | `red-test-isolation.txt`; la guarda detectó una BD de pruebas migrada en `public` (`guard-caught-migrated-public.txt`) | `green-test-isolation.txt` |
| Rollback de migraciones sin probar | `red-migrations-roundtrip.txt` | `green-migrations-roundtrip.txt` |
| Barrido de reservas sin pruebas | `red-sweeper.txt` | `green-sweeper.txt` |
| Calidad de pruebas (mutación manual) | primera corrida 11/14 (M10, M11 sobreviven) | `manual-mutation.txt` 14/14 |
Validación final: `final-*.txt`.

## Segunda revisión independiente (code-review sobre 1fe1524..5d0ce74)
8 hallazgos: migración 00003 con datos previos, backfill de 00004, historial de una reserva sin eventos, barrido que se detenía con el primer error de empresa, pánico con Interval ≤ 0 y primer barrido tardío, comparación de hosts loopback en la guarda, limpieza de la BD de sondeo y ADR-171 sin ratificar (ratificada el 2026-11-26).
RED: `red-review2.txt` (6 paquetes fallan, incluido el pánico de NewTicker). GREEN: `green-review2.txt`.
