"""Measure the shipping Windows executable with a disposable synthetic notebook."""
import ctypes
from ctypes import wintypes
import json
import os
from pathlib import Path
import platform
import sqlite3
import subprocess
import tempfile
import time
import urllib.request

ROOT = Path(__file__).resolve().parent.parent
EXE = ROOT / 'bin' / 'wordweave.exe'
TYPES = ['形音相似', '近义', '反义', '词族', '屈折变化', '拼写变体', '语义相关']
STAMP = '2026-01-01T00:00:00Z'


def fixture(path):
    db = sqlite3.connect(path)
    for sql in sorted((ROOT / 'internal/app/migrations').glob('*.sql')):
        db.executescript(sql.read_text(encoding='utf-8'))
    db.executescript('CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP); INSERT INTO schema_migrations(version) VALUES (1), (2);')
    db.executemany('INSERT INTO entries(id, spelling, spelling_key, definition, created_at) VALUES (?, ?, ?, ?, ?)',
        ((i, f'word{i:05}', f'word{i:05}', f'释义 {i:05} example definition for learning', STAMP) for i in range(1, 10001)))
    for i in range(1, 30001):
        n = 2 + (i % 9)
        start = (i - 1) % 10000
        step = 1 + (i - 1) // 10000
        members = sorted((start + j * step) % 10000 + 1 for j in range(n))
        key = ','.join(map(str, members))
        db.execute('INSERT INTO relation_groups(id, type, member_key, note, created_at) VALUES (?, ?, ?, ?, ?)',
            (i, TYPES[(i - 1) % 7], key, f'关系说明 {i:05} example comparison', STAMP))
        db.executemany('INSERT INTO group_members(group_id, entry_id) VALUES (?, ?)', ((i, m) for m in members))
    db.commit()
    print('fixture:', db.execute('SELECT count(*), avg(length(definition)) FROM entries').fetchone(),
          db.execute('SELECT count(*), avg(length(note)) FROM relation_groups').fetchone(),
          db.execute('SELECT count(*) FROM group_members').fetchone(), flush=True)
    db.close()


class Memory(ctypes.Structure):
    _fields_ = [('cb', wintypes.DWORD), ('PageFaultCount', wintypes.DWORD),
                ('PeakWorkingSetSize', ctypes.c_size_t), ('WorkingSetSize', ctypes.c_size_t),
                ('QuotaPeakPagedPoolUsage', ctypes.c_size_t), ('QuotaPagedPoolUsage', ctypes.c_size_t),
                ('QuotaPeakNonPagedPoolUsage', ctypes.c_size_t), ('QuotaNonPagedPoolUsage', ctypes.c_size_t),
                ('PagefileUsage', ctypes.c_size_t), ('PeakPagefileUsage', ctypes.c_size_t)]


def memory(process):
    counters = Memory()
    counters.cb = ctypes.sizeof(counters)
    if not ctypes.windll.psapi.GetProcessMemoryInfo(wintypes.HANDLE(process._handle), ctypes.byref(counters), counters.cb):
        raise ctypes.WinError()
    return round(counters.WorkingSetSize / 1048576, 1), round(counters.PeakWorkingSetSize / 1048576, 1)


def main():
    if os.name != 'nt' or not EXE.exists():
        raise SystemExit('Build bin/wordweave.exe on Windows first.')
    print('platform:', platform.platform(), platform.machine(), platform.processor(), flush=True)
    with tempfile.TemporaryDirectory(prefix='wordweave-measure-') as directory:
        fixture(Path(directory) / 'wordweave.db')
        start = time.perf_counter()
        process = subprocess.Popen([str(EXE), '-listen', '127.0.0.1:0', '-data-dir', directory],
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding='utf-8')
        try:
            address = process.stdout.readline().strip().removeprefix('Wordweave: ')
            if not address.startswith('http://'):
                raise RuntimeError(process.stderr.read())
            print('startup ms:', round((time.perf_counter() - start) * 1000, 1), 'working set / peak MiB:', memory(process), flush=True)
            time.sleep(5)
            print('idle working set / lifetime peak MiB:', memory(process), flush=True)
            requests = [('/api/entries/5000', None), ('/api/search?q=word05000', None),
                        ('/api/search?q=example%20comparison%2005000', None),
                        ('/api/entries/5000/groups', None),
                        ('/api/entries/5000', {'spelling': 'word05000', 'definition': 'edited definition'})]
            for path, data in requests:
                elapsed = []
                for _ in range(5):
                    body = json.dumps(data).encode() if data is not None else None
                    request = urllib.request.Request(address + path, body, {'Content-Type': 'application/json'}, method='PUT' if body else 'GET')
                    before = time.perf_counter()
                    with urllib.request.urlopen(request, timeout=60) as response:
                        response.read()
                    elapsed.append(round((time.perf_counter() - before) * 1000, 1))
                print(path, 'ms:', elapsed, 'working set / lifetime peak MiB:', memory(process), flush=True)
        finally:
            process.terminate()
            process.wait(timeout=10)


if __name__ == '__main__':
    main()
