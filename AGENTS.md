# BalanzaPro3.0 — instrucciones obligatorias para agentes
## Antes de cambiar
Leer README, ROADMAP, BACKLOG, TECH_STACK, docs/decisions/README y ADR del alcance, reglas de negocio y seguridad. Confirmar criterio de aceptación y decisiones pendientes; pedir ratificación si afectan comportamiento. No inventar requisitos ni promover propuestas a aprobadas. Nunca tocar otras instalaciones en /opt, producción, bases reales, secretos ni desplegar sin autorización.
## Arquitectura
Go modular hexagonal; Flutter/Dart con design system Shadcn encapsulado; PostgreSQL autoridad única por ámbito; SQLite/Drift caché y borradores; OpenAPI como contrato. Dependencias entre módulos por puertos; no importaciones cíclicas, acceso directo a BD desde UI, lógica comercial duplicada, ni microservicios por defecto. Dinero y cantidades exactas; strings decimales en API. Operaciones confirmadas inmutables, compensaciones y auditoría. Stock, kardex, costos y obligaciones compatibles se registran en una transacción. Idempotencia durable con hash del payload; no repetir pagos/fiscal de resultado desconocido. Permisos servidor y ámbito, denegar por defecto.
## Flujo obligatorio
Plan pequeño → casos de aceptación → prueba RED reproducible → código mínimo GREEN → refactor → pruebas integración/concurrencia → revisión independiente → PR. No adaptar tests para ocultar fallos. Incluir evidencia de comandos, resultados y fallos; no afirmar ejecución no realizada. Migraciones compatibles y recuperación documentada. Nunca introducir librería sin justificar licencia, mantenimiento, seguridad, soporte de plataformas, benchmark cuando proceda y alternativa. Guardar decisión.
## Coordinación
Un agente por rama/worktree y tarea; cambios de contratos coordinados. No fusionar ni publicar automáticamente. No usar datos reales. Validar lint, tests, vulnerabilidades y secretos en CI cuando exista. No comprometer tokens, credenciales, llaves, dumps ni datos sensibles.
## Política de Pull Requests y flujo en GitHub
Obligatoria para agentes y desarrolladores. Complementa "Flujo obligatorio" y "Coordinación"; no sustituye ninguna regla anterior.
- **Unidad de PR:** 1 PR = 1 unidad lógica revisable: una subtarea de un `BP-XXX` o un PR exclusivo de ratificación de ADRs. Prohibido mezclar varios ítems `BP-*` en un PR, o mezclar el debate de ADRs con una implementación masiva.
- **Tamaño objetivo:** entre 200 y 500 líneas de código productivo, sin contar `go.sum`, migraciones generadas, snapshots ni fixtures de tests. Si una tarea va a superar ese tamaño, se divide en subtareas **antes** de empezar a codificar, y cada una se registra como checkbox en el Issue del `BP-XXX`.
- **Cómo dividir:**
  - *Módulos fundacionales:* PRs independientes para (1) migraciones y esquema PostgreSQL, (2) dominio + puertos + tests unitarios, (3) adaptador `pgx` + tests de integración y (4) contrato OpenAPI + handlers HTTP.
  - *Funcionalidades sobre una base existente:* un caso de uso o endpoint vertical completo por PR.
  - *Dependencias:* si un PR depende de otro aún abierto, usar *stacked PRs* (base en la rama previa) y cambiar la base a `main` tras el merge del anterior.
- **Formato y trazabilidad:**
  - El título sigue Conventional Commits: `feat(scope): …`, `fix(scope): …`, `docs(adr): …`, `chore(scope): …`, `refactor(scope): …`, `test(scope): …`.
  - Referencia obligatoria al Issue (`Part of #N` o `Closes #N`) y al código `BP-XXX` o `ADR-XXX`.
  - Plantilla `.github/PULL_REQUEST_TEMPLATE.md` completa.
  - Labels de área y tipo; milestone del hito del ROADMAP.
- **Issues:** cada `BP-XXX` del BACKLOG se abre con la plantilla "Tarea del BACKLOG" y cada ratificación con "Ratificación de ADR". Una tarea cuyas ADR estén pendientes lleva el label `blocked-by-adr` y no se implementa hasta ratificarlas.
- **Merge:** *squash and merge* hacia `main` protegida, con el check `backend` en verde y la revisión humana indicada en "Coordinación". El agente no fusiona sin pedido explícito del propietario.

## Skills
Consultar .agents/skills/<nombre>/SKILL.md según tarea. No sustituyen reglas anteriores. Si las herramientas no están instaladas, documentar bloqueo; no instalar globalmente o ejecutar cambios de infraestructura no autorizados.
## Entrega
Resumen, archivos, ADR aplicados, evidencia RED/GREEN, impacto en seguridad/datos, migración/rollback, riesgos y pendientes. Si no hay validación fiscal específica, no etiquetar documentos como facturas autorizadas.

