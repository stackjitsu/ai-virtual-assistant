#!/usr/bin/env python3
"""Bound the trusted-input Chromium test, with local SQLite and no AI key."""

import json
import math
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import time

from check import CheckError, ROOT, require


DEFAULT_TIMEOUT_SECONDS = 120
VERSION_TIMEOUT_SECONDS = 15
BUILD_TIMEOUT_SECONDS = 180
CLEANUP_GRACE_SECONDS = 1
REAP_TIMEOUT_SECONDS = 2


class Interrupted(CheckError):
    def __init__(self, signum):
        self.signum = signum
        super().__init__(f"interrupted by {signal.Signals(signum).name}")


def interrupt(signum, _frame):
    raise Interrupted(signum)


def cleanup_group(process):
    """Kill the owned group even if its leader already exited; never wait forever."""
    cleanup = {"sigtermSent": False, "sigkillSent": False, "leaderReaped": False}
    # A second normal termination signal must not interrupt mandatory cleanup.
    previous = {sig: signal.signal(sig, signal.SIG_IGN) for sig in (signal.SIGINT, signal.SIGTERM)}
    try:
        for sig, key in ((signal.SIGTERM, "sigtermSent"), (signal.SIGKILL, "sigkillSent")):
            try:
                os.killpg(process.pid, sig)
                cleanup[key] = True
            except ProcessLookupError:
                break
            if sig == signal.SIGTERM:
                # Reap the leader promptly, but still escalate against descendants.
                try:
                    process.wait(timeout=CLEANUP_GRACE_SECONDS)
                except subprocess.TimeoutExpired:
                    pass
                else:
                    time.sleep(CLEANUP_GRACE_SECONDS)
        try:
            process.wait(timeout=REAP_TIMEOUT_SECONDS)
            cleanup["leaderReaped"] = True
        except subprocess.TimeoutExpired:
            cleanup["error"] = "leader did not exit after SIGKILL within reap bound"
    finally:
        for sig, handler in previous.items():
            signal.signal(sig, handler)
    return cleanup


def run_owned(args, stage, timeout, report):
    """Inherit log streams, strip live credentials, and own one POSIX group."""
    env = dict(os.environ)
    Path("/tmp/opencode").mkdir(parents=True, exist_ok=True)
    env.update(OPENAI_API_KEY="", TURSO_AUTH_TOKEN="", TURSO_DATABASE_URL="", TMPDIR="/tmp/opencode")
    record = {"stage": stage, "timeoutSeconds": timeout, "outcome": "starting"}
    report["stages"].append(record)
    started = time.monotonic()
    process = None
    try:
        process = subprocess.Popen(args, cwd=ROOT, env=env, start_new_session=True)
        record["pid"] = process.pid
        record["outcome"] = "running"
        try:
            code = process.wait(timeout=timeout)
        except subprocess.TimeoutExpired as error:
            record["outcome"] = "timeout"
            raise CheckError(f"{stage} timed out after {timeout:g}s; terminating owned process group {process.pid}") from error
        record["returncode"] = code
        record["outcome"] = "failed" if code else "passed"
        require(code == 0, f"{stage} command failed ({code}): {' '.join(args)}")
    except Interrupted:
        record["outcome"] = "interrupted"
        raise
    except OSError:
        record["outcome"] = "failed"
        raise
    finally:
        if process is not None:
            record["cleanup"] = cleanup_group(process)
        record["elapsedSeconds"] = round(time.monotonic() - started, 3)
    require(record["cleanup"]["leaderReaped"], f"{stage}: process cleanup failed")


def main(argv=None):
    args = sys.argv[1:] if argv is None else argv
    directory = None
    handlers = {}
    report = {"status": "failed", "stages": []}
    try:
        require(len(args) == 1, "usage: python3 scripts/browser_check.py FRESH_EVIDENCE_DIR")
        require(os.name == "posix", "browser adapter requires POSIX process groups")
        try:
            timeout = float(os.environ.get("BROWSER_CHECK_TIMEOUT_SECONDS", DEFAULT_TIMEOUT_SECONDS))
        except ValueError as error:
            raise CheckError("BROWSER_CHECK_TIMEOUT_SECONDS must be a number") from error
        require(math.isfinite(timeout) and 0.1 <= timeout <= 300,
                "BROWSER_CHECK_TIMEOUT_SECONDS must be finite and between 0.1 and 300 seconds")
        destination = Path(args[0]).resolve()
        require(not destination.exists(), "browser evidence directory must be fresh")
        script = ROOT / "web/tests/sandbox-send.browser.mjs"
        require(script.is_file(), "missing trusted-input browser regression")
        chromium = os.environ.get("CHROMIUM") or shutil.which("chromium")
        require(chromium and Path(chromium).is_file() and os.access(chromium, os.X_OK),
                "missing executable Chromium; set CHROMIUM to the pinned browser path")
        os.environ["CHROMIUM"] = chromium
        destination.mkdir(parents=True)
        directory = destination
        handlers = {sig: signal.signal(sig, interrupt) for sig in (signal.SIGINT, signal.SIGTERM)}
        run_owned([chromium, "--version"], "browser-version", VERSION_TIMEOUT_SECONDS, report)
        binary = directory / "assistant"
        run_owned(["go", "build", "-o", str(binary), "./cmd/assistant"], "backend-build", BUILD_TIMEOUT_SECONDS, report)
        run_owned(["node", str(script), str(binary), str(directory)], "trusted-input-regression", timeout, report)
        report["status"] = "passed"
        print("browser: PASS", flush=True)
        return 0
    except (CheckError, OSError) as error:
        report["error"] = str(error)
        print(f"browser: FAIL: {error}", file=sys.stderr, flush=True)
        return 128 + error.signum if isinstance(error, Interrupted) else 1
    finally:
        for sig, handler in handlers.items():
            signal.signal(sig, handler)
        if directory is not None:
            # Separate from Node results.json, which inherited cleanup may never write.
            (directory / "adapter-results.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    sys.exit(main())
