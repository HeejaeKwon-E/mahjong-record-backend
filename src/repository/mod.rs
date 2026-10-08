//! SQLite 데이터 접근 계층. SQL의 정렬/필터/트랜잭션은 Go 원본을 기준으로 합니다.

use chrono::{DateTime, NaiveDateTime, SecondsFormat};
use sqlx::{Row, SqlitePool};

use crate::domain::{DateRange, Player, PlayerRoundResult, PlayerTotalStats, Round};

/// SQLite 행을 SQLx가 컴파일된 Rust 타입으로 변환합니다.
/// `created_at`은 SQLite의 DATETIME(TEXT)이며, 도메인으로 옮기기 전에 검증합니다.
#[derive(Debug, sqlx::FromRow)]
struct DbPlayer {
    id: i64,
    name: String,
    created_at: String,
}

impl DbPlayer {
    fn into_domain(self) -> Result<Player, sqlx::Error> {
        Ok(Player {
            id: self.id,
            name: self.name,
            created_at: Some(parse_sqlite_time(&self.created_at)?),
        })
    }
}

/// SQL 결과에 LEFT JOIN의 NULL 합계가 포함될 수 있음을 타입에 표현합니다.
#[derive(Debug, sqlx::FromRow)]
struct DbPlayerCounts {
    player_id: i64,
    name: String,
    games: i64,
    first: Option<i64>,
    second: Option<i64>,
    third: Option<i64>,
    fourth: Option<i64>,
}

/// 최근 대국 집계용 행. SELECT에서 `id AS round_id` 별칭을 사용합니다.
#[derive(Debug, sqlx::FromRow)]
struct DbPlayerRound {
    round_id: i64,
    date: String,
    rank: i64,
    created_at: String,
}

#[derive(Clone)]
pub struct Repository {
    pool: SqlitePool,
}

impl Repository {
    pub fn new(pool: SqlitePool) -> Self {
        Self { pool }
    }

    pub async fn get_all_players(&self) -> Result<Option<Vec<Player>>, sqlx::Error> {
        let rows =
            sqlx::query_as::<_, DbPlayer>("SELECT id, name, created_at FROM players ORDER BY id")
                .fetch_all(&self.pool)
                .await?;
        let players: Vec<_> = rows
            .into_iter()
            .map(DbPlayer::into_domain)
            .collect::<Result<_, _>>()?;
        // Go의 nil slice가 JSON null로 직렬화되는 기존 응답과 호환.
        Ok((!players.is_empty()).then_some(players))
    }

    pub async fn create_player(&self, name: &str) -> Result<Player, sqlx::Error> {
        let result = sqlx::query("INSERT INTO players (name) VALUES (?)")
            .bind(name)
            .execute(&self.pool)
            .await?;
        let row =
            sqlx::query_as::<_, DbPlayer>("SELECT id, name, created_at FROM players WHERE id = ?")
                .bind(result.last_insert_rowid())
                .fetch_one(&self.pool)
                .await?;
        row.into_domain()
    }

    pub async fn get_player_by_id(&self, id: i64) -> Result<Option<Player>, sqlx::Error> {
        let row =
            sqlx::query_as::<_, DbPlayer>("SELECT id, name, created_at FROM players WHERE id = ?")
                .bind(id)
                .fetch_optional(&self.pool)
                .await?;
        row.map(DbPlayer::into_domain).transpose()
    }

    pub async fn create_round(&self, date: &str, ranking: &[i64]) -> Result<i64, sqlx::Error> {
        // 어느 순위라도 FK/PK 제약에 실패하면 rounds 테이블의 INSERT까지 롤백됩니다.
        let mut tx = self.pool.begin().await?;
        let result = sqlx::query("INSERT INTO rounds (date) VALUES (?)")
            .bind(date)
            .execute(&mut *tx)
            .await?;
        let id = result.last_insert_rowid();
        for (index, player_id) in ranking.iter().enumerate() {
            sqlx::query("INSERT INTO round_results (round_id, player_id, rank) VALUES (?, ?, ?)")
                .bind(id)
                .bind(player_id)
                .bind(index as i64 + 1)
                .execute(&mut *tx)
                .await?;
        }
        tx.commit().await?;
        Ok(id)
    }

