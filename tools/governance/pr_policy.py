"""Trusted-base PR metadata policy. No PR code is executed or interpolated."""
import json
import os
import re
import subprocess
from pathlib import Path

AREAS = {"backend", "frontend", "db", "openapi", "adr", "infra"}
TYPES = {"feat", "fix", "refactor", "spike", "chore"}
TYPE_MAP = {"feat": "feat", "fix": "fix", "refactor": "refactor"}
TITLE = re.compile(r"^(feat|fix|refactor|docs|chore|test|ci|build|perf|style|revert|security|spike)(?:\([a-zA-Z0-9._/-]+\))?!?: \S.+$")
REF = re.compile(r"(?im)^\s*(Part of|Closes)\s+#(\d+)\s*$")
CODE = re.compile(r"\b(?:BP|ADR)-\d{3}\b")
HEADERS = ["BP / Issue relacionado", "ADRs relacionados o ratificados", "Alcance",
           "Criterios de aceptación", "Checklist de validación", "Cómo probar", "Tamaño y excepciones"]
EXCEPTIONS = {"gobernanza", "documentación", "seguridad", "migraciones", "dependencias"}


def gh(*args):
    result = subprocess.run(["gh", *args], capture_output=True, text=True, check=True)
    return json.loads(result.stdout)


def sections(body):
    body = re.sub(r"<!--.*?-->", "", body, flags=re.S)
    pieces = re.split(r"(?m)^## +(.+?)\s*$", body)
    return dict(zip(pieces[1::2], (s.strip() for s in pieces[2::2])))


def meaningful(value):
    return bool(re.search(r"[A-Za-zÁÉÍÓÚáéíóúñÑ0-9]{3,}", value)) and value.strip() not in {"N/A", "BP-XXX"}


def productive(files):
    def excluded(path):
        return (path.endswith(("go.sum", "_test.go", ".lock", "_test.dart"))
                or "/fixtures/" in path or "/testdata/" in path or "/snapshots/" in path
                or re.search(r"(^|/)tests?/", path) or path.endswith((".md", ".rst")))
    return sum(f["additions"] + f["deletions"] for f in files if not excluded(f["filename"]))


def governance_only(files):
    def allowed(path):
        return (path.startswith((".github/", "tools/governance/", ".agents/skills/"))
                or path in {"AGENTS.md", "README.md", "docs/deployment/github-governance.md"})
    return bool(files) and all(allowed(p) for f in files
                               for p in [f["filename"], f.get("previous_filename", f["filename"])])


# A ratification may touch the decision records and the two documents that
# mirror ADR/BP status; nothing else (no code, workflows or other docs).
RATIFICATION_EXTRA = {"docs/MASTER.md", "BACKLOG.md"}


def ratification_only(files):
    def allowed(path):
        return path.startswith("docs/decisions/") or path in RATIFICATION_EXTRA
    return bool(files) and all(allowed(p) for f in files
                               for p in [f["filename"], f.get("previous_filename", f["filename"])])


