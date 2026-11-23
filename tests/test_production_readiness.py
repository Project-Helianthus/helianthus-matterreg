"""Mutation tests for the fail-closed endpoint readiness decision."""

from __future__ import annotations

import copy
import hashlib
import json
import unittest
from pathlib import Path

from runtime import production_readiness
from runtime.production_readiness import evaluate, validate_build_evidence, validate_composition, validate_lock


ROOT = Path(__file__).resolve().parents[1]
LOCK_PATH = ROOT / "sdk" / "matter-sdk.lock.json"
SYNTHETIC_MAPPING = {
    "repository": "https://example.invalid/synthetic-docs-semantic.git",
    "path": "synthetic/matter-1.6.1-roster.json",
    "sha256": hashlib.sha256(b"synthetic docs-semantic roster").hexdigest(),
}


def committed_lock() -> dict:
    return json.loads(LOCK_PATH.read_text(encoding="utf-8"))


def load_lock(mapping: dict | None = SYNTHETIC_MAPPING) -> dict:
    """The committed go-fabric pin with a synthetic accepted roster pin."""
    lock = committed_lock()
    lock["mapping"] = copy.deepcopy(mapping)
    return lock


def accepted_composition(lock: dict) -> dict:
    return {
        "schema_version": 1,
        "status": "accepted",
        "profile_id": "matter-1.6.1-electrical-sensor-v1",
        "module": lock["sdk"]["module"],
        "release_tag": lock["sdk"]["release_tag"],
        "checksum": lock["sdk"]["checksum"],
        "mapping": copy.deepcopy(lock["mapping"]),
        "device_type": copy.deepcopy(lock["target"]["device_type"]),
        "clusters": [
            {"id": "0x001D", "name": "Descriptor"},
            {"id": "0x009C", "name": "Power Topology"},
            {"id": "0x0090", "name": "Electrical Power Measurement"},
        ],
        "electrical_power_measurement": {
            "power_mode": {"value": "ac"},
            "number_of_measurement_types": {"value": 1},
            "accuracy": [{"measurement_type": "active_power"}],
            "active_power": {"value": None},
        },
    }


def go_fabric_build(lock: dict) -> dict:
    return {
        "schema_version": 1,
        "module": lock["sdk"]["module"],
        "release_tag": lock["sdk"]["release_tag"],
        "checksum": lock["sdk"]["checksum"],
    }


def pinned(lock: dict, composition: dict) -> dict:
    """Return the lock with the profile pin set to the digest of the composition."""
    pinned_lock = copy.deepcopy(lock)
    pinned_lock["accepted_runtime_composition_profile_sha256"] = production_readiness._composition_digest(composition)
    return pinned_lock


