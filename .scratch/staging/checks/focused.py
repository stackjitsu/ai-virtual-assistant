"""Read-only deployment checks; not a Docker engine/parser replacement."""
import fnmatch
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

root = Path.cwd()
data = json.loads((root / '.scratch/staging/checks/render-parsed.json').read_text())
assert set(data) == {'services'}
assert len(data['services']) == 1
service = data['services'][0]
assert service['type'] == 'web' and service['runtime'] == 'docker'
assert service['plan'] == 'free' and service['autoDeployTrigger'] == 'off'
assert set(service) == {'type', 'name', 'runtime', 'plan', 'dockerfilePath', 'dockerContext', 'autoDeployTrigger', 'healthCheckPath', 'envVars'}
assert service['healthCheckPath'] == '/widget/widget.css'
assert Path('web/assets/widget.css').is_file()
variables = {entry['key']: entry for entry in service['envVars']}
assert len(variables) == len(service['envVars'])
for key in ('OPENAI_API_KEY', 'TURSO_DATABASE_URL', 'TURSO_AUTH_TOKEN', 'ASSISTANT_ORIGIN', 'WIDGET_PARENT_ORIGINS'):
    assert variables[key] == {'key': key, 'sync': False}
for key, value in {'STORE_MODE': 'turso', 'ASSISTANT_PUBLIC_DEMO': 'true', 'OPENAI_MODEL': 'gemini-2.5-flash-lite', 'OPENAI_BASE_URL': 'https://generativelanguage.googleapis.com/v1beta/openai/chat/completions'}.items():
    assert variables[key] == {'key': key, 'value': value}
assert len(variables) == 9
print('PASS: one free service; no paid resources, hooks, auto deploys or generated secrets; exact operator origins')

docker = Path('Dockerfile').read_text()
assert 'FROM golang:1.26.3-bookworm AS build' in docker
assert 'USER 65532:65532' in docker
assert 'ca-certificates' in docker
assert 'COPY --from=build --chmod=0555 /out/assistant /app/assistant' in docker
assert not re.search(r'^(ARG|ENV|ADD)\s', docker, re.M)
assert not re.search(r'\b(migrate|seed-demo|import-faq)\b', docker)
cmd = json.loads(next(line[4:] for line in docker.splitlines() if line.startswith('CMD ')))
assert cmd[:2] == ['/bin/sh', '-c']
script = cmd[2]
subprocess.run(['/bin/sh', '-n', '-c', script], check=True, timeout=5)
# Substitute only the final executable with a harmless argv/environment reporter.
# This checks shell expansion, not the actual application/container.
with tempfile.TemporaryDirectory(prefix='stage02-shell-') as tmp:
    reporter = Path(tmp) / 'report'
    reporter.write_text('#!/bin/sh\nprintf "%s\\n" "$LISTEN_ADDR" "$@"\n')
    reporter.chmod(0o700)
    assert script.endswith('exec /app/assistant serve')
    mock_script = script.replace('exec /app/assistant serve', f'exec {reporter} serve')
    base = {'PATH': '/usr/bin:/bin', 'ASSISTANT_ORIGIN': 'https://actual-host.invalid', 'WIDGET_PARENT_ORIGINS': 'https://actual-host.invalid', 'LISTEN_ADDR': '127.0.0.1:9'}
    for port in ('10000', '8080', '65535', '1'):
        p = subprocess.run(['/bin/sh', '-c', mock_script], env={**base, 'PORT': port}, capture_output=True, text=True, timeout=5)
        assert p.returncode == 0 and p.stdout == f'0.0.0.0:{port}\nserve\n', (port, p)
    for port in ('', '0', '65536', '-1', 'abc', '10000;touch /tmp/stage02-unwanted', '10000\n'):
        p = subprocess.run(['/bin/sh', '-c', mock_script], env={**base, 'PORT': port}, capture_output=True, timeout=5)
        assert p.returncode != 0
    for key in ('PORT', 'ASSISTANT_ORIGIN', 'WIDGET_PARENT_ORIGINS'):
        env = {**base, 'PORT': '10000'}
        env.pop(key)
        assert subprocess.run(['/bin/sh', '-c', mock_script], env=env, capture_output=True, timeout=5).returncode != 0
print('PASS: nonroot multi-stage structure; CA package; shell syntax, safe PORT expansion, mandatory origins and exec argv')

# Focused review of these simple patterns; NOT native Docker context evaluation.
patterns = [line for line in Path('.dockerignore').read_text().splitlines() if line and not line.startswith('#')]
assert patterns[0] == '**'
allowed_patterns = [p[1:] for p in patterns if p.startswith('!') and not p.endswith('/')]
def admitted(path):
    return any(fnmatch.fnmatchcase(path, p) for p in allowed_patterns) and not path.endswith('_test.go')
tracked = subprocess.run(['git', 'ls-files'], check=True, capture_output=True, text=True, timeout=10).stdout.splitlines()
admitted_paths = [p for p in tracked if admitted(p)]
assert admitted_paths
for path in admitted_paths:
    assert path in ('go.mod', 'go.sum', 'database/embed.go', 'web/embed.go') or path.startswith(('cmd/assistant/', 'internal/chat/', 'database/migrations/', 'web/assets/'))
    assert not any(x in path for x in ('.git', '.env', '.scratch', 'node_modules', '.db', 'testdata', '_test.go'))
for path in ('.git/config', '.env', '.env.example', 'data/assistant.db', 'internal/chat/private.key', 'web/assets/.env', '.scratch/staging/secret', 'node_modules/x.js', 'internal/chat/cache.db', 'web/assets/node_modules/key.js'):
    # fnmatch's slash wildcard differs from Docker; check nested paths separately.
    if path == 'web/assets/node_modules/key.js':
        continue
    assert not admitted(path), path
Path('.scratch/staging/checks/context-source-paths.txt').write_text('\n'.join(admitted_paths) + '\n')
print(f'PASS: deny-default context allowlist reviewed ({len(admitted_paths)} tracked source inputs); no native context proof')
