"""Idempotent planning of the next units, no changes to ADR status or business code."""
import argparse
import json
import time
from reconcile import R, OWNER, PROJECT, gh, gql, field

# Proposed work units derived from BACKLOG.md and the existing milestone gates.
# Later phases must be similarly split before work begins, not speculatively designed here.
PLAN = {
    2: [("Flutter/Dart: versiones y licencias", "Matriz reproducible de herramientas Flutter/Dart", "frontend"),
        ("Distribución: versiones y licencias", "Matriz de instalación, actualización y alternativas", "infra")],
    3: [("Scaffold Flutter/Dart pendiente", "Build, lint y tests Flutter; integración CI sin sustituir backend", "frontend,infra")],
    6: [("Exactitud Dart/SQLite", "Roundtrip decimal exacto Dart/SQLite; evidencia de BP-006", "frontend,db"),
        ("Contrato decimal OpenAPI", "Contrato de importes exactos y roundtrip Go/Dart; ADR-009 ratificada", "openapi,backend")],
    8: [("Esquema de identidad y ámbitos", "Migración reversible, aislamiento por empresa y pruebas SEC-01", "backend,db"),
        ("Dominio y puertos de identidad", "Pruebas unitarias SEC-01 de ámbito y sesión", "backend"),
        ("Adaptador de identidad PostgreSQL", "Integración real y aislamiento concurrente", "backend,db"),
        ("API de identidad y sesiones", "OpenAPI y handlers con autorización; SEC-01", "backend,openapi")],
    9: [("Esquema de módulos, configuración y auditoría", "Migración reversible; no perder historia al desactivar", "backend,db"),
        ("Dominio y puertos de permisos", "SEC-02 y ADR-024 ratificada; TDD", "backend"),
        ("Adaptador de permisos y auditoría", "Integración PostgreSQL; denegación y auditoría", "backend,db"),
        ("Contrato y API de administración", "SEC-02 y contrato autenticado", "backend,openapi")],
    10: [("Esquema e identidad de dispositivos", "Migración, identidad y ciclo de revocación según ADR vigentes", "backend,db"),
         ("Outbox y cursor local", "Replay, idempotencia, lag y reconexión", "backend,db"),
         ("Adaptador de sincronización de terminal", "Contrato y pruebas de reconexión, sin duplicar autoridad", "backend,frontend,openapi")],
    11: [("API de catálogo", "INV-03 vía OpenAPI, autorización y validación; no rehacer núcleo backend", "backend,openapi")],
    12: [("Conteos y ajustes INV-06", "TDD e integración de conteos según reglas de inventario", "backend,db"),
         ("API de kardex, reservas y transferencias", "INV-01/06/07 vía OpenAPI, autorización y ámbito", "backend,openapi")],
    13: [("Lotes y FEFO", "INV-02; dividir migración/dominio/adaptador si supera el tamaño", "backend,db"),
         ("Seriales e IMEI", "INV-04; dividir migración/dominio/adaptador si supera el tamaño", "backend,db"),
         ("Combos", "INV-05; dividir migración/dominio/adaptador si supera el tamaño", "backend,db")],
    14: [("Tasas y moneda", "FX-01 con reglas ratificadas y exactitud decimal", "backend,db"),
         ("Valoración y costos", "COST-01/02 tras ratificación de ADR-044", "backend,db"),
         ("Listas y versionado de precios", "PRICE-01 tras ADR-051; no recalcular historia", "backend,db"),
         ("Contrato de monedas y precios", "OpenAPI y pruebas de aceptación del motor", "backend,openapi")],
}


MAX_PAGES = 50  # 5000 issues/items; stop rather than loop forever on a bad cursor

REPO_QUERY = """query($after:String){
  repository(owner:"gamijoam",name:"Balanzapro3.0"){
    id labels(first:100){nodes{id name}} milestones(first:20){nodes{id number}}
    issues(first:100, after:$after){pageInfo{hasNextPage endCursor} nodes{id number title state milestone{id number}
      labels(first:30){nodes{name}} blockedBy(first:100){nodes{id number title state}}
      parent{number} }}
  }
}"""

PROJECT_QUERY = """query($after:String){
  user(login:"gamijoam"){projectV2(number:3){id items(first:100, after:$after){pageInfo{hasNextPage endCursor} nodes{
    id content{... on Issue{id number title state}}
    fieldValueByName(name:"Estado"){... on ProjectV2ItemFieldSingleSelectValue{name}}
  }}}}
}"""