class ProductionReadinessTest(unittest.TestCase):
    def setUp(self) -> None:
        self.lock = load_lock()
        self.composition = accepted_composition(self.lock)
        self.build = go_fabric_build(self.lock)

    def test_matching_pins_and_build_record_permit(self) -> None:
        decision = evaluate(pinned(self.lock, self.composition), self.composition, self.build)
        self.assertEqual(decision, production_readiness.Decision(True, ()))

    def test_complete_self_asserted_evidence_cannot_enable_the_gate(self) -> None:
        decision = evaluate(self.lock, self.composition, self.build)
        self.assertFalse(decision.permitted)
        self.assertIn("no reviewed accepted runtime-composition profile is pinned", decision.reasons)

    def test_unset_mapping_never_permits_even_with_a_matching_profile_digest(self) -> None:
        lock = load_lock(mapping=None)
        composition = accepted_composition(lock)
        decision = evaluate(pinned(lock, composition), composition, go_fabric_build(lock))
        self.assertFalse(decision.permitted)
        self.assertIn("no reviewed mapping is pinned", decision.reasons)

    def test_mapping_digest_differing_from_the_profile_mapping_is_named(self) -> None:
        lock = load_lock()
        composition = accepted_composition(lock)
        lock["mapping"]["sha256"] = hashlib.sha256(b"other synthetic roster").hexdigest()
        decision = evaluate(pinned(lock, composition), composition, self.build)
        self.assertFalse(decision.permitted)
        self.assertIn("runtime-composition profile pins a different mapping revision", decision.reasons)

    def test_other_release_tag_or_checksum_is_not_the_committed_pin(self) -> None:
        for field, value in (("release_tag", "v0.0.0"), ("checksum", "h1:" + "A" * 43 + "=")):
            with self.subTest(field=field):
                lock = load_lock()
                lock["sdk"][field] = value
                self.assertTrue(validate_lock(lock))
                self.assertFalse(evaluate(pinned(lock, self.composition), self.composition, self.build).permitted)

    def test_build_record_for_another_release_gives_a_reason(self) -> None:
        build = copy.deepcopy(self.build)
        build["release_tag"] = "v0.0.0"
        self.assertTrue(validate_build_evidence(self.lock, build))
        self.assertFalse(evaluate(pinned(self.lock, self.composition), self.composition, build).permitted)

    def test_composition_for_another_go_fabric_release_gives_a_reason(self) -> None:
        composition = copy.deepcopy(self.composition)
        composition["release_tag"] = "v0.0.0"
        self.assertFalse(evaluate(pinned(self.lock, composition), composition, self.build).permitted)
        self.assertTrue(any("different go-fabric release" in reason for reason in validate_composition(self.lock, composition)))

    def test_absent_composition_cannot_enable_the_gate(self) -> None:
        self.assertFalse(evaluate(self.lock, None, self.build).permitted)
        self.assertIn("absent or malformed", validate_composition(self.lock, None)[0])

    def test_proposed_composition_cannot_enable_the_gate(self) -> None:
        composition = copy.deepcopy(self.composition)
        composition["status"] = "proposed"
        self.assertFalse(evaluate(pinned(self.lock, composition), composition, self.build).permitted)
        self.assertTrue(any("not accepted" in reason for reason in validate_composition(self.lock, composition)))

    def test_missing_power_topology_cannot_enable_the_gate(self) -> None:
        composition = copy.deepcopy(self.composition)
        composition["clusters"] = [item for item in composition["clusters"] if item["id"] != "0x009C"]
        self.assertFalse(evaluate(pinned(self.lock, composition), composition, self.build).permitted)
        self.assertIn("mandatory Power Topology composition metadata is absent", validate_composition(self.lock, composition))

    def test_empty_accuracy_cannot_enable_the_gate(self) -> None:
        composition = copy.deepcopy(self.composition)
        composition["electrical_power_measurement"]["accuracy"] = []
        self.assertFalse(evaluate(pinned(self.lock, composition), composition, self.build).permitted)
        self.assertIn("mandatory Accuracy list is empty or malformed", validate_composition(self.lock, composition))

    def test_composition_carries_no_evidence_field(self) -> None:
        self.assertNotIn("evidence", json.dumps(self.composition))
        self.assertEqual(validate_composition(self.lock, self.composition), ())

    def test_build_record_without_module_path_gives_a_reason(self) -> None:
        build = copy.deepcopy(self.build)
        del build["module"]
        self.assertTrue(validate_build_evidence(self.lock, build))

    def test_module_pin_file_cannot_enable_the_gate(self) -> None:
        disk = committed_lock()
        self.assertFalse(evaluate(disk, self.composition, self.build).permitted)

    def test_schema_provider_or_model_drift_withholds_permission(self) -> None:
        drifts = (
            ("schema_version", 2),
            ("sdk.provider", "other"),
            ("sdk.model", {"name": "Matter", "version": "1.7"}),
        )
        for key, value in drifts:
            with self.subTest(key=key):
                lock = pinned(self.lock, self.composition)
                if key.startswith("sdk."):
                    lock["sdk"][key.split(".", 1)[1]] = value
                else:
                    lock[key] = value
                self.assertTrue(validate_lock(lock))
                decision = evaluate(lock, self.composition, self.build)
                self.assertFalse(decision.permitted)
                self.assertTrue(decision.reasons)

    def test_missing_profile_or_mapping_key_is_a_denial_not_an_error(self) -> None:
        for key in ("accepted_runtime_composition_profile_sha256", "mapping"):
            with self.subTest(key=key):
                lock = pinned(self.lock, self.composition)
                del lock[key]
                self.assertTrue(validate_lock(lock))
                decision = evaluate(lock, self.composition, self.build)
                self.assertFalse(decision.permitted)
                self.assertTrue(decision.reasons)

    def test_mismatched_attribute_identifier_cannot_enable_the_gate(self) -> None:
        lock = copy.deepcopy(self.lock)
        lock["target"]["active_current_attribute"]["id"] = "0x0006"
        self.assertFalse(evaluate(pinned(lock, self.composition), self.composition, self.build).permitted)


if __name__ == "__main__":
    unittest.main()
