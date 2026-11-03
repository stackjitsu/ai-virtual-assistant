import hashlib
import json
from pathlib import Path
import subprocess
from jsonschema import Draft7Validator

own = Path('.scratch/automation/ci-re-review/evidence')
fixture = own / 'crash-fixture'
report = json.loads((fixture / 'adapter-results.json').read_text())
assert report['status'] == 'failed'
stage = report['stages'][-1]
assert stage['stage'] == 'trusted-input-regression'
assert stage['outcome'] == 'timeout' and stage['timeoutSeconds'] == 8
assert all(stage['cleanup'][key] for key in ('sigtermSent', 'sigkillSent', 'leaderReaped'))
assert stage['elapsedSeconds'] < 15
assert not (fixture / 'results.json').exists()
assert 'Timeout: Chromium startup' in (own / 'crash-fixture.log').read_text()
pids = json.loads((fixture / 'fixture-pids.json').read_text())
assert pids['group'] == stage['pid']
for name in ('parent', 'ignoresSigtermChild'):
    pid = pids[name]
    path = Path(f'/proc/{pid}/stat')
    state = path.read_text().rsplit(')', 1)[1].split()[0] if path.exists() else 'absent'
    print(name, pid, state)
    assert state in ('absent', 'Z')
# Inspect all surviving members of each recorded owned process group.
for evidence in ('crash-fixture', 'browser'):
    report = json.loads((own / evidence / 'adapter-results.json').read_text())
    groups = {stage['pid'] for stage in report['stages']}
    live = []
    for path in Path('/proc').glob('[0-9]*/stat'):
        try:
            fields = path.read_text().rsplit(')', 1)[1].split()
        except (FileNotFoundError, ProcessLookupError):
            continue
        if int(fields[2]) in groups and fields[0] != 'Z':
            live.append(str(path))
    assert not live, live
    print(evidence, 'no live owned-group processes')
for name in ('.harness/tasks.json', '.harness/tasks.schema.json', 'web/tests/sandbox-send.browser.mjs'):
    base = subprocess.check_output(['git', 'show', 'HEAD:' + name])
    assert base == Path(name).read_bytes()
    print(name, 'byte-identical to HEAD', hashlib.sha256(base).hexdigest())
ledger = json.loads(Path('.harness/tasks.json').read_text())
schema = json.loads(Path('.harness/tasks.schema.json').read_text())
Draft7Validator.check_schema(schema)
errors = sorted(Draft7Validator(schema).iter_errors(ledger), key=lambda e: list(e.path))
assert [list(e.path) for e in errors] == [['tasks', n, 'evidence'] for n in range(21, 25)]
for error in errors:
    task = ledger['tasks'][error.path[1]]
    assert task['status'] == 'completed' and task['evidence'] == []
    print(list(error.path), task['id'], task['status'], error.message)
p = (own / 'implementation.diff').read_bytes()
q = Path('.scratch/automation/ci-repair/evidence/implementation.diff').read_bytes()
normalized = p.replace(b'diff --git c/', b'diff --git i/').replace(b'--- c/', b'--- i/')
assert normalized == q
print('Exact git diff HEAD fingerprint', hashlib.sha256(p).hexdigest())
print('Only source-prefix differs from repair diff; normalized fingerprint', hashlib.sha256(normalized).hexdigest())
before = json.loads((own / 'before-sha256.json').read_text())
after = {name: hashlib.sha256(Path(name).read_bytes()).hexdigest() for name in before}
assert before == after
(own / 'after-sha256.json').write_text(json.dumps(after, indent=2) + '\n')
print('All', len(before), 'fingerprinted source/existing evidence files unchanged')
print('PASS')
