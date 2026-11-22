"""Check the documented upstream engagement boundary and its reference."""

import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]


class UpstreamEngagementTest(unittest.TestCase):
    def test_board_decision_and_workflow_link(self):
        readme = (ROOT / "README.md").read_text(encoding="utf-8")
        decision = (
            "The current go-fabric dependency is the upstream v0.25.0 release. "
            "Helianthus has no go-fabric fork. Upstream contributions remain subject "
            "to the contributor agreement and an explicit Board decision."
        )
        self.assertIn(decision, readme)
        section = readme.split("## Upstream engagement\n", 1)[1].split("\n## ", 1)[0]
        self.assertIn(
            "Evidence class: **Unknown**. Status: **candidate**. "
            "Provenance is not published.",
            section,
        )
        self.assertNotIn("audit report", section)
        self.assertNotIn("Board decision,", section)
        target = "docs/adoption/upstream-contribution-workflow.md"
        self.assertIn(f"]({target})", readme)
        workflow = (ROOT / target).read_text(encoding="utf-8")
        self.assertIn("Consuming the Apache-2.0 library and granting rights", workflow)
        self.assertIn("owner CLA review pending", workflow)
        self.assertIn("per-contribution authorization must precede it", workflow)
        self.assertIn("a local fix", workflow)
        self.assertIn("contacting the maintainer", workflow)


if __name__ == "__main__":
    unittest.main()
