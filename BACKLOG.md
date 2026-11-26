# Backlog inicial — en curso
| ID | Prioridad | Tarea | Dependencias | Aceptación |
|---|---|---|---|---|
| BP-001 | P0 | Ratificar ADR pendientes críticas 005/006/014/016/024/040/044/051 | Blueprint | Decisión explícita y criterios |
| BP-002 | P0 | Selección versiones y licencias herramientas | 001 | Matriz reproducible y alternativas |
| BP-003 | P0 | Scaffold monorepo, Task, CI DEV | 002 | Build Go/Flutter, lint y tests verdes |
| BP-004 | P0 | Spike Shadcn y tabla/caché 100k | 003 | Matriz plataformas y tiempos |
| BP-005 | P0 | Spike atomicidad/idempotencia PG | 003 | INV-01, SALE-01 |
| BP-006 | P0 | Spike exactitud decimal y OpenAPI | 003 | Go/Dart/PG/SQLite roundtrip |
| BP-007 | P0 | Spike fiscal/periféricos | 002 | Matriz legal oficial, equipo/proveedor |
| BP-008 | P0 | Identidad, empresa, ámbitos, sesiones | 003 | SEC-01 |
| BP-009 | P0 | Módulos, configuración, permisos, auditoría | 008 | SEC-02 y desactivación sin pérdida |
| BP-010 | P0 | Dispositivos y outbox/cursor local | 005,009 | Replay/lag/reconexión |
| BP-011 | P0 | SKU, unidades, presentaciones, códigos | 006,009 | INV-03 |
| BP-012 | P0 | Kardex, saldos, reservas, transferencias | 010,011 | INV-01,06,07 |
| BP-013 | P0 | Lotes, seriales, combos | 012 | INV-02,04,05 |
| BP-014 | P0 | Tasas, moneda, costos, listas | 006,012,001 | FX-01,COST-01,PRICE-01 |
| BP-015 | P0 | Terceros/proveedores, recepción | 012,014 | BUY-01 |
| BP-016 | P0 | Libro y obligaciones mínimos | 006,009 | FIN-01,02 |
| BP-017 | P0 | Factura compra, pago y devolución | 015,016 | BUY-02,03,COST-02 |
| BP-018 | P0 | Carrito POS y checkout | 013,014,016 | SALE-01,02 |
| BP-019 | P0 | Crédito, apartados, pedidos y devolución | 018 | SALE-03,04,05 |
| BP-020 | P0 | Caja/arqueos, gastos, medios pago | 016,018 | FIN-03 |
| BP-021 | P1 | Conciliación, divisas, liquidación | 020 | FIN-04,05 |
| BP-022 | P0 | Impuestos, documentos, series, adaptador fiscal | 007,018 | FISC-01 a 04 y validación legal |
| BP-023 | P0 | Backup/restauración básico (iniciar con 003) | 003 | OPS-03 |
| BP-024 | P0 | Instalador/emparejamiento/licencias | 008,010,023 | OPS-01 |
| BP-025 | P0 | Actualizador, diagnóstico, soporte remoto | 024 | OPS-02,04 |
| BP-026 | P0 | E2E, carga, auditoría y piloto | 017–025 | Evidencia, aceptación humana |
| BP-027 | P1 | Compras avanzadas, promos/analítica | 026 | Casos y alcance aprobados |
| BP-028 | P2 | Restaurantes/producción/híbrido | 026 | ADR específicos y pruebas |
P0 no implica entregar simultáneamente: ordenar por dependencias y vertical funcional. Primer trabajo Codex: BP-001 y propuesta BP-002, no generar todo el ERP.

## Estado al 2026-11-26 (camino corto, ADR-169)
| ID | Estado | Nota |
|---|---|---|
| BP-001 | Parcial | Ratificadas 005, 006, 014, 016, 040. Pendientes: 024 (permisos), 044 (costeo), 051 (precios versionados). |
| BP-002 | Parcial | Backend Go fijado en ADR-170 (Go 1.26.11, govulncheck sin hallazgos); desviaciones en ADR-172 pendientes. Falta Flutter/Dart y distribución. |
| BP-003 | Parcial | Módulo Go, Makefile y CI backend (`.github/workflows/backend.yml`). Falta Flutter y monorepo Dart. |
| BP-005 | Hecho (backend) | INV-01, idempotencia concurrente, atomicidad y orden de locks con PostgreSQL 16 real. |
| BP-006 | Parcial | Go↔PostgreSQL exacto. Falta Dart/SQLite y contrato OpenAPI. |
| BP-011 | Hecho (backend, sin API) | SKU, unidad base, presentaciones, códigos. INV-03. |
| BP-012 | Hecho (backend, sin API) | Kardex, saldos, reservas con historial de eventos y barrido automático, transferencias. INV-01/07. Conteos (INV-06) pendientes. |
| BP-008/009/010 | Pendiente | El núcleo usa el puerto mínimo `Scope`; no exponer API ni usar con datos reales hasta completarlos. |

Evidencia RED/GREEN en `docs/evidence/`.