def _paginate(query, owner_of, connection):
    """Follow endCursor until the connection is exhausted; returns the owner
    object of the first page with the connection's nodes concatenated."""
    after, owner, nodes = None, None, []
    for _ in range(MAX_PAGES):
        data = owner_of(gql(query, after=after))
        conn = data[connection]
        owner = owner or data
        nodes.extend(conn["nodes"])
        if not conn["pageInfo"]["hasNextPage"]:
            owner[connection] = {"pageInfo": {"hasNextPage": False}, "nodes": nodes}
            return owner
        after = conn["pageInfo"]["endCursor"]
    raise RuntimeError("Pagination did not finish; stop rather than reconcile an incomplete dataset")


def fetch():
    repo = _paginate(REPO_QUERY, lambda d: d["repository"], "issues")
    project = _paginate(PROJECT_QUERY, lambda d: d["user"]["projectV2"], "items")
    return repo, project


def create(apply):
    repo, project = fetch()
    issues = {i["title"]: i for i in repo["issues"]["nodes"]}
    parents = {int(i["title"][3:6]): i for i in issues.values()
               if i["title"].startswith("BP-") and not i["parent"]}
    labels = {l["name"]: l["id"] for l in repo["labels"]["nodes"]}
    pending = {int(i["title"][4:7]): i for i in issues.values()
               if i["title"].startswith("ADR-") and i["state"] == "OPEN"}
    for n, units in PLAN.items():
        parent = parents[n]
        for name, acceptance, areas in units:
            title = f"BP-{n:03d} · {name}"
            if title in issues:
                continue
            print("Create subissue:", title)
            if not apply:
                continue
            typ = "chore" if n in (2, 3) else "spike" if n == 6 else "feat"
            ls = areas.split(",") + [typ]
            body = (f"## Alcance\nUnidad planificada de #{parent['number']}; confirmar tamaño y alcance antes de implementar.\n\n"
                    f"## Aceptación\n- [ ] {acceptance}.\n- [ ] TDD y revisión independiente; evidencia real.\n\n"
                    "## Fuera de alcance\nOtras tareas BP, decisiones ADR no ratificadas y despliegue.\n\n"
                    "Dividir nuevamente si supera 500 líneas productivas. El merge parcial no cierra la tarea padre.")
            values = dict(repositoryId=repo["id"], title=title, body=body,
                          parentIssueId=parent["id"], projectV2Ids=[project["id"]],
                          labelIds=[labels[l] for l in ls])
            if parent["milestone"]:
                values["milestoneId"] = parent["milestone"]["id"]
            gql("mutation($v:CreateIssueInput!){createIssue(input:$v){issue{number}}}", v=values)
            time.sleep(.3)
    # Explicit contract and pricing gates from docs/deployment/github-governance.md.
    if not apply:
        print("Reconcile ADR gates for M1/M2 and subissue order")
        return
    repo, project = fetch()
    issues = {i["title"]: i for i in repo["issues"]["nodes"]}
    parents = {int(i["title"][3:6]): i for i in issues.values() if i["title"].startswith("BP-") and not i["parent"]}
    def block(issue, dependency):
        if issue["id"] == dependency["id"] or dependency["id"] in {d["id"] for d in issue["blockedBy"]["nodes"]}:
            return
        gql("mutation($v:AddBlockedByInput!){addBlockedBy(input:$v){issue{number}}}",
            v=dict(issueId=issue["id"], blockingIssueId=dependency["id"]))
        issue["blockedBy"]["nodes"].append(dependency)
        print("Dependency:", issue["number"], "<-", dependency["number"])
    for a in [9]:
        if a in pending:
            block(parents[6], pending[a])
    for a in list(range(34, 50)) + [51]:
        if a in pending:
            block(parents[14], pending[a])
    for n, units in PLAN.items():
        parent = parents[n]
        previous = None
        for name, acceptance, areas in units:
            issue = issues[f"BP-{n:03d} · {name}"]
            if previous:
                block(issue, previous)
            else:
                for dep in parent["blockedBy"]["nodes"]:
                    if dep["state"] == "OPEN":
                        block(issue, dep)
            if "openapi" in areas and 9 in pending:
                block(issue, pending[9])
            if n == 9 and 24 in pending:
                block(issue, pending[24])
            previous = issue
    # Labels reflect direct open ADR dependencies, not inferred architectural approval.
    for issue in issues.values():
        if issue["state"] != "OPEN" or not issue["title"].startswith("BP-"):
            continue
        blocked = any(d["state"] == "OPEN" and d["title"].startswith("ADR-")
                      for d in issue["blockedBy"]["nodes"])
        if blocked and "blocked-by-adr" not in {l["name"] for l in issue["labels"]["nodes"]}:
            gh("issue", "edit", str(issue["number"]), "-R", R, "--add-label", "blocked-by-adr")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true")
    create(parser.parse_args().apply)
