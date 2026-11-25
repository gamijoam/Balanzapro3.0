---
name: postgres-performance
description: Optimizar PostgreSQL sin sacrificar consistencia.
---
# postgres-performance
Leer AGENTS.md, docs/architecture/overview.md, ADR correspondientes y docs/testing/STRATEGY.md antes de actuar.
## Procedimiento
Medir EXPLAIN ANALYZE con datos sintéticos, locks/índices, concurrencia, transacciones y migraciones; antes/después verificable.
Trabajar con alcance autorizado, datos sintéticos y rama aislada. Reportar archivos, comandos realmente ejecutados, resultados, riesgos, alternativas y pendientes. No desplegar ni modificar producción. Si existe contradicción, proponer ADR y detener cambio afectado.

