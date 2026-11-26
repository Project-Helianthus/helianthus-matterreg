"""Production endpoint admission is deliberately fail-closed.

This module does not create a Matter node, open a fabric, or transform a
Gateway value. It only evaluates evidence a future launcher must supply before
it is allowed to create a Helianthus Electrical Sensor endpoint. Admission is
declared from the specification mapping the lock pins and the accepted
composition profile it pins; a bench, hosted or fixture run is a recorded level
and is not an input to this decision.
"""

from __future__ import annotations

import hashlib
import json
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any


COMMITTED_LOCK_PATH = Path(__file__).resolve().parents[1] / "sdk" / "matter-sdk.lock.json"
SDK_PIN_FIELDS = ("module", "release_tag", "checksum")
MAPPING_FIELDS = frozenset({"repository", "path", "sha256"})
COMPOSITION_FILE_FIELDS = frozenset({"schema_version", "profiles"})
PROFILE_FIELDS = frozenset({
    "schema_version",
    "status",
    "profile_id",
    "module",
    "release_tag",
    "checksum",
    "mapping",
    "device_type",
    "clusters",
    "electrical_power_measurement",
})
REQUIRED_CLUSTERS = {"0x001D": "Descriptor", "0x009C": "Power Topology", "0x0090": "Electrical Power Measurement"}
_DIGEST = re.compile(r"[0-9a-f]{64}")


@dataclass(frozen=True)
class Decision:
    """A decision suitable for a caller to log without exposing credentials."""

    permitted: bool
    reasons: tuple[str, ...]


def _is_nonempty_text(value: Any) -> bool:
    return isinstance(value, str) and bool(value.strip())


def _is_digest(value: Any) -> bool:
    return isinstance(value, str) and _DIGEST.fullmatch(value) is not None


def committed_lock() -> dict[str, Any]:
    """The committed go-fabric pin, the reference for the module, release tag and checksum."""
    lock = json.loads(COMMITTED_LOCK_PATH.read_text(encoding="utf-8"))
    if not isinstance(lock, dict):
        raise ValueError("committed lock is not an object")
    return lock


def validate_lock(lock: Any) -> tuple[str, ...]:
    """Return drift reasons against the committed go-fabric pin and the optional roster and profile pins."""
    if not isinstance(lock, dict):
        return ("lock is not an object",)
    reasons: list[str] = []
    committed = committed_lock()
    if type(lock.get("schema_version")) is not int or lock["schema_version"] != 1:
        reasons.append("lock schema is unsupported")
    sdk = lock.get("sdk")
    if not isinstance(sdk, dict) or any(sdk.get(field) != committed["sdk"][field] for field in SDK_PIN_FIELDS):
        reasons.append("lock differs from the committed go-fabric pin")
    elif sdk.get("provider") != "go-fabric" or sdk.get("model") != {"name": "Matter", "version": "1.6.1"}:
        reasons.append("lock provider and data model must be go-fabric Matter 1.6.1")
    if lock.get("target") != committed["target"]:
        reasons.append("lock target differs from the committed target")
    if "mapping" not in lock or "accepted_runtime_composition_profile_sha256" not in lock:
        reasons.append("lock must carry the mapping and accepted runtime-composition profile pins")
    mapping = lock.get("mapping")
    if mapping is not None and (not isinstance(mapping, dict) or set(mapping) != MAPPING_FIELDS):
        reasons.append("mapping must be unset or carry exactly repository, path and sha256")
    elif isinstance(mapping, dict) and not _is_digest(mapping.get("sha256")):
        reasons.append("mapping sha256 must be a sha256 digest")
    profile = lock.get("accepted_runtime_composition_profile_sha256")
    if profile is not None and not _is_digest(profile):
        reasons.append("accepted runtime composition profile pin must be unset or a sha256 digest")
    return tuple(reasons)


def validate_build_evidence(lock: dict[str, Any], evidence: Any) -> tuple[str, ...]:
    """The go-fabric build record must name the module, release tag and checksum the lock names."""
    if not isinstance(evidence, dict):
        return ("build evidence is absent or malformed",)
    if evidence.get("schema_version") != 1:
        return ("build evidence schema is unsupported",)
    if any(evidence.get(field) != lock["sdk"][field] for field in SDK_PIN_FIELDS):
        return ("build evidence is for a different go-fabric release",)
    return ()


