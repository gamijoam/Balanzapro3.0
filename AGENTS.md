# BalanzaPro3.0 — instrucciones obligatorias para agentes
## Antes de cambiar
Leer README, ROADMAP, BACKLOG, TECH_STACK, docs/decisions/README y ADR del alcance, reglas de negocio y seguridad. Confirmar criterio de aceptación y decisiones pendientes; pedir ratificación si afectan comportamiento. No inventar requisitos ni promover propuestas a aprobadas. Nunca tocar otras instalaciones en /opt, producción, bases reales, secretos ni desplegar sin autorización.
## Arquitectura
Go modular hexagonal; Flutter/Dart con design system Shadcn encapsulado; PostgreSQL autoridad única por ámbito; SQLite/Drift caché y borradores; OpenAPI como contrato. Dependencias entre módulos por puertos; no importaciones cíclicas, acceso directo a BD desde UI, lógica comercial duplicada, ni microservicios por defecto. Dinero y cantidades exactas; strings decimales en API. Operaciones confirmadas inmutables, compensaciones y auditoría. Stock, kardex, costos y obligaciones compatibles se registran en una transacción. Idempotencia durable con hash del payload; no repetir pagos/fiscal de resultado desconocido. Permisos servidor y ámbito, denegar por defecto.
## Diseño y UX multiplataforma — obligatorio (ADR-011)
- Antes de crear o editar una pantalla, leer docs/design-system/UX.md y usar la skill .agents/skills/flutter-shadcn-ui/SKILL.md. Revisar los componentes y tokens existentes.
- Usar únicamente componentes BP y tokens semánticos compartidos cuando exista equivalente. No declarar colores, tamaños tipográficos, radios, espaciados o breakpoints arbitrarios en features. No importar Shadcn directamente fuera del design system (excepto spikes aislados explícitamente autorizados).
- Si falta un componente, extender la biblioteca central con tests y catálogo antes de duplicarlo. Mantener variantes semánticas (primario, secundario, destructivo, etc.); los cambios globales se hacen en los tokens o componentes centrales, no pantalla por pantalla.
- Diseñar por ancho y modo de interacción: escritorio con teclado/ratón, tablet y móvil táctil. Controles primarios táctiles >=48 px lógicos, foco visible, orden de tabulación, etiquetas accesibles, contraste AA como objetivo y escalado de texto. No depender solo de color, hover o gestos.
- Implementar estados de carga con BPSkeleton cuando corresponda, vacío, error recuperable, desconexión y datos obsoletos. Skeletons mantienen geometría; respetar movimiento reducido y nunca simular datos ni ocultar fallos.
- Probar variantes claro/oscuro, 360/800/1440 px lógicos como casos base, texto ampliado, navegación/teclado y táctil. Golden tests para componentes estables, widget tests para comportamiento, pruebas nativas antes de release. Documentar resultados y limitaciones reales; no afirmar certificación por simulación.
- Actualizar catálogo de componentes y ejemplos al cambiar la API visual. Widgetbook candidato sujeto a auditoría de dependencias; catálogo interno provisional permitido.
- Priorizar tareas frecuentes y simplicidad; ocultar complejidad opcional, no controles de seguridad o requisitos legales. Nunca duplicar reglas comerciales ni acceder directamente a BD desde UI. Mantener ADR-169.
- En PRs de UI incluir evidencia visual antes/después cuando sea posible, tamaños probados, estados y accesibilidad, regresiones de tokens/componentes, y pendientes. Excepciones deben justificarse y revisarse; no relajar reglas de seguridad.

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


## Gobernanza avanzada — propuesta en PRs separados
- **Desarrollador:** una subtarea y rama/worktree; TDD, pruebas reales, PR y evidencia. No aprueba su propio trabajo.
- **Auditor independiente:** revisar diff, ADR, arquitectura, seguridad, invariantes comerciales y cobertura; registrar hallazgos en el PR con archivo/línea y prueba, sin fingir una aprobación humana. P0 crítico, P1 importante bloqueante, P2 mejora necesaria, P3 sugerencia no bloqueante. P0/P1 impiden recomendar merge; P2 se corrige o queda aceptado explícitamente con issue.
- **Propietario humano:** ratifica ADR, decide excepciones y aceptación, autoriza explícitamente cada merge. Una IA no fusiona automáticamente ni suplanta al revisor. Dos agentes autenticados como el mismo usuario no son dos aprobaciones GitHub independientes.
- Usar `Part of #N` para trabajo parcial (incluidas subtareas de una tarea padre); `Closes #N` solamente para una unidad totalmente aceptada, sin hijos pendientes, y con autorización humana. No cerrar el BP padre por fusionar un PR. Un issue ADR solo permite su ratificación documentada.
- 200–500 líneas productivas es objetivo, no mínimo ni límite inflexible. Si superar 500 está justificado, declarar razón y riesgo; dividir cuando sea posible. Gobernanza, documentación, seguridad, migraciones y dependencias admiten excepciones explícitas, nunca elusión de ADR, seguridad o revisión.
- **Cierre de tareas (obligatorio):** después de cada merge, revisar los issues afectados. Si el PR completó todos los criterios de aceptación de un issue, marcar sus casillas, comentar los PR y la evidencia, cerrarlo como completado y poner **Estado** = Completado en el Project. Si el trabajo fue parcial, comentar lo hecho y lo pendiente y dejarlo abierto. Cerrar la tarea padre cuando todos sus hijos estén cerrados y su aceptación se cumpla. Al terminar cada tarea, listar al propietario qué issues se cerraron y cuáles siguen abiertos. Autorización del propietario (2026-11-26): un merge que él autoriza y que completa la aceptación permite cerrar el issue sin otra confirmación; ante una duda de alcance, preguntar.
- Campo canónico del Project: **Estado**. `Status` es nativo y no es una segunda fuente de verdad. No actualizar ambos como si fueran equivalentes. Estado Completado exige aceptación de todo el issue, no merge parcial.
- Checks nuevos se prueban en modo no obligatorio; distinguir previews controlados por el PR de validación desde base confiable. No modificar protecciones ni aprobar excepciones administrativas sin autorización humana.