    pub async fn get_rounds_by_date(&self, date: &str) -> Result<Option<Vec<Round>>, sqlx::Error> {
        // Go에서 최적화해 둔 단일 JOIN을 유지(N+1 쿼리 방지).
        let rows = sqlx::query(
            r#"SELECT r.id, r.date, r.created_at, rr.player_id, rr.rank
               FROM rounds r
               LEFT JOIN round_results rr ON rr.round_id = r.id
               WHERE r.date = ?
               ORDER BY r.created_at DESC, r.id DESC, rr.rank ASC"#,
        )
        .bind(date)
        .fetch_all(&self.pool)
        .await?;

        let mut rounds: Vec<Round> = Vec::new();
        for row in rows {
            let id: i64 = row.try_get("id")?;
            if rounds.last().map(|r| r.id) != Some(id) {
                rounds.push(Round {
                    id,
                    date: row.try_get("date")?,
                    created_at: Some(parse_sqlite_time(&row.try_get::<String, _>("created_at")?)?),
                    ranking: Vec::with_capacity(4),
                });
            }
            // LEFT JOIN에서 결과 없는 라운드 역시 ranking=[]로 반환합니다.
            let player_id: Option<i64> = row.try_get("player_id")?;
            let rank: Option<i64> = row.try_get("rank")?;
            if let (Some(pid), Some(_)) = (player_id, rank) {
                if let Some(current) = rounds.last_mut() {
                    current.ranking.push(pid);
                }
            }
        }
        Ok((!rounds.is_empty()).then_some(rounds))
    }

    pub async fn get_players_by_date(
        &self,
        date: &str,
    ) -> Result<Option<Vec<Player>>, sqlx::Error> {
        let rows = sqlx::query_as::<_, DbPlayer>(
            r#"SELECT DISTINCT p.id, p.name, p.created_at
               FROM players p
               JOIN round_results rr ON p.id = rr.player_id
               JOIN rounds r ON rr.round_id = r.id
               WHERE r.date = ?
               ORDER BY p.id"#,
        )
        .bind(date)
        .fetch_all(&self.pool)
        .await?;
        let players: Vec<_> = rows
            .into_iter()
            .map(DbPlayer::into_domain)
            .collect::<Result<_, _>>()?;
        Ok((!players.is_empty()).then_some(players))
    }

    pub async fn delete_round(&self, id: i64) -> Result<Option<Round>, sqlx::Error> {
        // 기존에는 조회 후 삭제했지만, 삭제 결과 일관성을 위해 하나의 트랜잭션에서 처리.
        let mut tx = self.pool.begin().await?;
        let row = sqlx::query("SELECT id, date, created_at FROM rounds WHERE id = ?")
            .bind(id)
            .fetch_optional(&mut *tx)
            .await?;
        let Some(row) = row else { return Ok(None) };
        let ranking_rows =
            sqlx::query("SELECT player_id FROM round_results WHERE round_id = ? ORDER BY rank ASC")
                .bind(id)
                .fetch_all(&mut *tx)
                .await?;
        // `get()`은 타입 불일치 시 패닉할 수 있으므로 실패를 DB 오류로 전파합니다.
        let ranking: Vec<i64> = ranking_rows
            .iter()
            .map(|row| row.try_get("player_id"))
            .collect::<Result<_, sqlx::Error>>()?;
        let round = Round {
            id: row.try_get("id")?,
            date: row.try_get("date")?,
            created_at: Some(parse_sqlite_time(&row.try_get::<String, _>("created_at")?)?),
            ranking,
        };
        let deleted = sqlx::query("DELETE FROM rounds WHERE id = ?")
            .bind(id)
            .execute(&mut *tx)
            .await?;
        if deleted.rows_affected() == 0 {
            return Ok(None);
        }
        tx.commit().await?;
        Ok(Some(round))
    }

