"""Browser adapter cannot silently skip a missing tool or failed regression."""

import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import browser_check as browser


class BrowserAdapterTests(unittest.TestCase):
    def test_missing_browser_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            with patch.dict(os.environ, {"CHROMIUM": directory + "/absent"}):
                self.assertEqual(browser.main([directory + "/evidence"]), 1)

    def test_existing_evidence_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            self.assertEqual(browser.main([directory]), 1)

    def test_real_script_selected_and_failure_propagates(self):
        with tempfile.TemporaryDirectory() as directory:
            executable = Path(directory) / "chromium"
            executable.write_text("fixture")
            executable.chmod(0o700)
            with patch.dict(os.environ, {"CHROMIUM": str(executable)}):
                with patch.object(browser, "run", side_effect=["", "", browser.CheckError("test failure")]) as run:
                    self.assertEqual(browser.main([directory + "/evidence"]), 1)
                    command = run.call_args.args[0]
                    self.assertEqual(command[:2], ["node", str(browser.ROOT / "web/tests/sandbox-send.browser.mjs")])


if __name__ == "__main__":
    unittest.main()
