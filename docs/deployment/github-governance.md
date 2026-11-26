# Configuración de GitHub (gobernanza)

Estos ajustes del repositorio no viven en archivos: se aplican con `gh` autenticado por alguien con permisos de administración sobre `gamijoam/Balanzapro3.0`. Los comandos son idempotentes: se pueden volver a ejecutar sin duplicar nada.

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
