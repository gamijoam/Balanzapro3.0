# BalanzaPro3.0
Blueprint de ERP comercial modular, local-first respecto a Internet (no escrituras autónomas sin autoridad en V1), para Venezuela y expansión internacional. Estado: planificación aprobada parcialmente; núcleo de inventario backend (Go + PostgreSQL) implementado y probado, sin API de negocio, sin frontend ni certificación fiscal (ver BACKLOG.md y ADR-169).
## Comenzar
1. Leer AGENTS.md, docs/MASTER.md, docs/decisions/README.md y TECH_STACK.md.
2. Leer ROADMAP.md y BACKLOG.md; ejecutar únicamente una tarea autorizada.
3. Ratificar ADR pendientes que afecten esa tarea y registrar alternativas.
4. Utilizar PROMPT_INICIAL.md en Codex; validar con TDD y revisión.
Salvo `backend/`, las carpetas de código son marcadores. No es una aplicación operativa: no desplegar en producción. Remoto: GitHub gamijoam/Balanzapro3.0. El PDF maestro es una vista de la documentación; los Markdown y ADR versionados prevalecen.

## Gestión del desarrollo
- [Roadmap en GitHub Projects](https://github.com/users/gamijoam/projects/3) y [issues](https://github.com/gamijoam/Balanzapro3.0/issues).
- [Gobernanza, revisión independiente y controles](docs/deployment/github-governance.md).
- El propietario autoriza los merges; un PR no completa automáticamente un BP padre.
