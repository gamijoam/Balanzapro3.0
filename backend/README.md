# Backend Go — núcleo de inventario

Monolito modular hexagonal (ADR-004). Estado: **núcleo de inventario BP-011/BP-012 implementado y probado contra PostgreSQL 16 real**, sin API HTTP de negocio todavía (ver ADR-169). No usar con datos reales ni en producción.

## Estructura
```
cmd/bpserver/                 migrate | serve (solo /healthz)
internal/platform/qty/        decimal exacto (ADR-040): sin float, JSON solo strings
internal/platform/postgres/   conexión pgx + migraciones goose embebidas
internal/platform/pgtest/     BD de prueba: esquema nuevo y migrado por test
internal/organization/        organización > empresa > sucursal > almacén (ADR-014)
internal/inventory/domain/    SKU, unidad base, presentaciones, códigos, tipos de movimiento
internal/inventory/application/ casos de uso + puertos (Store/Tx), idempotencia, orden de locks
internal/inventory/adapters/pgstore/ implementación PostgreSQL + pruebas de integración
```

## Reglas que el código garantiza
- **Un escritor por saldo** y descuento condicional `UPDATE ... WHERE physical - reserved - blocked >= q` (ADR-005/006). Dos cajas por la última unidad: una confirma, la otra recibe `ErrInsufficientStock`.
- **Locks en orden estable** (`PlanDeltas`: agrupa líneas del mismo saldo y ordena por almacén, SKU). Reintento solo en `40001/40P01`, máximo 3, y contador observable.
- **Idempotencia durable**: clave empresa + tipo + `operation_id`, hash SHA-256 del payload, resultado JSON guardado en la misma transacción. Repetición idéntica → mismo resultado sin efectos; payload distinto → `ErrIdempotencyConflict`.
- **Kardex solo-append** (triggers bloquean UPDATE/DELETE/TRUNCATE) y saldos en la misma transacción; las pruebas verifican que Σ kardex = saldo por bucket.
- **Unidad base única**: caja ×24, paquete ×6 y unidad descuentan el mismo stock; factor inmutable y con snapshot en cada movimiento; precisión por SKU.
- **Reservas** con propósito, dueño y expiración; liberar, consumir o expirar ocurre una sola vez.
- **Transferencias** en dos pasos: el destino solo cambia al recibir; las unidades se conservan.
- **Aislamiento por empresa**: todo filtra por `company_id`; FKs compuestas impiden mezclar empresas.

## Ejecutar
```sh
export BP_TEST_DATABASE_URL='postgres://usuario@127.0.0.1:5432/bp_test?sslmode=disable'  # BD desechable
make test        # gofmt + vet + unitarias + integración con -race
make build       # bin/bpserver
BP_DATABASE_URL=... ./bin/bpserver migrate
```
Sin `BP_TEST_DATABASE_URL` las pruebas de integración se omiten; con `BP_REQUIRE_DB=1` (CI) fallan.

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
API OpenAPI + HTTP (tras ADR-009 y Fase 03), permisos (ADR-024), auditoría, outbox/cursor (BP-010), lotes/seriales/combos (BP-013), conteos/ubicaciones/bloqueos, valoración (ADR-044), lado Dart/SQLite de BP-006, recepción parcial de transferencias.
