import unittest

import plan_milestones


def page(kind, start, count, cursor, more):
    nodes = [{"number": n} for n in range(start, start + count)]
    conn = {"pageInfo": {"hasNextPage": more, "endCursor": cursor}, "nodes": nodes}
    if kind == "repo":
        return {"repository": {"id": "R", "labels": {"nodes": []}, "milestones": {"nodes": []}, "issues": conn}}
    return {"user": {"projectV2": {"id": "P", "items": conn}}}


class FetchPaginationTest(unittest.TestCase):
    def setUp(self):
        self.original = plan_milestones.gql

    def tearDown(self):
        plan_milestones.gql = self.original

    def test_fetch_follows_cursors_for_issues_and_items(self):
        calls = []

        def fake(query, **variables):
            calls.append(variables.get("after"))
            kind = "repo" if "repository(" in query else "project"
            if variables.get("after") is None:
                return page(kind, 1, 100, "c1", True)
            return page(kind, 101, 50, "c2", False)

        plan_milestones.gql = fake
        repo, project = plan_milestones.fetch()
        self.assertEqual(150, len(repo["issues"]["nodes"]))
        self.assertEqual(150, len(project["items"]["nodes"]))
        self.assertEqual("R", repo["id"])
        self.assertEqual("P", project["id"])
        self.assertEqual([None, "c1", None, "c1"], calls)

    def test_fetch_stops_on_runaway_pagination(self):
        plan_milestones.gql = lambda query, **v: page(
            "repo" if "repository(" in query else "project", 1, 1, "same", True)
        with self.assertRaises(RuntimeError):
            plan_milestones.fetch()


if __name__ == "__main__":
    unittest.main()
