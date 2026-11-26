<!-- Título en Conventional Commits: feat(scope): …, fix(scope): …, docs(adr): …, chore(scope): …
     1 PR = 1 unidad lógica (una subtarea de un BP-XXX o solo ratificación de ADR). Ver AGENTS.md. -->

## BP / Issue relacionado
Part of # <!-- o: Closes # -->
BP-XXX <!-- subtarea: … -->

## ADRs relacionados o ratificados
<!-- ADR-XXX (aplicada / ratificada en este PR) o "N/A" -->

## Alcance
**Entra:**
-

**Queda fuera (explícito):**
-

## Checklist de validación
- [ ] PR acotado a una sola unidad lógica (sin mezclar varios `BP-*` ni ADRs con código masivo); ~200–500 líneas productivas
- [ ] Cumple la arquitectura hexagonal y las reglas de `AGENTS.md` (decimales exactos, idempotencia, sin acceso directo a la BD desde la UI, permisos en el servidor)
- [ ] Tests unitarios / de integración ejecutados y en verde; evidencia RED → GREEN incluida o enlazada
- [ ] Migraciones (con `Down`) y/o contrato OpenAPI actualizados (si aplica)
- [ ] Sin secretos, credenciales ni datos reales

## Cómo probar
```sh
cd backend
export BP_TEST_DATABASE_URL='postgres://usuario@127.0.0.1:5432/bp_test?sslmode=disable'  # BD desechable
make test
```
<!-- Agrega comandos específicos (go test -run …) y resultados reales. Riesgos, migración/rollback y pendientes, si los hay. -->
