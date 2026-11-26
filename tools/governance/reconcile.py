"""Administrative reconciliation; dry-run by default, never merges or changes visibility."""
import argparse
import json
import subprocess

R = "gamijoam/Balanzapro3.0"
OWNER = "gamijoam"
PROJECT = 3
DESCRIPTIONS = {
    "backend": "Backend Go", "frontend": "Flutter/Dart",
    "db": "PostgreSQL, migraciones y SQLite", "openapi": "Contrato OpenAPI",
    "adr": "Decisiones de arquitectura (ADR)", "infra": "CI/CD, instalador y entornos",
    "feat": "Funcionalidad nueva", "fix": "Corrección de defecto",
    "refactor": "Refactor sin cambio de comportamiento", "spike": "Investigación o prueba de concepto",
    "chore": "Mantenimiento y gobernanza", "blocked-by-adr": "Bloqueada hasta ratificar ADR",
}
STATES = [("Backlog", "GRAY", "Planificada; revisar dependencias"),
          ("ADR pendiente", "YELLOW", "Ratificación humana necesaria"),
          ("Listo para desarrollar", "GREEN", "Sin bloqueos y alcance aceptado"),
          ("En progreso", "BLUE", "Implementación o avance comprobado"),
          ("En revisión", "PURPLE", "PR abierto pendiente de revisión"),
          ("Cambios solicitados", "ORANGE", "Revisión con cambios bloqueantes"),
          ("Bloqueado", "RED", "Dependencia pendiente o impedimento explícito"),
          ("Completado", "GREEN", "Aceptación total demostrada; issue cerrado")]


def gh(*args):
    return subprocess.check_output(["gh", *args], text=True)


def gql(query, **variables):
    p = subprocess.run(["gh", "api", "graphql", "--input", "-"],
                       input=json.dumps({"query": query, "variables": variables}),
                       text=True, capture_output=True, check=True)
    response = json.loads(p.stdout)
    if response.get("errors"):
        raise RuntimeError("GraphQL rejected request; inspect privately")
    return response["data"]


def field():
    data = json.loads(gh("project", "field-list", str(PROJECT), "--owner", OWNER,
                         "--limit", "100", "--format", "json"))
    fields = [f for f in data["fields"] if f["name"] == "Estado"]
    if len(fields) != 1:
        raise RuntimeError("Expected exactly one canonical Estado field")
    return fields[0]


def reconcile(apply):
    labels = json.loads(gh("label", "list", "-R", R, "--limit", "100",
                          "--json", "name,description,color"))
    byname = {l["name"]: l for l in labels}
    for name, desc in DESCRIPTIONS.items():
        if name not in byname:
            raise RuntimeError("Expected custom label missing: " + name)
        if byname[name]["description"] != desc:
            print("Update label:", name)
            if apply:
                gh("label", "edit", name, "-R", R, "--description", desc)
    f = field()
    desired = {n for n, _, _ in STATES}
    if {x["name"] for x in f["options"]} != desired:
        previous = {x["name"]: x["id"] for x in f["options"]}
        options = []
        for name, color, desc in STATES:
            opt = dict(name=name, color=color, description=desc)
            existing = previous.get(name) or (previous.get("Listo") if name == "Completado" else None)
            if existing:
                opt["id"] = existing
            options.append(opt)
        print("Reconcile canonical Estado options (preserve existing IDs)")
        if apply:
            gql("""mutation($id:ID!,$opts:[ProjectV2SingleSelectFieldOptionInput!]!){
              updateProjectV2Field(input:{fieldId:$id,singleSelectOptions:$opts}){
                projectV2Field{... on ProjectV2SingleSelectField{id options{id name}}}}}""",
                id=f["id"], opts=options)
    print("No modifications to built-in Status or visibility.")


def cleanup(apply):
    # Require merged PR, exact head, matching squash tree, merge ancestor, and no open PR using the branch.
    prs = json.loads(gh("pr", "list", "-R", R, "--state", "merged", "--limit", "100",
                        "--json", "number,headRefName,headRefOid,mergeCommit,isCrossRepository"))
    opened = json.loads(gh("pr", "list", "-R", R, "--state", "open", "--limit", "100",
                           "--json", "headRefName,baseRefName"))
    if len(prs) >= 100 or len(opened) >= 100:
        raise RuntimeError("PR pagination required before cleanup")
    protected = {"main"} | {p[k] for p in opened for k in ("headRefName", "baseRefName")}
    branches = json.loads(gh("api", f"repos/{R}/branches?per_page=100"))
    if len(branches) >= 100:
        raise RuntimeError("Branch pagination required before cleanup")
    for b in branches:
        name, head = b["name"], b["commit"]["sha"]
        if name in protected or b["protected"]:
            continue
        candidates = [p for p in prs if p["headRefName"] == name and p["headRefOid"] == head
                      and p["mergeCommit"] and not p["isCrossRepository"]]
        if not candidates:
            continue
        merged = candidates[0]["mergeCommit"]["oid"]
        tree = lambda sha: json.loads(gh("api", f"repos/{R}/git/commits/{sha}"))["tree"]["sha"]
        compare = json.loads(gh("api", f"repos/{R}/compare/{merged}...main"))
        if tree(head) != tree(merged) or compare["status"] not in ("ahead", "identical"):
            continue
        print("Verified merged branch:", name, "PR", candidates[0]["number"])
        if apply:
            # Recheck just before removal; do not delete if the ref has moved.
            current = json.loads(gh("api", f"repos/{R}/git/ref/heads/{name}"))["object"]["sha"]
            if current != head:
                raise RuntimeError("Branch changed during audit; stopped")
            gh("api", "-X", "DELETE", f"repos/{R}/git/refs/heads/{name}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--cleanup-merged", action="store_true")
    args = parser.parse_args()
    reconcile(args.apply)
    if args.cleanup_merged:
        cleanup(args.apply)
