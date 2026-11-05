#!/usr/bin/env python3
"""Run the existing trusted-input Chromium test, with local SQLite and no AI key."""

import os
from pathlib import Path
import shutil
import sys

from check import CheckError, ROOT, require, run


def main(argv=None):
    args = sys.argv[1:] if argv is None else argv
    try:
        require(len(args) == 1, "usage: python3 scripts/browser_check.py FRESH_EVIDENCE_DIR")
        directory = Path(args[0]).resolve()
        require(not directory.exists(), "browser evidence directory must be fresh")
        script = ROOT / "web/tests/sandbox-send.browser.mjs"
        require(script.is_file(), "missing trusted-input browser regression")
        chromium = os.environ.get("CHROMIUM") or shutil.which("chromium")
        require(chromium and Path(chromium).is_file() and os.access(chromium, os.X_OK),
                "missing executable Chromium; set CHROMIUM to the pinned browser path")
        os.environ["CHROMIUM"] = chromium
        directory.mkdir(parents=True)
        run([chromium, "--version"])
        binary = directory / "assistant"
        run(["go", "build", "-o", str(binary), "./cmd/assistant"])
        run(["node", str(script), str(binary), str(directory)])
        print("browser: PASS", flush=True)
        return 0
    except (CheckError, OSError) as error:
        print(f"browser: FAIL: {error}", file=sys.stderr, flush=True)
        return 1


if __name__ == "__main__":
    sys.exit(main())
