import copy
import unittest
from pr_policy import validate

BODY = """## BP / Issue relacionado
Part of #68
BP-003
## ADRs relacionados o ratificados
No aplica: gobernanza sin decisiones comerciales.
## Alcance
**Entra:**
- Validador de metadatos
**Queda fuera (explícito):**
- Código comercial
## Criterios de aceptación
- Rechazar referencias inválidas
## Checklist de validación
- [x] Unidad
- [x] Arquitectura
- [x] Pruebas
- [x] Migraciones N/A
- [x] Secretos
## Cómo probar
Resultado: cinco pruebas en verde.
## Tamaño y excepciones
Excepción: gobernanza
Justificación: herramientas administrativas y pruebas aisladas.
"""
ISSUE = dict(number=68, title="BP-003 · Gobernanza", state="OPEN",
             labels=["infra", "chore"], milestone=None, blockers=[], open_children=False)
PR = dict(title="chore(github): policy", body=BODY,
          labels=[{"name": "infra"}, {"name": "chore"}])
FILES = [dict(filename=".github/workflows/pr-policy.yml", additions=30, deletions=0)]


class PolicyTest(unittest.TestCase):
    def errors(self, pr=None, files=None, issue=None, registry=None):
        return validate(pr or PR, files or FILES, issue or ISSUE, registry or {})[0]

    def test_valid_governance(self):
        self.assertEqual([], self.errors())

    def test_bad_title_and_reference(self):
        pr = dict(PR, title="anything", body=BODY.replace("#68", "#999"))
        self.assertGreaterEqual(len(self.errors(pr)), 2)

    def test_governance_cannot_touch_business(self):
        self.assertTrue(self.errors(files=[dict(FILES[0], filename="backend/main.go")]))

    def test_pending_adr(self):
        issue = dict(ISSUE, blockers=["ADR-024 · Permisos"])
        self.assertTrue(self.errors(issue=issue))

    def test_milestone_mismatch(self):
        self.assertTrue(self.errors(issue=dict(ISSUE, milestone=1)))

    def test_missing_acceptance_and_evidence(self):
        pr = dict(PR, body=BODY.replace("- Rechazar referencias inválidas", "-").replace("Resultado:", ""))
        self.assertGreaterEqual(len(self.errors(pr)), 2)

    def test_large_with_justification(self):
        files = [dict(FILES[0], additions=700)]
        self.assertEqual([], self.errors(files=files))
        pr = dict(PR, body=BODY.replace("Justificación:", "Motivo:"))
        self.assertTrue(self.errors(pr=pr, files=files))

    def test_cannot_close_parent(self):
        self.assertTrue(self.errors(pr=dict(PR, body=BODY.replace("Part of", "Closes")),
                                    issue=dict(ISSUE, open_children=True)))

    def test_renamed_business_path(self):
        self.assertTrue(self.errors(files=[dict(FILES[0], previous_filename="backend/main.go")]))

    def test_closing_elsewhere(self):
        self.assertTrue(self.errors(pr=dict(PR, body=BODY+"\nFixes #42")))

    def test_dependency_requires_justification(self):
        self.assertTrue(self.errors(files=[dict(FILES[0], filename="backend/go.mod")]))

    def test_adr_issue_cannot_implement_code(self):
        issue = dict(ISSUE, title="ADR-024 · Permisos", labels=["adr"])
        pr = dict(PR, body=BODY.replace("BP-003", "ADR-024"))
        self.assertTrue(self.errors(pr=pr, files=[dict(FILES[0], filename="backend/main.go")], issue=issue))
