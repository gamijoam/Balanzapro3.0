# Configuración de GitHub (gobernanza)

Estos ajustes del repositorio no viven en archivos: se aplican con `gh` autenticado por alguien con permisos de administración sobre `gamijoam/Balanzapro3.0`. Los comandos de las secciones 1–4 son ejemplos históricos de inicialización, no todos son idempotentes. No volver a ejecutarlos indiscriminadamente; consultar la reconciliación avanzada al final.

```sh
gh auth login                     # o: gh auth refresh -s repo,project
R=gamijoam/Balanzapro3.0
```

## 1. Labels
```sh
# Área
gh label create backend  -R $R --force --color 1D76DB --description "Backend Go"
gh label create frontend -R $R --force --color 5319E7 --description "Flutter/Dart"
gh label create db       -R $R --force --color 0E8A16 --description "PostgreSQL / migraciones / SQLite"
gh label create openapi  -R $R --force --color FBCA04 --description "Contrato OpenAPI"
gh label create adr      -R $R --force --color C5DEF5 --description "Decisiones de arquitectura (ADR)"
gh label create infra    -R $R --force --color BFD4F2 --description "CI/CD, instalador, entornos"
# Tipo
gh label create feat     -R $R --force --color A2EEEF --description "Funcionalidad nueva"
gh label create fix      -R $R --force --color D73A4A --description "Corrección de defecto"
gh label create refactor -R $R --force --color D4C5F9 --description "Refactor sin cambio de comportamiento"
gh label create spike    -R $R --force --color F9D0C4 --description "Prueba de concepto / investigación"
gh label create chore    -R $R --force --color EDEDED --description "Mantenimiento / gobernanza"
# Estado
gh label create blocked-by-adr -R $R --force --color B60205 --description "Bloqueada hasta ratificar ADR"
```

## 2. Milestones (ROADMAP + BACKLOG)
- **M1 · Vertical Inventario + Identidad**, Fases 03–04. Sobre el núcleo BP-011/012 ya implementado: ratificar ADR-024 (permisos) y ADR-009 (OpenAPI), BP-008 (identidad, empresa, ámbitos, sesiones), BP-009 (módulos, configuración, permisos, auditoría), BP-010 (dispositivos, outbox, cursor), BP-013 (lotes, seriales, combos) y la primera API OpenAPI de inventario. Puerta de salida: SEC-01/SEC-02, INV-02/04/05/06 y endpoints de inventario autenticados y autorizados.
- **M2 · Monedas, costos y precios**, Fase 05. BP-014 (tasas, moneda, costos, listas), previa ratificación de ADR-044 (costeo), ADR-051 (precios versionados) y ADR-034…049, más el lado Dart/SQLite de BP-006. Puerta de salida: FX-01, COST-01/02, PRICE-01.

```sh
gh api -X POST repos/$R/milestones -f title="M1 · Vertical Inventario + Identidad" \
  -f description="Fases 03–04: ADR-024/009, BP-008, BP-009, BP-010, BP-013 y API OpenAPI de inventario. Salida: SEC-01/02, INV-02/04/05/06."
gh api -X POST repos/$R/milestones -f title="M2 · Monedas, costos y precios" \
  -f description="Fase 05: ADR-044/051/034–049, BP-014 y BP-006 (Dart/SQLite). Salida: FX-01, COST-01/02, PRICE-01."
```
(La API responde 422 si el milestone ya existe; no se duplica.)

## 3. Merge y protección de `main`
Squash como único método. Las ramas se borran al fusionar:
```sh
gh repo edit $R --enable-squash-merge --enable-merge-commit=false --enable-rebase-merge=false --delete-branch-on-merge
```
Protección de `main`: PR obligatorio con una aprobación y el check `backend` en verde y al día con `main`.
```sh
gh api -X PUT repos/$R/branches/main/protection --input - <<'JSON'
{
  "required_status_checks": { "strict": true, "contexts": ["backend"] },
  "enforce_admins": false,
  "required_pull_request_reviews": { "required_approving_review_count": 1, "dismiss_stale_reviews": true },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true
}
JSON
```
El check se llama `backend` porque así se llama el job en `.github/workflows/backend.yml`. Si renombras el job, actualiza `contexts`.
Si trabajas sin otro revisor, usa `"required_approving_review_count": 0`; el PR sigue siendo obligatorio.

