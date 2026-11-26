# Backend Go — núcleo de inventario

Monolito modular hexagonal (ADR-004), Go 1.26.11 (ADR-170). Estado: **núcleo de inventario BP-011/BP-012 implementado y probado contra PostgreSQL 16 real**, sin API HTTP de negocio todavía (ver ADR-169). No usar con datos reales ni en producción.

## Estructura
```
cmd/bpserver/                 migrate | serve (solo /healthz)
internal/platform/qty/        decimal exacto (ADR-040): sin float, JSON solo strings
internal/platform/postgres/   conexión pgx + migraciones goose embebidas
internal/platform/pgtest/     BD de prueba con guardas anti-producción: esquema nuevo y migrado por test
internal/organization/        organización > empresa > sucursal > almacén (ADR-014)
internal/inventory/domain/    SKU, unidad base, presentaciones, códigos, tipos de movimiento
internal/inventory/application/ casos de uso + puertos (Store/Tx), idempotencia, orden de locks
internal/inventory/adapters/pgstore/ implementación PostgreSQL + pruebas de integración
internal/inventory/jobs/      barrido periódico de reservas vencidas
```

## Reglas que el código garantiza
- **Un escritor por saldo** y descuento condicional `UPDATE ... WHERE physical - reserved - blocked >= q` (ADR-005/006). Dos cajas por la última unidad: una confirma, la otra recibe `ErrInsufficientStock`.
- **Locks en orden estable** (`PlanDeltas`: agrupa líneas del mismo saldo y ordena por almacén, SKU). Reintento solo en `40001/40P01`, máximo 3, y contador observable.
- **Idempotencia durable**: clave empresa + tipo + `operation_id`, hash SHA-256 del payload con cantidades canónicas ("2.50" = "2.5"); las validaciones dependientes del tiempo corren después del reclamo para que un reintento tardío reciba el resultado original; resultado JSON guardado en la misma transacción. Repetición idéntica → mismo resultado sin efectos; payload distinto → `ErrIdempotencyConflict`.
- **Kardex solo-append** (triggers bloquean UPDATE/DELETE/TRUNCATE) y saldos en la misma transacción; las pruebas verifican que Σ kardex = saldo por bucket.
- **Rango almacenable**: máximo 18 dígitos enteros y 6 decimales; un desbordamiento (también al convertir presentaciones o al sumar saldos) es `ErrInvalidArgument`.
- **Unidad base por dimensión** (ADR-171): unit=und, weight=kg|g, length=m|cm, volume=l|ml; validado en dominio y por `CHECK` en la base.
- **Unidad base única**: caja ×24, paquete ×6 y unidad descuentan el mismo stock; factor inmutable y con snapshot en cada movimiento; precisión por SKU.
- **Historial de reservas** solo-append (`inventory_reservation_events`): created/released/consumed/expired con la operación o lote de barrido que lo causó y el saldo reservado resultante; la base admite un `created` y como máximo un terminal por reserva.
- **Reservas** con propósito, dueño y expiración; liberar, consumir o expirar ocurre una sola vez. Una reserva vencida aún no barrida se puede liberar (queda `expired`), no consumir; `bpserver serve` barre las vencidas cada minuto.
- **Transferencias** en dos pasos: el destino solo cambia al recibir; las unidades se conservan.
- **Aislamiento por empresa**: todo filtra por `company_id`; FKs compuestas impiden mezclar empresas.
- **Sin endpoints comerciales** (ADR-169): `cmd/bpserver/routes_test.go` falla si se registra algo distinto de `GET /healthz`.

## Ejecutar
```sh
export BP_TEST_DATABASE_URL='postgres://usuario@127.0.0.1:5432/bp_test?sslmode=disable'  # BD desechable
make test        # gofmt + vet + unitarias + integración con -race
make build       # bin/bpserver
BP_DATABASE_URL=... ./bin/bpserver migrate
```
Sin `BP_TEST_DATABASE_URL` las pruebas de integración se omiten; con `BP_REQUIRE_DB=1` (CI) fallan.

**Guardas de aislamiento (pgtest):** solo desde `go test`; host loopback o socket local; nombre de base `test` o `*_test`; distinta de `BP_DATABASE_URL`; sin esquema desplegado en `public` (tabla `goose_db_version`); cada prueba trabaja en un esquema aleatorio `t_<hex>` que se elimina al terminar. No ejecutes `bpserver migrate` contra la base de pruebas: la guarda la rechazará a partir de entonces.

## Casos de aceptación cubiertos (docs/testing/STRATEGY.md)
| Caso | Prueba |
|---|---|
| INV-01 última unidad / sin sobreventa | `TestINV01LastUnitOnlyOneConfirmation`, `TestINV01ManyUnitsNeverOversold` |
| INV-03 presentaciones y fracciones | `TestINV03PresentationsAndBarcodes`, `TestWeightSKUFractions`, dominio `TestINV03...` |
| INV-07 transferencia (unidades) | `TestINV07TransferConservesUnits` |
| SALE-01 (núcleo) reintento mismo ID | `TestIdempotentReplayAndConflict`, `TestIdempotentConcurrentSameOperation` |
| SEC-01 (núcleo) IDOR entre empresas | `TestCompanyIsolation` |
| BP-006 decimal Go↔PG | `TestDecimalRoundTripThroughPostgres`, paquete `qty` |
| Atomicidad, deadlocks, kardex inmutable, reservas | `TestMultiLineOperationIsAllOrNothing`, `TestOppositeLineOrderDoesNotDeadlock`, `TestKardexIsAppendOnly`, `TestReservation*` |

## Pendiente
Ratificar ADR-173 y piloto de mutación automática; API OpenAPI + HTTP (tras ADR-009 y Fase 03), permisos (ADR-024), auditoría, outbox/cursor (BP-010), lotes/seriales/combos (BP-013), conteos/ubicaciones/bloqueos, valoración (ADR-044), lado Dart/SQLite de BP-006, recepción parcial de transferencias.
