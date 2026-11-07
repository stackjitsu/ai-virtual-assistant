"""Package a source-only presentation snapshot; never include local secrets/data."""
import hashlib
import json
from pathlib import Path
import stat
import subprocess
import zipfile

source = Path('/home/me/Documents/agent-worktrees/assistant-local-demo')
output = Path('/home/me/Documents/formartting/Working_Milestone_Files/share')
output.mkdir(exist_ok=True)
archive = output / 'AI_Virtual_Assistant_Windows_Demo_v2.zip'
manifest_path = output / 'AI_Virtual_Assistant_Windows_Demo_v2.manifest.json'
if archive.exists() or manifest_path.exists():
    raise SystemExit('Refusing to overwrite an existing shared package')

tracked = subprocess.check_output(['git', 'ls-files', '-z'], cwd=source).decode().split('\0')
extras = [
    'cmd/assistant/local_demo.go', 'cmd/assistant/local_demo_test.go',
    'internal/chat/local_demo.go', 'internal/chat/local_demo_test.go',
    'docs/contracts/local-demo.md', 'scripts/demo.sh', 'scripts/demo.ps1',
    'scripts/test_demo.py', 'web/assets/demo-host.html',
]
roots = {'cmd', 'internal', 'database', 'web', 'scripts', 'docs', '.harness'}
root_files = {'README.md', 'AGENTS.md', 'GLOSSARY.md', 'Makefile', 'harness.json',
              'go.mod', 'go.sum', '.gitignore', '.env.example'}
files = []
for name in sorted(set(tracked + extras)):
    if not name:
        continue
    rel = Path(name)
    if rel.parts[0] not in roots and name not in root_files:
        continue
    if any(part in {'.git', '.scratch', '.local-demo', '__pycache__', 'node_modules'}
           for part in rel.parts):
        continue
    if rel.suffix in {'.db', '.sqlite', '.sqlite3', '.exe', '.log', '.png'}:
        raise SystemExit('Unexpected generated file in source whitelist: ' + name)
    if rel.name.startswith('.env') and name != '.env.example':
        raise SystemExit('Unexpected environment file: ' + name)
    path = source / rel
    if path.is_symlink() or not path.is_file():
        raise SystemExit('Missing or symlink source file: ' + name)
    files.append((name, path.read_bytes(), path.stat().st_mode))

required = set(extras)
assert required <= {name for name, _, _ in files}
hashes = {name: hashlib.sha256(data).hexdigest() for name, data, _ in files}
for name, digest in hashes.items():
    if hashlib.sha256((source / name).read_bytes()).hexdigest() != digest:
        raise SystemExit('Source changed during snapshot; rerun after owner stops: ' + name)

with zipfile.ZipFile(archive, 'x', zipfile.ZIP_DEFLATED) as z:
    for name, data, mode in files:
        info = zipfile.ZipInfo('ai-virtual-assistant/' + name)
        info.create_system = 3
        info.external_attr = (stat.S_IFREG | stat.S_IMODE(mode)) << 16
        info.compress_type = zipfile.ZIP_DEFLATED
        z.writestr(info, data)

manifest = {'source_branch': 'feat/one-command-demo', 'source_base': '65471be',
            'files': hashes, 'archive_sha256': hashlib.sha256(archive.read_bytes()).hexdigest(),
            'scope': 'Source-only presentation candidate; no real keys, databases, profiles, git metadata or internal scratch included',
            'windows_note': 'Native Windows execution is not tested on this Linux host'}
manifest_path.write_text(json.dumps(manifest, indent=2) + '\n')
print('Archive:', archive)
print('Source files:', len(files))
print('Bytes:', archive.stat().st_size)
print('SHA256:', manifest['archive_sha256'])