def _epm_reasons(epm: Any) -> list[str]:
    reasons: list[str] = []
    power_mode = epm["power_mode"]
    if not isinstance(power_mode, dict) or set(power_mode) != {"value"} or not _is_nonempty_text(power_mode["value"]) or power_mode["value"].lower() == "unknown":
        reasons.append("mandatory PowerMode is absent or unknown")
    count = epm["number_of_measurement_types"]
    if not isinstance(count, dict) or set(count) != {"value"} or isinstance(count["value"], bool) or not isinstance(count["value"], int) or count["value"] < 1:
        reasons.append("mandatory NumberOfMeasurementTypes is absent")
        expected_accuracy_count = None
    else:
        expected_accuracy_count = count["value"]
    accuracy = epm["accuracy"]
    if not isinstance(accuracy, list) or not accuracy:
        reasons.append("mandatory Accuracy list is empty or malformed")
    else:
        if expected_accuracy_count is not None and len(accuracy) != expected_accuracy_count:
            reasons.append("Accuracy does not match NumberOfMeasurementTypes")
        for item in accuracy:
            if not isinstance(item, dict) or set(item) != {"measurement_type"} or not _is_nonempty_text(item["measurement_type"]):
                reasons.append("mandatory Accuracy entry is absent")
                break
    active_power = epm["active_power"]
    if not isinstance(active_power, dict) or set(active_power) != {"value"}:
        reasons.append("mandatory nullable ActivePower metadata is absent")
    return reasons


def validate_composition(lock: dict[str, Any], composition: Any) -> tuple[str, ...]:
    """Validate the composition file: a schema version and one or more profile entries."""
    if not isinstance(composition, dict):
        return ("accepted runtime-composition profile is absent or malformed",)
    if set(composition) != COMPOSITION_FILE_FIELDS:
        return ("runtime-composition file has an unreviewed shape",)
    profiles = composition["profiles"]
    if type(composition["schema_version"]) is not int or composition["schema_version"] != 1:
        return ("runtime-composition file schema is unsupported",)
    if not isinstance(profiles, list) or not profiles:
        return ("runtime-composition file carries no profile entries",)
    reasons: list[str] = []
    for profile in profiles:
        for reason in _validate_profile(lock, profile):
            if reason not in reasons:
                reasons.append(reason)
    return tuple(reasons)


def _validate_profile(lock: dict[str, Any], composition: Any) -> tuple[str, ...]:
    if not isinstance(composition, dict):
        return ("accepted runtime-composition profile is absent or malformed",)
    if set(composition) != PROFILE_FIELDS:
        return ("runtime-composition profile has an unreviewed shape",)
    reasons = _identity_reasons(lock, composition)
    reasons.extend(_cluster_reasons(composition["clusters"]))
    epm = composition["electrical_power_measurement"]
    if not isinstance(epm, dict) or set(epm) != {"power_mode", "number_of_measurement_types", "accuracy", "active_power"}:
        return tuple(reasons + ["Electrical Power Measurement metadata is malformed"])
    return tuple(reasons + _epm_reasons(epm))


def _identity_reasons(lock: dict[str, Any], composition: dict[str, Any]) -> list[str]:
    reasons: list[str] = []
    if composition["schema_version"] != 1:
        reasons.append("runtime-composition schema is unsupported")
    if composition["status"] != "accepted":
        reasons.append("runtime-composition profile is not accepted")
    if not _is_nonempty_text(composition["profile_id"]):
        reasons.append("runtime-composition profile has no identity")
    if any(composition[field] != lock["sdk"][field] for field in SDK_PIN_FIELDS):
        reasons.append("runtime-composition profile pins a different go-fabric release")
    if composition["mapping"] != lock["mapping"]:
        reasons.append("runtime-composition profile pins a different mapping revision")
    if composition["device_type"] != lock["target"]["device_type"]:
        reasons.append("runtime-composition profile pins a different device type")
    return reasons


def _cluster_reasons(clusters: Any) -> list[str]:
    if not isinstance(clusters, list):
        return ["runtime-composition clusters are malformed"]
    reasons: list[str] = []
    found = {item.get("id"): item for item in clusters if isinstance(item, dict) and isinstance(item.get("id"), str)}
    for cluster_id, name in REQUIRED_CLUSTERS.items():
        cluster = found.get(cluster_id)
        if not isinstance(cluster, dict) or cluster.get("name") != name:
            reasons.append(f"mandatory {name} composition metadata is absent")
    return reasons


def _composition_digest(composition: Any) -> str:
    """Hash an exact canonical profile; the lock pins it."""
    canonical = json.dumps(composition, sort_keys=True, separators=(",", ":"), ensure_ascii=True)
    return hashlib.sha256(canonical.encode("utf-8")).hexdigest()


def evaluate(lock: Any, composition: Any, build_evidence: Any) -> Decision:
    """Permit endpoint allocation only when the pinned mapping, profile and go-fabric build agree."""
    lock_reasons = validate_lock(lock)
    if lock_reasons:
        return Decision(False, lock_reasons)
    assert isinstance(lock, dict)
    profile_pin = lock["accepted_runtime_composition_profile_sha256"]
    if profile_pin is not None and _composition_digest(composition) != profile_pin:
        return Decision(False, ("runtime-composition profile differs from the reviewed pinned profile",))
    if lock["mapping"] is None:
        return Decision(False, ("no reviewed mapping is pinned",))
    reasons = list(validate_build_evidence(lock, build_evidence))
    reasons.extend(validate_composition(lock, composition))
    return Decision(not reasons, tuple(reasons))
