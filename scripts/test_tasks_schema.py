"""Full-schema regressions use isolated ledgers, never rewrite shared state."""

import builtins
import copy
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from test_check import check


class TaskSchemaTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        (self.root / ".harness").mkdir()
        self.schema = json.loads((check.ROOT / ".harness/tasks.schema.json").read_text())
        (self.root / ".harness/tasks.schema.json").write_text(json.dumps(self.schema))
        self.task = {
            "id": "CI-01", "title": "Validation", "status": "claimed", "goal": "Validate",
            "acceptanceCriteria": ["Reject invalid records"], "context": "project-foundation",
            "owner": "foundation", "allowedPaths": ["scripts/**"], "contractRefs": [],
            "invariants": [], "checks": ["static"], "dependencies": [], "evidence": [],
        }
        self.ledger = {"schemaVersion": 1, "tasks": [self.task]}

    def validate(self, ledger):
        (self.root / ".harness/tasks.json").write_text(json.dumps(ledger))
        check.validate_tasks(self.root)

    def test_valid_states_and_optional_fields(self):
        for status in ("pending", "claimed", "blocked", "completed"):
            with self.subTest(status=status):
                self.task.update(status=status, blocker="fixture blocker", nextAction="Review",
                                 evidence=["fixture.log"])
                self.validate(self.ledger)

    def test_every_required_field_is_enforced(self):
        for field in self.schema["$defs"]["task"]["required"]:
            with self.subTest(field=field):
                ledger = copy.deepcopy(self.ledger)
                del ledger["tasks"][0][field]
                with self.assertRaises(check.CheckError):
                    self.validate(ledger)

    def test_negative_types_enums_arrays_and_unknown_properties(self):
        changes = [("status", "accepted"), ("id", ""), ("owner", 12),
                   ("allowedPaths", []), ("allowedPaths", [""]),
                   ("acceptanceCriteria", []), ("checks", ["browser"]),
                   ("dependencies", [1]), ("evidence", "log"), ("extra", True),
                   ("blocker", ""), ("nextAction", "")]
        for field, value in changes:
            with self.subTest(field=field, value=value):
                ledger = copy.deepcopy(self.ledger)
                ledger["tasks"][0][field] = value
                with self.assertRaises(check.CheckError):
                    self.validate(ledger)
        for ledger in ({"schemaVersion": 2, "tasks": []}, {"schemaVersion": 1},
                       {"schemaVersion": 1, "tasks": [], "extra": True},
                       {"schemaVersion": 1, "tasks": {}}, []):
            with self.subTest(ledger=ledger), self.assertRaises(check.CheckError):
                self.validate(ledger)

    def test_conditionals_and_all_records_are_validated_without_mutation(self):
        for status in ("blocked", "completed"):
            ledger = copy.deepcopy(self.ledger)
            ledger["tasks"].append(dict(self.task, id="CI-02", status=status))
            with self.subTest(status=status), self.assertRaisesRegex(check.CheckError, "/tasks/1"):
                self.validate(ledger)
            self.assertEqual(json.loads((self.root / ".harness/tasks.json").read_text()), ledger)
        self.task.update(status="blocked", blocker="Waiting")
        with self.assertRaisesRegex(check.CheckError, "nextAction"):
            self.validate(self.ledger)

    def test_missing_dependency_fails_explicitly(self):
        original = builtins.__import__

        def importing(name, *args, **kwargs):
            if name == "jsonschema":
                raise ImportError("fixture missing dependency")
            return original(name, *args, **kwargs)

        with patch("builtins.__import__", side_effect=importing):
            with self.assertRaisesRegex(check.CheckError, "missing JSON Schema dependency"):
                check.validate_tasks(self.root)


if __name__ == "__main__":
    unittest.main()
