#!/usr/bin/env python3
"""Small stdlib check adapter. No runtime stubs, installs or source mutation."""

import argparse
import ast
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import unittest


sys.dont_write_bytecode = True
ROOT = Path(__file__).resolve().parents[1]
CHECKS = ("format", "lint", "static", "unit", "contract", "fuzz-smoke", "fuzz-full")
MANDATORY_FUZZ = {"FuzzValidateMessage", "FuzzDecodeProvider", "FuzzDecodeTurso"}
SKIP = {".git", ".harness", ".scratch", "node_modules", "__pycache__", "data"}
TEXT_SUFFIXES = {".go", ".js", ".mjs", ".json", ".py", ".md", ".sql", ".html", ".css"}


class CheckError(Exception):
    pass


def require(condition, message):
    if not condition:
        raise CheckError(message)


def run(args, root=ROOT, capture=False):
    # Feature checks must not inherit live service credentials from a shell.
    env = dict(os.environ)
    env.update(OPENAI_API_KEY="", TURSO_AUTH_TOKEN="", TURSO_DATABASE_URL="", TMPDIR="/tmp/opencode")
    result = subprocess.run(args, cwd=root, env=env, text=True,
                            stdout=subprocess.PIPE if capture else None,
                            stderr=subprocess.PIPE if capture else None, check=False)
    if result.returncode:
        detail = (result.stdout or "") + (result.stderr or "") if capture else ""
        raise CheckError(f"command failed ({result.returncode}): {' '.join(args)}\n{detail}")
    return result.stdout or ""


def files(root=ROOT):
    return sorted(p for p in root.rglob("*") if p.is_file()
                  and not (set(p.relative_to(root).parts) & SKIP))


def format_check(root=ROOT):
    sources = files(root)
    bad = []
    for path in sources:
        if path.suffix not in TEXT_SUFFIXES and path.name not in {"Makefile", "go.mod", ".env.example"}:
            continue
        text = path.read_text(encoding="utf-8")
        if text and (not text.endswith("\n") or "\r" in text
                     or any(line.rstrip(" \t") != line for line in text.splitlines())):
            bad.append(str(path.relative_to(root)))
    require(not bad, "trailing whitespace/CRLF/missing final newline: " + ", ".join(bad))
    go = [str(p.relative_to(root)) for p in sources if p.suffix == ".go"]
    if go:
        unformatted = run(["gofmt", "-l", *go], root, capture=True).strip()
        require(not unformatted, "gofmt required (check did not edit):\n" + unformatted)


def syntax_check(root=ROOT):
    for path in files(root):
        if path.suffix in {".js", ".mjs"}:
            run(["node", "--check", str(path.relative_to(root))], root)
        elif path.suffix == ".py":
            ast.parse(path.read_text(encoding="utf-8"), filename=str(path))
        elif path.suffix == ".json":
            json.loads(path.read_text(encoding="utf-8"))


def validate_manifest(root=ROOT):
    """Targeted project invariants, NOT a general JSON Schema implementation."""
    manifest = json.loads((root / "harness.json").read_text(encoding="utf-8"))
    require(manifest.get("schemaVersion") == 1, "manifest schemaVersion must be 1")
    require(manifest.get("status") == "active", "contract/check baseline must be active")
    for key in ("name", "goal"):
        require(isinstance(manifest["project"].get(key), str) and manifest["project"][key].strip(),
                f"project.{key} must be nonempty")
    validation = manifest["validation"]
    require(validation["requiredChecks"] == list(CHECKS), "requiredChecks must list the seven common checks")
    require(validation["adapter"]["status"] == "configured" and validation["adapter"]["name"],
            "active adapter must be configured and named")
    require(set(validation["commands"]) == set(CHECKS), "all seven commands must be configured")
    for name in CHECKS:
        require(validation["commands"][name] == ["make", name], f"unexpected command for {name}")
    makefile = (root / "Makefile").read_text(encoding="utf-8")
    require("format lint static unit contract fuzz-smoke fuzz-full:" in makefile
            and "python3 scripts/check.py $@" in makefile, "Makefile must route the seven checks")
    architecture = manifest["architecture"]
    require((root / architecture["contextMap"]).is_file(), "missing context map")
    contexts = architecture["contexts"]
    require({c["id"] for c in contexts} ==
            {"project-foundation", "chat-backend", "chat-widget", "coordination"}, "missing/duplicate contexts")
    require(len(contexts) == 4, "context IDs must be unique")
    owned = []
    for context in contexts:
        for key in ("owner", "communicationPath", "contract"):
            require(isinstance(context[key], str) and context[key].strip(), f"missing context {key}")
        require((root / context["contract"]).is_file(), "missing published contract")
        require(isinstance(context["collaborators"], list), "collaborators must be a list")
        require(context["ownedPaths"], "context must own paths")
        for path in context["ownedPaths"]:
            require(isinstance(path, str) and path and not path.startswith("/")
                    and ".." not in Path(path).parts, "unsafe owned path")
            prefix = path.removesuffix("/**")
            for prior in owned:
                require(prefix != prior and not prefix.startswith(prior + "/")
                        and not prior.startswith(prefix + "/"), f"overlapping ownership: {path}, {prior}")
            owned.append(prefix)
    for contract in ("chat.md", "backend.md"):
        require((root / "docs/contracts" / contract).is_file(), f"missing {contract}")