    pub async fn get_all_player_total_stats(
        &self,
        range: &DateRange,
    ) -> Result<Vec<PlayerTotalStats>, sqlx::Error> {
        // WHERE를 동적으로 조합해 기존 Go 쿼리의 날짜 인덱스 활용 형태를 유지합니다.
        // SQL 파라미터 값은 절대로 문자열 연결하지 않고 bind()합니다.
        let mut sql = String::from(
            r#"SELECT p.id AS player_id, p.name AS name,
                   COUNT(fr.player_id) AS games,
                   SUM(CASE WHEN fr.rank = 1 THEN 1 ELSE 0 END) AS first,
                   SUM(CASE WHEN fr.rank = 2 THEN 1 ELSE 0 END) AS second,
                   SUM(CASE WHEN fr.rank = 3 THEN 1 ELSE 0 END) AS third,
                   SUM(CASE WHEN fr.rank = 4 THEN 1 ELSE 0 END) AS fourth
               FROM players p
               LEFT JOIN (
                   SELECT rr.player_id, rr.rank
                   FROM rounds r
                   JOIN round_results rr ON rr.round_id = r.id"#,
        );
        // 검증된 NaiveDate를 SQL 바인딩 경계에서 기존 TEXT 날짜 형식으로 변환합니다.
        let start_date = range.start.map(|date| date.format("%Y-%m-%d").to_string());
        let end_date = range.end.map(|date| date.format("%Y-%m-%d").to_string());
        let mut conditions = Vec::new();
        if start_date.is_some() {
            conditions.push("r.date >= ?");
        }
        if end_date.is_some() {
            conditions.push("r.date <= ?");
        }
        if !conditions.is_empty() {
            sql.push_str(" WHERE ");
            sql.push_str(&conditions.join(" AND "));
        }
        sql.push_str(" ) fr ON p.id = fr.player_id GROUP BY p.id, p.name ORDER BY p.name ASC");

        let mut query = sqlx::query_as::<_, DbPlayerCounts>(&sql);
        if let Some(ref date) = start_date {
            query = query.bind(date);
        }
        if let Some(ref date) = end_date {
            query = query.bind(date);
        }
        let rows = query.fetch_all(&self.pool).await?;

        let mut stats = Vec::with_capacity(rows.len());
        for row in rows {
            let games = row.games;
            let first = row.first.unwrap_or(0);
            let second = row.second.unwrap_or(0);
            let third = row.third.unwrap_or(0);
            let fourth = row.fourth.unwrap_or(0);
            let (first_rate, top2_rate, fourth_rate, avg_rank) = if games > 0 {
                let g = games as f64;
                (
                    first as f64 / g,
                    (first + second) as f64 / g,
                    fourth as f64 / g,
                    (first + second * 2 + third * 3 + fourth * 4) as f64 / g,
                )
            } else {
                (0.0, 0.0, 0.0, 0.0)
            };
            stats.push(PlayerTotalStats {
                player_id: row.player_id,
                name: row.name,
                games,
                first,
                second,
                third,
                fourth,
                first_rate,
                top2_rate,
                fourth_rate,
                avg_rank,
            });
        }
        // Go에서 []로 초기화하므로 빈 통계 역시 [] 반환.
        Ok(stats)
    }

    pub async fn get_player_round_results(
        &self,
        player_id: i64,
        range: &DateRange,
    ) -> Result<Vec<PlayerRoundResult>, sqlx::Error> {
        let mut sql = String::from(
            r#"SELECT r.id AS round_id, r.date, rr.rank, r.created_at
               FROM round_results rr
               JOIN rounds r ON r.id = rr.round_id
               WHERE rr.player_id = ?"#,
        );
        let start_date = range.start.map(|date| date.format("%Y-%m-%d").to_string());
        let end_date = range.end.map(|date| date.format("%Y-%m-%d").to_string());
        if start_date.is_some() {
            sql.push_str(" AND r.date >= ?");
        }
        if end_date.is_some() {
            sql.push_str(" AND r.date <= ?");
        }
        sql.push_str(" ORDER BY r.date ASC, r.created_at ASC, r.id ASC");
        let mut query = sqlx::query_as::<_, DbPlayerRound>(&sql).bind(player_id);
        if let Some(ref date) = start_date {
            query = query.bind(date);
        }
        if let Some(ref date) = end_date {
            query = query.bind(date);
        }
        let rows = query.fetch_all(&self.pool).await?;

        rows.into_iter()
            .map(|row| {
                Ok(PlayerRoundResult {
                    round_id: row.round_id,
                    date: row.date,
                    rank: row.rank,
                    created_at: Some(parse_sqlite_time(&row.created_at)?),
                })
            })
            .collect()
    }
}

/// SQLite UTC DATETIME(TEXT)를 기존 JSON 계약의 RFC3339 문자열로 변환합니다.
///
/// DB에 명시적 타임존 오프셋이 기록됐다면 그대로 보존합니다. 과거 구현에서
/// 잘못된 값을 `0001-01-01`로 숨기던 문제를 제거하고, 파싱 실패를 SQLx 오류로
/// 전파합니다. API 계층에서 이 오류는 HTTP 500으로 처리됩니다.
fn parse_sqlite_time(value: &str) -> Result<String, sqlx::Error> {
    let parsed = if let Ok(value) = DateTime::parse_from_rfc3339(value) {
        value
    } else {
        let naive = NaiveDateTime::parse_from_str(value, "%Y-%m-%d %H:%M:%S")
            .map_err(|error| sqlx::Error::Decode(Box::new(error)))?;
        naive.and_utc().fixed_offset()
    };
    Ok(parsed.to_rfc3339_opts(SecondsFormat::AutoSi, true))
}

#[cfg(test)]
mod tests {
    use super::parse_sqlite_time;

    #[test]
    fn sqlite_time_is_utc_rfc3339() {
        assert_eq!(
            parse_sqlite_time("2026-09-18 20:30:00").unwrap(),
            "2026-09-18T20:30:00Z"
        );
        // 기존 Go 응답 규칙: 오프셋이 이미 지정됐다면 유지.
        assert_eq!(
            parse_sqlite_time("2026-09-18T20:30:00+09:00").unwrap(),
            "2026-09-18T20:30:00+09:00"
        );
    }

    #[test]
    fn malformed_sqlite_time_is_an_error_not_a_fake_date() {
        assert!(parse_sqlite_time("not-a-timestamp").is_err());
    }
}
