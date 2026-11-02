"""Read-only integration scope and actual browser SQLite evidence audit."""
import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess

ROOT = Path.cwd()
E = ROOT / '.scratch/automation/ui-rating-integration'
UI = '6507570e70151d14f68f8e89944b54514613a055'
RATING = 'eface12be8e508a31efa6a8d1ceb7bd2daf07be5'


def git(*args):
    return subprocess.check_output(['git', *args])


def unchanged(ref, paths):
    for path in paths:
        assert (ROOT / path).read_bytes() == git('show', f'{ref}:{path}'), path
    return paths


backend = git('ls-tree', '-r', '--name-only', RATING, 'cmd', 'internal', 'database').decode().splitlines()
harness = git('ls-tree', '-r', '--name-only', UI, '.harness').decode().splitlines()
unchanged_rating_web = ['web/assets/protocol.mjs', 'web/tests/rating.test.mjs']
unchanged_ui_web = ['web/README.md', 'web/assets/loader.js', 'web/tests/embedding.contract.test.mjs',
                    'web/tests/escalation.test.mjs', 'web/tests/widget.test.mjs',
                    'web/tests/sandbox-send.browser.mjs', 'web/tests/widget-visual.browser.mjs']
summary = {
    'ui_parent': UI,
    'rating_parent': RATING,
    'backend_byte_identical_to_rating': unchanged(RATING, backend),
    'coordinator_harness_byte_identical_to_ui': unchanged(UI, harness),
    'rating_web_byte_identical_to_rating': unchanged(RATING, unchanged_rating_web),
    'approved_ui_web_byte_identical_to_ui': unchanged(UI, unchanged_ui_web),
    'chat_contract_byte_identical_to_ui': unchanged(UI, ['docs/contracts/chat.md']),
}
# The application runtime resolution must be precisely the accepted rating
# implementation plus the approved UI's already-reviewed close/label changes.
ui_core = git('show', f'{UI}:web/assets/widget-core.mjs').decode()
# Check the one conflicting element-list union and exact approved close path.
assert "'message-form', 'chat-close'," in (ROOT / 'web/assets/widget-core.mjs').read_text()
close_start = ui_core.index('  // Both controls close the host window')
close_end = ui_core.index('  // The stored recorded_at', close_start)
assert ui_core[close_start:close_end] in (ROOT / 'web/assets/widget-core.mjs').read_text()
summary['approved_close_path_preserved_verbatim'] = True

report = json.loads((E / 'rating-round1/results.json').read_text())
assert not report.get('fatal') and report['summary'] == {'passed': 61, 'failed': 0}
responses = [r['body'] for r in report['responses'] if r.get('source') == 'browser'
             and r['url'].endswith('/rating') and r['status'] == 200]
assert len(responses) == 4
assert responses[1] == responses[2] == responses[3]
db = sqlite3.connect(f'file:{E / "rating-round1/assistant.sqlite"}?mode=ro', uri=True)
row = db.execute('SELECT rating, comment, rated_at FROM ratings').fetchall()
counts = {name: db.execute(f'SELECT count(*) FROM {name}').fetchone()[0]
          for name in ['conversations', 'turns', 'ratings', 'escalation_tickets']}
db.close()
assert row == [(responses[1]['rating'], responses[1]['comment'], responses[1]['rated_at'])]
assert counts == {'conversations': 2, 'turns': 1, 'ratings': 1, 'escalation_tickets': 0}
summary['rating_database'] = {'counts': counts, 'final_row': responses[1],
                              'replay_and_process_restart_envelopes_unchanged': True}
summary['browser_checks'] = {}
for scenario in ['rating', 'sandbox-send', 'widget-visual']:
    result = json.loads((E / f'{scenario}-round1/results.json').read_text())
    if isinstance(result, list):
        checks = {'passed': sum(bool(entry['pass']) for entry in result),
                  'failed': sum(not entry['pass'] for entry in result)}
    else:
        assert not result.get('fatal')
        checks = result['summary']
    assert checks['failed'] == 0
    summary['browser_checks'][scenario] = checks

manifest = []
for path in git('ls-files', 'web', 'cmd', 'internal', 'database', 'docs/contracts').decode().splitlines():
    manifest.append(f'{hashlib.sha256((ROOT / path).read_bytes()).hexdigest()}  {path}')
(E / 'logs/source.sha256').write_text('\n'.join(manifest) + '\n')
summary['source_manifest_sha256'] = hashlib.sha256((E / 'logs/source.sha256').read_bytes()).hexdigest()
(E / 'logs/scope-and-storage-audit.json').write_text(json.dumps(summary, indent=2) + '\n')
print(json.dumps({'scope_audit': 'PASS', 'storage_audit': 'PASS', 'browser_checks': summary['browser_checks']}))