## 4. Project (Kanban)
```sh
gh auth refresh -s project
gh project create --owner gamijoam --title "BalanzaPro 3.0 · Roadmap"
N=$(gh project list --owner gamijoam --format json --jq '.projects[] | select(.title=="BalanzaPro 3.0 · Roadmap") | .number')
gh project link "$N" --owner gamijoam --repo $R
gh project field-create "$N" --owner gamijoam --name "Estado" --data-type SINGLE_SELECT \
  --single-select-options "Backlog,ADR pendiente,En progreso,En revisión,Listo"
```
En la interfaz web del Project, crea una vista *Board* agrupada por **Estado**. También puedes renombrar las opciones del campo `Status` predeterminado (la CLI no las edita).


## 5. Gobernanza avanzada — estado y activación progresiva

### Auditoría inicial
Repositorio público, Project #3 público, squash exclusivo, protección de main con
backend obligatorio y actualizado, una aprobación, invalidación de revisiones,
conversaciones resueltas, sin force push ni eliminación. Excepción del propietario
activa; Actions en lectura y sin permiso de aprobar PRs. Secret scanning y push
protection nativos activos. No tocar estos controles para obtener CI verde.
Dos ramas históricas de PRs #1 y #2 fueron verificadas por SHA de head, árbol idéntico
al squash, ancestro de main y ausencia de PRs abiertos; solo entonces se retiraron.

### PRs independientes
| Control | PR | Activación |
|---|---|---|
| PR Policy | #99 | Validación de metadatos desde base confiable y preview separado |
| Secretos | #100 | Gitleaks 8.30.1, SHA256 verificado, sin exponer hallazgos |
| Backend CI | #101 | checkout 7.0.1 y setup-go 7.1.0 (Node 24), SHA fijos |
| Arquitectura | #102 | Go parser/AST, imports, ciclos y flotantes en capas críticas |
| Gobernanza | este PR | Procedimientos, roles, scripts, propuestas |

Los previews de pull_request tienen código controlado por el PR; son evidencia,
no barrera de seguridad. Los checks de pull_request_target toman el workflow y
validador de base, con permisos de lectura, sin checkout/ejecución de código
del PR en contexto privilegiado. El análisis arquitectónico lee Go como datos.
El secreto se analiza mediante git, no ejecutando archivos de PR; configuración
e ignore explícitos y confiables, logs privados, sin credenciales reales en tests.
Un scan limpio no demuestra ausencia de todas las contraseñas o datos sensibles.
Las excepciones futuras requieren regla y ruta/huella específicas, prueba negativa,
justificación y revisión de seguridad; nunca desactivar reglas enteras.

No hacer obligatorio un nombre de check por el solo hecho de aparecer verde:
comprobar evento, procedencia, SHA/commit asociado, ejecución en PR legítimo y
casos negativos. Un check de Actions con igual nombre puede ser imitado por un
workflow modificado en la rama, y un job omitido puede aparecer como éxito;
evaluar reglas de workflows requeridos/rulesets disponibles antes de activar.
No introducir credenciales de Projects ni token administrativo en Actions.
Revalidar PR cuando cambia issue/ADR/milestone de origen; la edición del PR
dispara la política, no todos los cambios externos generan un evento de PR.

### Reglas automáticas vs revisión humana
Automatizable: forma del título y referencia local, código BP/ADR, metadata,
secciones, dependencia ADR abierta registrada, tamaño aproximado, manifiestos
modificados, imports y ciclos Go, tipos/conversiones flotantes en capas críticas,
detección heurística de secretos. Go parser evita ejecutar el árbol inspeccionado;
compilador/vet existentes también detectan ciclos. Se evaluó depguard de
golangci-lint como opción cuando exista más de un módulo; no añadir dependencia
de lint todavía. Excepción de compatibilidad explícita: store mínimo
internal/organization/organization.go (no extenderlo fuera del diseño vigente).
No se demuestra por AST la semántica de dinero, autorización, SQL dinámico,
transacciones, fiscalidad ni cumplimiento completo de ADR. El auditor debe
comprobar estos aspectos, dependencias/licencias y evidencia real; futuras reglas
Dart/SQLite cuando exista frontend. No introducir una ADR mediante este PR.

### Protección de main — propuesta no aplicada
Mantener backend como único check obligatorio por ahora. Tras comprobar checks
nuevos desde base y su procedencia, proponer agregarlos sin perder backend ni
strict. Historial lineal recomendado (compatible con squash). No activar todavía
aprobación posterior al último push, CODEOWNERS requerido, ni enforce_admins:
hay un solo propietario humano y GitHub no permite autoaprobar un PR propio.
Mantener el bypass del propietario como excepción explícita y auditable, no
como vía ordinaria. Para independencia real incorporar otro revisor humano
o cuentas/agentes separados con acceso mínimo: desarrollador write a ramas,
auditor read/triage o permisos mínimos de comentario, sin administración ni
merge; propietario admin. CODEOWNERS es propuesta futura, no aprobación automática.
Ningún agente cambia protección, publica, despliega o fusiona sin autorización.

