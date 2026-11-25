---
name: tdd-workflow
description: Dirigir pruebas antes de implementación.
---
# tdd-workflow
Leer AGENTS.md, docs/testing/STRATEGY.md, ADR correspondientes y docs/testing/STRATEGY.md antes de actuar.
## Procedimiento
RED reproducible, GREEN mínimo, refactor, integración/race/fuzz/mutación si viable; no falsear pruebas.
Trabajar con alcance autorizado, datos sintéticos y rama aislada. Reportar archivos, comandos realmente ejecutados, resultados, riesgos, alternativas y pendientes. No desplegar ni modificar producción. Si existe contradicción, proponer ADR y detener cambio afectado.

