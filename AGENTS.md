# BalanzaPro3.0 — instrucciones obligatorias para agentes
## Antes de cambiar
Leer README, ROADMAP, BACKLOG, TECH_STACK, docs/decisions/README y ADR del alcance, reglas de negocio y seguridad. Confirmar criterio de aceptación y decisiones pendientes; pedir ratificación si afectan comportamiento. No inventar requisitos ni promover propuestas a aprobadas. Nunca tocar otras instalaciones en /opt, producción, bases reales, secretos ni desplegar sin autorización.
## Arquitectura
Go modular hexagonal; Flutter/Dart con design system Shadcn encapsulado; PostgreSQL autoridad única por ámbito; SQLite/Drift caché y borradores; OpenAPI como contrato. Dependencias entre módulos por puertos; no importaciones cíclicas, acceso directo a BD desde UI, lógica comercial duplicada, ni microservicios por defecto. Dinero y cantidades exactas; strings decimales en API. Operaciones confirmadas inmutables, compensaciones y auditoría. Stock, kardex, costos y obligaciones compatibles se registran en una transacción. Idempotencia durable con hash del payload; no repetir pagos/fiscal de resultado desconocido. Permisos servidor y ámbito, denegar por defecto.
## Flujo obligatorio
Plan pequeño → casos de aceptación → prueba RED reproducible → código mínimo GREEN → refactor → pruebas integración/concurrencia → revisión independiente → PR. No adaptar tests para ocultar fallos. Incluir evidencia de comandos, resultados y fallos; no afirmar ejecución no realizada. Migraciones compatibles y recuperación documentada. Nunca introducir librería sin justificar licencia, mantenimiento, seguridad, soporte de plataformas, benchmark cuando proceda y alternativa. Guardar decisión.
## Coordinación
Un agente por rama/worktree y tarea; cambios de contratos coordinados. No fusionar ni publicar automáticamente. No usar datos reales. Validar lint, tests, vulnerabilidades y secretos en CI cuando exista. No comprometer tokens, credenciales, llaves, dumps ni datos sensibles.
## Skills
Consultar .agents/skills/<nombre>/SKILL.md según tarea. No sustituyen reglas anteriores. Si las herramientas no están instaladas, documentar bloqueo; no instalar globalmente o ejecutar cambios de infraestructura no autorizados.
## Entrega
Resumen, archivos, ADR aplicados, evidencia RED/GREEN, impacto en seguridad/datos, migración/rollback, riesgos y pendientes. Si no hay validación fiscal específica, no etiquetar documentos como facturas autorizadas.

