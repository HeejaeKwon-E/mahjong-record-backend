//! Go internal/schema/schema.go와 동일한 테이블, 인덱스 정의.
use sqlx::SqlitePool;

pub async fn init_schema(pool: &SqlitePool) -> Result<(), sqlx::Error> {
    // sqlx::query()는 한 번에 복수 SQL을 실행하지 않으므로 문장별로 실행합니다.
    // IF NOT EXISTS: 이미 운영 중인 Go DB에 연결해도 데이터는 변경하지 않습니다.
    for statement in [
        r#"CREATE TABLE IF NOT EXISTS players (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL UNIQUE,
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        )"#,
        r#"CREATE TABLE IF NOT EXISTS rounds (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            date TEXT NOT NULL,
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        )"#,
        r#"CREATE TABLE IF NOT EXISTS round_results (
            round_id INTEGER NOT NULL,
            player_id INTEGER NOT NULL,
            rank INTEGER NOT NULL,
            PRIMARY KEY (round_id, player_id),
            FOREIGN KEY (round_id) REFERENCES rounds(id) ON DELETE CASCADE,
            FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
        )"#,
        "CREATE INDEX IF NOT EXISTS idx_round_results_player_id ON round_results(player_id, round_id)",
        "CREATE INDEX IF NOT EXISTS idx_rounds_date_created_at ON rounds(date, created_at, id)",
    ] {
        sqlx::query(statement).execute(pool).await?;
    }
    Ok(())
}
