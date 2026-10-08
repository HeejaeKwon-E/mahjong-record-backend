#!/usr/bin/env python3
"""Rust 실행 없이 원본과 동일한 SQLite SQL의 핵심 동작을 점검합니다.

이 검사는 SQL 문자열과 테이블 스키마를 Python sqlite3로 실행할 뿐,
Rust 컴파일 / Axum HTTP 검사를 대신하지 않습니다.
"""

import re
import sqlite3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
schema_source = (ROOT / 'src/schema/mod.rs').read_text()
repo_source = (ROOT / 'src/repository/mod.rs').read_text()
api_source = (ROOT / 'src/api/mod.rs').read_text()

schema_sqls = re.findall(r'r#"(CREATE TABLE IF NOT EXISTS .*?)"#', schema_source, re.S)
index_sqls = re.findall(r'"(CREATE INDEX IF NOT EXISTS [^"\n]+)"', schema_source)
assert len(schema_sqls) == 3, len(schema_sqls)
assert len(index_sqls) == 2, len(index_sqls)

conn = sqlite3.connect(':memory:')
conn.execute('PRAGMA foreign_keys=ON')
for stmt in [*schema_sqls, *index_sqls]:
    conn.execute(stmt)
assert [r[0] for r in conn.execute('SELECT name FROM sqlite_master WHERE type="table" AND name NOT LIKE "sqlite_%" ORDER BY name')] == [
    'players', 'round_results', 'rounds'
]

for name in ['희재93', '도윤95', '지호99', '인섭02', '성호97']:
    conn.execute('INSERT INTO players (name) VALUES (?)', (name,))
round_id = conn.execute('INSERT INTO rounds (date) VALUES (?)', ('2026-10-08',)).lastrowid
for rank, player_id in enumerate([3, 1, 4, 2], start=1):
    conn.execute('INSERT INTO round_results (round_id, player_id, rank) VALUES (?, ?, ?)', (round_id, player_id, rank))

round_sql_match = re.search(r'let rows = sqlx::query\(\s*r#"(SELECT r\.id, r\.date, r\.created_at, rr\.player_id, rr\.rank.*?)"#', repo_source, re.S)
assert round_sql_match, 'get_rounds_by_date query not found'
rows = conn.execute(round_sql_match.group(1), ('2026-10-08',)).fetchall()
assert [r[3] for r in rows] == [3, 1, 4, 2]

stats_sql_match = re.search(r'let mut sql = String::from\(\s*r#"(SELECT p\.id AS player_id.*?)"#', repo_source, re.S)
assert stats_sql_match, 'stats base query not found'
stats_sql = stats_sql_match.group(1) + ' WHERE r.date >= ? AND r.date <= ? ) fr ON p.id = fr.player_id GROUP BY p.id, p.name ORDER BY p.name ASC'
stats = conn.execute(stats_sql, ('2026-10-08', '2026-10-08')).fetchall()
assert len(stats) == 5
assert sorted(row[2] for row in stats) == [0, 1, 1, 1, 1]

# 정상 fixture 트랜잭션을 먼저 확정해야 실패 트랜잭션만 롤백됩니다.
conn.commit()
# FK가 하나라도 틀리면 라운드 및 결과 모두 롤백된다는 계약.
count_before = conn.execute('SELECT count(*) FROM rounds').fetchone()[0]
try:
    with conn:
        failed_id = conn.execute('INSERT INTO rounds (date) VALUES (?)', ('2026-10-08',)).lastrowid
        conn.execute('INSERT INTO round_results VALUES (?, ?, ?)', (failed_id, 99999, 1))
except sqlite3.IntegrityError:
    pass
else:
    raise AssertionError('foreign key check should have failed')
assert conn.execute('SELECT count(*) FROM rounds').fetchone()[0] == count_before

conn.execute('DELETE FROM rounds WHERE id = ?', (round_id,))
assert conn.execute('SELECT count(*) FROM round_results WHERE round_id = ?', (round_id,)).fetchone()[0] == 0

route_count = len(re.findall(r'\.route\("/api/', api_source))
assert route_count == 8, f'expected 8 route declarations (9 method/path pairs), got {route_count}'

print('PASS: SQLite schema 3 tables, 2 indexes')
print('PASS: round ranking order and 0-game player in filtered statistics')
print('PASS: FK failure rolls back inserts, DELETE cascades')
print('PASS: 8 Axum route declarations / 9 HTTP handlers')
print('NOTE: Rust cargo check/test and actual HTTP behavior were NOT run here')
