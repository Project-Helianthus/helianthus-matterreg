"""Replay the moved Matter packages against the recorded outputs."""

from __future__ import annotations

import hashlib
import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PACKAGES = ("./projection/...", "./feedv1/...", "./libraryhost/...")
TREES = (
    ("current/gateway/matter", "matter"),
    ("current/gateway/matter-feedv1", "matter-feedv1"),
    ("current/gateway/matter-libraryhost", "matter-libraryhost"),
)
GOLDEN_PATH = "cmd/gateway/testdata/characterization/consumers/matter_feed.json"


def run_go(*args: str) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env["GOWORK"] = "off"
    env["GO111MODULE"] = "on"
    return subprocess.run(
        ["go", *args], cwd=ROOT, env=env, capture_output=True, text=True, check=False,
    )


def require_dir(path: Path) -> None:
    if not path.is_dir():
        raise AssertionError("recorded tree is absent")


def vectors_root() -> Path:
    raw = os.environ.get("HELIANTHUS_ORACLE_VECTORS", "")
    if raw == "":
        raise AssertionError("HELIANTHUS_ORACLE_VECTORS is unset")
    root = Path(raw)
    for relative, _produced in TREES:
        require_dir(root / relative)
    return root


def golden_file() -> Path:
    raw = os.environ.get("HELIANTHUS_FEED_GOLDEN", "")
    if raw == "":
        raise AssertionError("HELIANTHUS_FEED_GOLDEN is unset")
    path = Path(raw)
    if not path.is_file():
        raise AssertionError("recorded feed output is absent")
    return path


def recorded_request() -> str:
    vectors = os.environ.get("HELIANTHUS_ORACLE_VECTORS", "")
    golden = os.environ.get("HELIANTHUS_FEED_GOLDEN", "")
    if vectors == "" and golden == "":
        return "absent"
    if vectors == "" or golden == "":
        return "incomplete"
    return "present"


def output_files(root: Path) -> list[Path]:
    return sorted(path for path in root.rglob("output.json") if path.is_file())


def assert_same_bytes(case: unittest.TestCase, recorded: Path, produced: Path) -> None:
    case.assertTrue(produced.is_file(), str(produced.name))
    case.assertEqual(produced.read_bytes(), recorded.read_bytes(), recorded.name)


def assert_tree(case: unittest.TestCase, recorded: Path, produced: Path) -> None:
    files = output_files(recorded)
    case.assertTrue(files)
    for recorded_file in files:
        relative = recorded_file.relative_to(recorded)
        assert_same_bytes(case, recorded_file, produced / relative)
    case.assertEqual(output_files(produced), [produced / path.relative_to(recorded) for path in files])


def named_golden(document: object) -> dict[str, object] | None:
    if not isinstance(document, dict):
        return None
    output = document.get("output")
    if not isinstance(output, dict):
        return None
    golden = output.get("golden")
    if not isinstance(golden, dict) or golden.get("path") != GOLDEN_PATH:
        return None
    return golden


def assert_golden(case: unittest.TestCase, produced: Path, golden: Path) -> None:
    digest = hashlib.sha256(golden.read_bytes()).hexdigest()
    named = [path for path in output_files(produced) if named_golden(json.loads(path.read_text(encoding="utf-8"))) is not None]
    case.assertGreater(len(named), 0)
    for path in named:
        entry = named_golden(json.loads(path.read_text(encoding="utf-8")))
        case.assertIsNotNone(entry)
        assert entry is not None
        case.assertEqual(entry.get("sha256"), digest)


class ProjectionMoveTest(unittest.TestCase):
    def test_copied_packages(self) -> None:
        completed = run_go("test", "-count=1", "-timeout=5m", *PACKAGES)
        self.assertEqual(completed.returncode, 0, completed.stdout + completed.stderr)

    def test_recorded_outputs_match(self) -> None:
        request = recorded_request()
        if request == "absent":
            self.skipTest("recorded outputs are not supplied")
        self.assertNotEqual(request, "incomplete")
        vectors = vectors_root()
        golden = golden_file()
        with tempfile.TemporaryDirectory() as temporary:
            out = Path(temporary)
            completed = run_go("run", "projection/replay_move.go", str(vectors), str(out), str(golden))
            detail = completed.stdout + completed.stderr
            self.assertEqual(completed.returncode, 0, detail)
            for recorded_rel, produced_name in TREES:
                assert_tree(self, vectors / recorded_rel, out / produced_name)
            assert_golden(self, out / "matter-feedv1", golden)