def setup_tests(root=ROOT):
    suite = unittest.defaultTestLoader.discover(str(root / "scripts"), pattern="test_*.py")
    require(suite.countTestCases() > 0, "missing check-adapter tests")
    require(unittest.TextTestRunner(verbosity=2).run(suite).wasSuccessful(), "check-adapter tests failed")


def go_packages(root=ROOT):
    packages = run(["go", "list", "./..."], root, capture=True).splitlines()
    require(packages, "application Go packages missing (feature implementation pending)")
    return packages


def go_targets(prefix, root=ROOT):
    targets = []
    for package in go_packages(root):
        listing = run(["go", "test", "-list", "^" + prefix, package], root, capture=True)
        for line in listing.splitlines():
            if re.fullmatch(prefix + r"[A-Za-z0-9_]+", line):
                targets.append((package, line))
    return targets


def node_tests(contract=False, root=ROOT):
    paths = sorted((root / "web").rglob("*.test.mjs"))
    selected = [str(p.relative_to(root)) for p in paths
                if p.name.endswith(".contract.test.mjs") == contract]
    require(selected, "missing widget " + ("contract" if contract else "unit") + " tests (*.test.mjs)")
    run(["node", "--test", *selected], root)


def fuzz_plan(targets, seconds):
    require(targets, "no Go fuzz targets found (feature implementation pending)")
    names = {name for _, name in targets}
    missing = MANDATORY_FUZZ - names
    require(not missing, "mandatory backend fuzz targets missing: " + ", ".join(sorted(missing)))
    return [["go", "test", package, "-run=^$", "-fuzz=^" + name + "$",
             f"-fuzztime={seconds}s", "-parallel=2"] for package, name in sorted(set(targets))]


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("check", choices=(*CHECKS, "setup"))
    choice = parser.parse_args(argv).check
    try:
        if choice == "format":
            format_check()
        elif choice == "setup":
            format_check()
            syntax_check()
            validate_manifest()
            setup_tests()
        elif choice == "lint":
            syntax_check()
            go_packages()
            run(["go", "vet", "./..."])
        elif choice == "static":
            validate_manifest()
            syntax_check()
            go_packages()
            run(["go", "test", "-run=^$", "./..."])
        elif choice == "unit":
            setup_tests()
            require(go_targets("Test"), "missing application Go tests")
            run(["go", "test", "-race", "./..."])
            node_tests()
        elif choice == "contract":
            validate_manifest()
            require(go_targets("TestContract"), "missing backend TestContract… tests")
            run(["go", "test", "-race", "-run=^TestContract", "./..."])
            node_tests(contract=True)
        else:
            seconds = 2 if choice == "fuzz-smoke" else 30
            targets = go_targets("Fuzz")
            for command in fuzz_plan(targets, seconds):
                print("+ " + " ".join(command), flush=True)
                run(command)
        print(f"{choice}: PASS", flush=True)
        return 0
    except (CheckError, OSError, ValueError, KeyError, TypeError, SyntaxError) as error:
        print(f"{choice}: FAIL: {error}", file=sys.stderr, flush=True)
        return 1


if __name__ == "__main__":
    sys.exit(main())
