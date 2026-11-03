"""Browser adapter cannot silently skip a missing tool or failed regression."""

import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
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
                with patch.object(browser, "run_owned", side_effect=[None, None, browser.CheckError("test failure")]) as run:
                    self.assertEqual(browser.main([directory + "/evidence"]), 1)
                    command = run.call_args.args[0]
                    self.assertEqual(command[:2], ["node", str(browser.ROOT / "web/tests/sandbox-send.browser.mjs")])

    def test_invalid_timeout_fails(self):
        for value in ("nan", "inf", "0", "301", "not-a-number"):
            with self.subTest(value=value), tempfile.TemporaryDirectory() as directory:
                with patch.dict(os.environ, {"BROWSER_CHECK_TIMEOUT_SECONDS": value}):
                    self.assertEqual(browser.main([directory + "/evidence"]), 1)


@unittest.skipUnless(sys.platform == "linux", "process-tree assertions require Linux /proc")
class BrowserProcessTests(unittest.TestCase):
    """Real executable failure fixtures, never substitutes for browser acceptance."""

    def assert_stopped(self, pid):
        deadline = time.monotonic() + 2
        while time.monotonic() < deadline:
            try:
                state = Path(f"/proc/{pid}/stat").read_text().rsplit(")", 1)[1].split()[0]
            except FileNotFoundError:
                return
            if state == "Z":  # Dead; orphan zombies are reaped by the host, not Popen.
                return
            time.sleep(0.02)
        self.fail(f"live descendant {pid} survived owned process-group cleanup")

    def exercise(self, mode, interrupt=None):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            evidence = directory / "evidence"
            fixture = directory / "chromium"
            fixture.write_text(f'''#!{sys.executable}
import os, pathlib, signal, subprocess, sys, time
if "--version" in sys.argv:
    print("FAILURE CONTAINMENT FIXTURE ONLY")
    sys.exit(0)
directory = pathlib.Path(next(arg.split("=", 1)[1] for arg in sys.argv if arg.startswith("--user-data-dir="))).parent
child = subprocess.Popen([sys.executable, "-c", "import signal,time; signal.signal(signal.SIGTERM, signal.SIG_IGN); print('ready', flush=True); time.sleep(300)"], stdout=subprocess.PIPE, text=True)
assert child.stdout.readline().strip() == "ready"
(directory / "fixture-pids.json").write_text(__import__("json").dumps([os.getpid(), child.pid, os.getpgrp()]))
if {mode!r} == "crash":
    os.kill(os.getpid(), signal.SIGTERM)
while True:
    time.sleep(1)
''')
            fixture.chmod(0o700)
            timeout = 8 if mode == "crash" or interrupt is not None else 1
            env = dict(os.environ, CHROMIUM=str(fixture), BROWSER_CHECK_TIMEOUT_SECONDS=str(timeout),
                       GOPROXY="off", GOSUMDB="off", GOTOOLCHAIN="local", GOTELEMETRY="off")
            with (directory / "adapter.log").open("w+") as log:
                # Outer group/bound is test safety only; adapter must finish before it.
                process = subprocess.Popen([sys.executable, str(browser.ROOT / "scripts/browser_check.py"), str(evidence)],
                                           env=env, stdout=log, stderr=log, start_new_session=True)
                pids = []
                try:
                    if interrupt is not None:
                        deadline = time.monotonic() + 15
                        while not (evidence / "fixture-pids.json").exists() and time.monotonic() < deadline:
                            if process.poll() is not None:
                                break
                            time.sleep(0.02)
                        self.assertTrue((evidence / "fixture-pids.json").exists(), "fixture did not start")
                        process.send_signal(interrupt)
                    try:
                        code = process.wait(timeout=25)
                    except subprocess.TimeoutExpired:
                        self.fail("adapter hung on real executable failure; outer safety timeout expired")
                    log.seek(0)
                    output = log.read()
                    self.assertEqual(code, 128 + interrupt if interrupt else 1, output)
                    report = json.loads((evidence / "adapter-results.json").read_text())
                    self.assertEqual(report["status"], "failed")
                    stage = report["stages"][-1]
                    self.assertEqual(stage["stage"], "trusted-input-regression")
                    self.assertEqual(stage["outcome"], "interrupted" if interrupt else "timeout")
                    self.assertEqual(stage["timeoutSeconds"], timeout)
                    self.assertTrue(stage["cleanup"]["leaderReaped"])
                    self.assertTrue(stage["cleanup"]["sigkillSent"], "SIGTERM-ignoring child requires escalation")
                    pids = json.loads((evidence / "fixture-pids.json").read_text())
                    for pid in pids[:2]:
                        self.assert_stopped(pid)
                    self.assertEqual(pids[2], stage["pid"], "Chromium must inherit the Node-owned group")
                    if mode == "crash" and interrupt is None:
                        self.assertIn("Timeout: Chromium startup", output)
                    self.assertNotIn("browser: PASS", output)
                finally:
                    # Also clean the deliberately broken pre-fix adapter safely.
                    if (evidence / "fixture-pids.json").exists():
                        pids = json.loads((evidence / "fixture-pids.json").read_text())
                    for group in {process.pid, *pids[2:]}:
                        try:
                            os.killpg(group, signal.SIGKILL)
                        except ProcessLookupError:
                            pass
                    process.wait(timeout=3)

    def test_signal_crashed_executable_cleanup_hang_is_bounded(self):
        self.exercise("crash")

    def test_hanging_executable_and_sigterm_ignoring_descendant_are_killed(self):
        self.exercise("hang")

    def test_sigterm_interrupt_cleans_owned_group(self):
        self.exercise("hang", signal.SIGTERM)

    def test_sigint_interrupt_cleans_owned_group(self):
        self.exercise("hang", signal.SIGINT)


if __name__ == "__main__":
    unittest.main()
