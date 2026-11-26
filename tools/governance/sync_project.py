"""Explicit, idempotent reconciliation of Estado from evidence; never closes issues."""
import argparse
import json
import re
from reconcile import R, OWNER, PROJECT, gh, gql, field
from plan_milestones import fetch

def sync(apply):
    repo, project = fetch()
    options = {o["name"]: o["id"] for o in field()["options"]}
    issues = {i["number"]: i for i in repo["issues"]["nodes"]}
    prs = json.loads(gh("pr", "list", "-R", R, "--state", "open", "--limit", "100",
                        "--json", "number,body,isDraft,reviewDecision"))
    if len(prs) >= 100:
        raise RuntimeError("PR pagination required")
    links = {}
    for p in prs:
        for n in re.findall(r"(?im)^\s*(?:Part of|Closes)\s+#(\d+)\s*$", p["body"] or ""):
            links.setdefault(int(n), []).append(p)
    changes = []
    for item in project["items"]["nodes"]:
        c = item.get("content") or {}
        if c.get("number") not in issues:
            continue
        issue = issues[c["number"]]
        old = (item["fieldValueByName"] or {}).get("name")
        blockers = [b for b in issue["blockedBy"]["nodes"] if b["state"] == "OPEN"]
        adr = any(b["title"].startswith("ADR-") for b in blockers)
        linked = links.get(issue["number"], [])
        if issue["state"] == "CLOSED":
            # Only a closed issue with completed reason may be considered done.
            reason = json.loads(gh("api", f"repos/{R}/issues/{issue['number']}"))["state_reason"]
            if reason != "completed":
                continue
            new = "Completado"
        elif issue["title"].startswith("ADR-"):
            new = "ADR pendiente"
        elif adr or (blockers and old in {"En progreso", "Listo para desarrollar", "Bloqueado"}):
            new = "Bloqueado"
        elif linked:
            if any(p["reviewDecision"] == "CHANGES_REQUESTED" for p in linked):
                new = "Cambios solicitados"
            elif all(p["isDraft"] for p in linked):
                new = "En progreso"
            else:
                new = "En revisión"
        elif old in {"Cambios solicitados", "En revisión", "En progreso"}:
            # Do not infer completion or change evidence-based states from absence of an open PR.
            continue
        elif blockers:
            new = "Backlog"
        elif old in (None, "Backlog", "Bloqueado", "Completado"):
            # A lack of graph blockers is not proof of approval of scope; leave planned.
            new = "Backlog"
        else:
            continue
        if old != new:
            changes.append((item["id"], issue["number"], old, new))
    print("Planned status updates:", len(changes))
    for _, number, old, new in changes:
        print(number, old, "->", new)
    if apply:
        f = field()
        for start in range(0, len(changes), 10):
            part = changes[start:start+10]
            vars = {f"v{i}": dict(projectId=project["id"], itemId=item,
                                   fieldId=f["id"], value=dict(singleSelectOptionId=options[new]))
                    for i, (item, _, _, new) in enumerate(part)}
            decl = ",".join(f"$v{i}:UpdateProjectV2ItemFieldValueInput!" for i in range(len(part)))
            calls = " ".join(f"a{i}:updateProjectV2ItemFieldValue(input:$v{i}){{projectV2Item{{id}}}}"
                             for i in range(len(part)))
            gql("mutation("+decl+"){"+calls+"}", **vars)

if __name__ == "__main__":
    p=argparse.ArgumentParser()
    p.add_argument("--apply", action="store_true")
    sync(p.parse_args().apply)