### Project #3 — una fuente canónica
Usar exclusivamente **Estado** como estado de gestión; conservar **Status** nativo
sin sincronización bidireccional ni eliminación (preservación de datos y
compatibilidad). Ocho opciones: Backlog, ADR pendiente, Listo para desarrollar,
En progreso, En revisión, Cambios solicitados, Bloqueado, Completado.
Listo para desarrollar requiere alcance aceptado y ausencia de bloqueos;
Backlog puede tener predecesores previstos; Bloqueado refleja ADR o impedimento
actual. El estado de cierre real del issue prevalece sobre etiquetas visuales.
La API permite crear Board y opciones, pero no configurar el agrupamiento:
en Project → Kanban → View options → Group by → Estado; ocultar Status.
El campo Estado se mantiene manualmente o mediante el script explícito,
sin token administrativo almacenado en CI.

Automatizaciones nativas observadas habilitadas: Auto-add sub-issues to project,
Auto-close issue, Item added to project, Item closed, Pull request linked to
issue y Pull request merged. El API público expone nombre/estado, no permite
editar de forma segura sus acciones/condiciones. **Antes de usar workflows
nativos para cierres, desactivar en la UI Auto-close issue y Pull request merged**
y revisar sus reglas. No borrar workflows nativos. No hacer que Status=Done
cierre un BP. Propuesta:
- Issue creado o subissue añadido: Backlog; ADR pendiente si aplica; no iniciar.
- PR creado/vinculado (Part of o referencia): En progreso o En revisión solo
  para el issue de alcance exacto; un draft sigue En progreso.
- Revisión solicitada: En revisión; cambios solicitados: Cambios solicitados.
- Aprobado: En revisión hasta aceptación/merge del propietario, no completado.
- PR fusionado: registrar evidencia; no cerrar el BP padre ni cambiarlo a
  Completado por ese evento.
- Issue cerrado por aceptación completa: Completado; issue reabierto:
  reevaluar dependencias, no asumir Listo.
- Bloqueo ADR: bloqueado hasta ratificación explícita y registro actualizado.
Configurar condiciones en UI si son soportadas; para eventos de revisión
complejos usar reconciliación explícita o una futura GitHub App de mínimo
permiso. El script no fusiona, no cierra issues y no sincroniza Status.

### Scripts reproducibles (desde copia autorizada)
```sh
gh auth status
python3 tools/governance/reconcile.py                    # dry-run
python3 tools/governance/reconcile.py --apply            # descripciones, estados
python3 tools/governance/reconcile.py --cleanup-merged   # audita, no borra
# Solo tras verificar PR/squash y autorización:
python3 tools/governance/reconcile.py --cleanup-merged --apply
python3 tools/governance/plan_milestones.py               # dry-run
python3 tools/governance/plan_milestones.py --apply       # subtareas/dependencias
python3 tools/governance/sync_project.py                  # dry-run
python3 tools/governance/sync_project.py --apply          # solo campo Estado
```
Los scripts están limitados a este repo/Project, no cambian visibilidad,
protecciones ni decisiones ADR; no crean duplicados por título exacto. Por
seguridad, Issues y elementos del Project se paginan por cursor (hasta 50 páginas
de 100; si no termina, se detienen). Los listados de PRs y ramas siguen limitados
a 100 y se detienen al alcanzarlos; ampliarlos antes de crecer. Verificar la
situación real y revisar el dry-run antes de aplicar. Nunca ejecutar sobre
otro repo por copiar/pegar. No almacenar tokens en archivos o comandos.

Las 28 tareas BP y las 37 ADR pendientes del registro siguen siendo las tareas
raíz. Subtareas de BP-003 para gobernanza, y unidades de M1/M2 y fundación
derivadas de BACKLOG/ROADMAP, no nuevos requisitos comerciales. Dividir fases
posteriores antes de iniciar trabajo. El árbol de issues y dependencias no
ratifica decisiones ni acredita desarrollo. Registrar pruebas RED/GREEN,
comentarios P0–P3 y aceptación humana en cada PR. BP-005 completado según
evidencia backend; BP-011/012 siguen abiertos por API y conteos INV-06.