def validate(pr, files, issue, registry):
    errors = []
    title = TITLE.fullmatch(pr["title"])
    if not title:
        errors.append("Título: Conventional Commits requerido.")
    body = re.sub(r"<!--.*?-->", "", pr.get("body") or "", flags=re.S)
    sec = sections(body)
    refs = REF.findall(sec.get(HEADERS[0], ""))
    if len(refs) != 1 or issue is None or int(refs[0][1]) != issue["number"]:
        errors.append("Una referencia Part of #N o Closes #N a un issue local existente es obligatoria.")
    closing = re.findall(r"(?i)\b(?:close[sd]?|fix(?:es|ed)?|resolve[sd]?)\s+(?:[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)?#(\d+)", body)
    allowed_closing = [refs[0][1]] if len(refs) == 1 and refs[0][0].lower() == "closes" else []
    if closing != allowed_closing:
        errors.append("No usar palabras de cierre fuera de la referencia declarada.")
    for h in HEADERS:
        if not meaningful(sec.get(h, "")):
            errors.append("Sección incompleta: " + h)
    scope = sec.get("Alcance", "")
    for label in ("Entra", "Queda fuera"):
        m = re.search(r"(?im)^\*\*" + label + r"[^\n]*?\*\*\s*:?\s*\n?([^\n]+)", scope)
        if not m or not meaningful(m.group(1)):
            errors.append("Declarar alcance explícito: " + label)
    acceptance = sec.get("Criterios de aceptación", "")
    if not re.search(r"(?m)^\s*[-*] (?:\[[ xX]\] )?\S.{5,}", acceptance):
        errors.append("Criterios de aceptación específicos requeridos.")
    checks = sec.get("Checklist de validación", "")
    if len(re.findall(r"(?m)^\s*- \[[xX]\]", checks)) < 5 or re.search(r"(?m)^\s*- \[ \]", checks):
        errors.append("Completar las cinco comprobaciones de la plantilla; justificar N/A en Cómo probar.")
    evidence = sec.get("Cómo probar", "")
    if not re.search(r"(?im)^\s*(?:resultado|evidencia|no aplica)\s*:\s*\S.{8,}", evidence):
        errors.append("Evidencia/resultado concreto o No aplica con justificación requerido.")
    labels = {l["name"] for l in pr.get("labels", [])}
    if not labels & AREAS or not labels & TYPES:
        errors.append("Labels de área y tipo requeridos.")
    if title and TYPE_MAP.get(title[1], "chore" if title[1] != "spike" else "spike") not in labels:
        errors.append("Label de tipo debe corresponder al título.")
    size = productive(files)
    exception = re.search(r"(?im)^Excepción:\s*(\S+)", sec.get("Tamaño y excepciones", ""))
    reason = re.search(r"(?im)^Justificación:\s*(\S.{15,})", sec.get("Tamaño y excepciones", ""))
    if size > 500 and not reason:
        errors.append("Más de 500 líneas productivas: dividir o justificar explícitamente.")
    if exception and exception[1].lower() not in EXCEPTIONS:
        errors.append("Excepción no reconocida.")
    if exception and not reason:
        errors.append("Toda excepción requiere justificación.")
    gov = bool(exception and exception[1].lower() == "gobernanza" and reason
               and governance_only(files) and {"infra", "chore"} <= labels)
    manifests = {"go.mod", "go.sum", "pubspec.yaml", "pubspec.lock", "package.json", "package-lock.json", "requirements.txt", "pyproject.toml", "Cargo.toml", "Cargo.lock"}
    if any(Path(f["filename"]).name in manifests for f in files):
        if not re.search(r"(?im)^Dependencias:\s*\S.{20,}", sec.get("Tamaño y excepciones", "")):
            errors.append("Cambios de dependencias: declarar licencia, mantenimiento, seguridad y alternativa.")
    # No exemption for changes in business source. A label or body alone cannot bypass.
    if exception and exception[1].lower() == "gobernanza" and not gov:
        errors.append("Excepción gobernanza limitada a rutas administrativas y labels infra/chore.")
    if issue:
        codes = set(CODE.findall(sec.get(HEADERS[0], "")))
        expected = set(CODE.findall(issue["title"]))
        bp = {x for x in codes if x.startswith("BP-")}
        if not expected or not codes & expected or len(bp) > 1:
            errors.append("Código BP/ADR debe corresponder al issue; una sola unidad BP.")
        if issue["state"] != "OPEN":
            errors.append("Issue de trabajo debe estar abierto.")
        if refs and refs[0][0].lower() == "closes" and issue.get("open_children"):
            errors.append("No cerrar una tarea padre con subtareas abiertas; usar Part of.")
        ilabels = set(issue["labels"])
        if ilabels & AREAS and not labels & ilabels & AREAS:
            errors.append("Área incompatible con issue.")
        pm = (pr.get("milestone") or {}).get("number")
        if issue.get("milestone") and pm != issue["milestone"]:
            errors.append("Milestone del PR debe coincidir con el issue.")
        pending = {c for c in CODE.findall(sec.get(HEADERS[1], ""))
                   if c in registry and "PENDIENTE" in registry[c]}
        blocking = set(issue["blockers"])
        if "blocked-by-adr" in ilabels and not gov:
            errors.append("Issue marcado blocked-by-adr: ratificar primero.")
        ratification = bool(expected and all(c.startswith("ADR-") for c in expected)
                            and ratification_only(files) and "adr" in labels)
        if expected and all(c.startswith("ADR-") for c in expected) and not ratification:
            errors.append("Un issue ADR solo permite documentación de ratificación, no implementación.")
        if (pending or blocking) and not ratification:
            errors.append("ADR bloqueantes pendientes: ratificar antes de implementar.")
        if any(c not in registry for c in CODE.findall(sec.get(HEADERS[1], "")) if c.startswith("ADR-")):
            errors.append("ADR citada no existe en el registro de main.")
    return errors, size


def get_issue(repo, number):
    owner, name = repo.split("/")
    query = """query($o:String!,$r:String!,$n:Int!){
      repository(owner:$o,name:$r){issue(number:$n){
        number title state labels(first:100){nodes{name}} milestone{number}
        subIssues(first:100){nodes{state}}
        blockedBy(first:100){nodes{title state}}
        parent{blockedBy(first:100){nodes{title state}}}
      }}}"""
    data = gh("api", "graphql", "-f", "query=" + query, "-f", "o=" + owner,
              "-f", "r=" + name, "-F", "n=" + str(number))
    issue = data["data"]["repository"]["issue"]
    if not issue:
        return None
    blockers = issue["blockedBy"]["nodes"] + (issue.get("parent") or {}).get("blockedBy", {}).get("nodes", [])
    return {"number": issue["number"], "title": issue["title"], "state": issue["state"],
            "labels": [l["name"] for l in issue["labels"]["nodes"]],
            "milestone": (issue["milestone"] or {}).get("number"),
            "open_children": any(c["state"] == "OPEN" for c in issue["subIssues"]["nodes"]),
            "blockers": [c["title"] for c in blockers if c["state"] == "OPEN" and c["title"].startswith("ADR-")]}


def main():
    event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text())
    repo = event["repository"]["full_name"]
    n = int(event["pull_request"]["number"])
    pr = gh("api", f"repos/{repo}/pulls/{n}")
    # gh --paginate returns multiple arrays without slurp in older CLI versions.
    files = []
    for page in range(1, 31):
        batch = gh("api", f"repos/{repo}/pulls/{n}/files?per_page=100&page={page}")
        files.extend(batch)
        if len(batch) < 100:
            break
    if len(files) != pr["changed_files"]:
        raise SystemExit("Diff incompleto; revisión manual requerida.")
    ref = REF.findall(sections(pr.get("body") or "").get(HEADERS[0], ""))
    issue = get_issue(repo, int(ref[0][1])) if len(ref) == 1 else None
    registry = {"ADR-" + x["id"]: x["status"] for x in json.loads(
        Path("docs/decisions/registry.json").read_text())}
    errors, size = validate(pr, files, issue, registry)
    print(f"Líneas productivas aproximadas: {size}; objetivo 200–500 (no mínimo obligatorio).")
    for error in errors:
        print("ERROR: " + error)
    return bool(errors)


if __name__ == "__main__":
    raise SystemExit(main())
