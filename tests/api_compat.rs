//! 사용자 API와 SQLite 스키마를 실제 Axum HTTP 요청으로 검증합니다.
//! 프로덕션 DB 대신 테스트별 독립된 in-memory SQLite를 사용합니다.

use axum::{
    body::Body,
    http::{Request, StatusCode},
    Router,
};
use http_body_util::BodyExt;
use mahjong_stat_back::{
    api::build_router,
    repository::Repository,
    schema,
    service::{today_kst, Service},
};
use serde_json::{json, Value};
use sqlx::{
    sqlite::{SqliteConnectOptions, SqlitePoolOptions},
    SqlitePool,
};
use tower::ServiceExt;

async fn setup_with_pool() -> (Router, SqlitePool) {
    // :memory:는 연결별로 달라지므로 pool 최대 연결수를 1로 고정.
    let pool = SqlitePoolOptions::new()
        .max_connections(1)
        .connect_with(
            SqliteConnectOptions::new()
                .in_memory(true)
                .foreign_keys(true),
        )
        .await
        .unwrap();
    schema::init_schema(&pool).await.unwrap();
    (
        build_router(Service::new(Repository::new(pool.clone()))),
        pool,
    )
}

async fn setup() -> Router {
    setup_with_pool().await.0
}

async fn request(app: &Router, method: &str, uri: &str, body: Value) -> (StatusCode, Value) {
    let request = Request::builder()
        .method(method)
        .uri(uri)
        .header("content-type", "application/json")
        .body(Body::from(body.to_string()))
        .unwrap();
    let response = app.clone().oneshot(request).await.unwrap();
    let status = response.status();
    let bytes = response.into_body().collect().await.unwrap().to_bytes();
    (status, serde_json::from_slice(&bytes).unwrap())
}

#[tokio::test]
async fn empty_legacy_arrays_preserve_null_vs_empty_list() {
    let app = setup().await;
    let (_, players) = request(&app, "GET", "/api/players", Value::Null).await;
    assert_eq!(players, Value::Null); // Go: nil slice -> null
    let (_, rounds) = request(
        &app,
        "GET",
        "/api/rounds/by-date?date=2026-10-08",
        Value::Null,
    )
    .await;
    assert_eq!(rounds, Value::Null);
    let (_, stats) = request(&app, "GET", "/api/stats/all", Value::Null).await;
    assert_eq!(stats, json!([])); // Go: make([]...) -> []
}

#[tokio::test]
async fn creates_rounds_stats_and_deletes_with_same_json_shapes() {
    let app = setup().await;
    let mut players = Vec::new();
    for name in ["희재93", "도윤95", "지호99", "인섭02"] {
        let (status, player) = request(&app, "POST", "/api/players", json!({"name":name})).await;
        assert_eq!(status, StatusCode::OK);
        assert!(player["created_at"].as_str().unwrap().ends_with('Z'));
        players.push(player["id"].as_i64().unwrap());
    }
    let date = today_kst();
    let (status, created) = request(
        &app,
        "POST",
        "/api/rounds",
        json!({"date":date, "ranking":players}),
    )
    .await;
    assert_eq!(status, StatusCode::OK);
    let round_id = created["round_id"].as_i64().unwrap();
    let (_, rounds) = request(
        &app,
        "GET",
        &format!("/api/rounds/by-date?date={date}"),
        Value::Null,
    )
    .await;
    assert_eq!(rounds[0]["ranking"], json!(players));

    let (_, stats) = request(&app, "GET", "/api/stats/all", Value::Null).await;
    assert_eq!(stats.as_array().unwrap().len(), 4);
    let (_, detail) = request(
        &app,
        "GET",
        &format!("/api/stats/players/{}/history", players[0]),
        Value::Null,
    )
    .await;
    assert_eq!(detail["summary"]["games"], 1);
    assert_eq!(detail["summary"]["avg_rank"], 1.0);
    assert_eq!(detail["rank_counts"]["first"], 1);
    assert_eq!(detail["recent_rounds"][0]["round_id"], round_id);

    let (status, deleted) = request(
        &app,
        "DELETE",
        &format!("/api/rounds/{round_id}"),
        Value::Null,
    )
    .await;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(deleted["ranking"], json!(players));
    let (_, rounds) = request(
        &app,
        "GET",
        &format!("/api/rounds/by-date?date={date}"),
        Value::Null,
    )
    .await;
    assert_eq!(rounds, Value::Null);
}

#[tokio::test]
async fn invalid_dates_unknown_players_and_invalid_payloads() {
    let app = setup().await;
    let (status, body) = request(&app, "GET", "/api/rounds/by-date", Value::Null).await;
    assert_eq!(status, StatusCode::BAD_REQUEST);
    assert_eq!(body, json!({"error":"date query is required"}));

    let (status, body) = request(
        &app,
        "GET",
        "/api/stats/all?start_date=2026-02-29",
        Value::Null,
    )
    .await;
    assert_eq!(status, StatusCode::BAD_REQUEST);
    assert_eq!(
        body,
        json!({"error":"invalid date format; expected YYYY-MM-DD"})
    );

    let (status, _) = request(&app, "GET", "/api/stats/players/12345/history", Value::Null).await;
    assert_eq!(status, StatusCode::NOT_FOUND);
    let (status, body) = request(&app, "POST", "/api/players", json!({"wrong":"field"})).await;
    assert_eq!(status, StatusCode::BAD_REQUEST);
    assert_eq!(body, json!({"error":"invalid payload"}));
    let (status, body) = request(
        &app,
        "POST",
        "/api/rounds",
        json!({"date":"2000-01-01","ranking":[1,2,3,4]}),
    )
    .await;
    assert_eq!(status, StatusCode::BAD_REQUEST);
    assert_eq!(
        body,
        json!({"error":"라운드는 오늘 날짜에만 저장할 수 있습니다"})
    );
    let (status, _) = request(&app, "DELETE", "/api/rounds/9999", Value::Null).await;
    assert_eq!(status, StatusCode::NOT_FOUND);
    let (status, body) = request(&app, "GET", "/api/no-such-endpoint", Value::Null).await;
    assert_eq!(status, StatusCode::NOT_FOUND);
    assert_eq!(body, json!({"error":"not found"}));
    let (status, body) = request(&app, "POST", "/api/server-date", Value::Null).await;
    assert_eq!(status, StatusCode::NOT_FOUND);
    assert_eq!(body, json!({"error":"not found"}));
}

#[tokio::test]
async fn foreign_key_failure_does_not_leave_partial_round() {
    let app = setup().await;
    let today = today_kst();
    let (status, _) = request(
        &app,
        "POST",
        "/api/rounds",
        json!({"date":today,"ranking":[100,101,102,103]}),
    )
    .await;
    assert_eq!(status, StatusCode::BAD_REQUEST);
    let (_, rounds) = request(
        &app,
        "GET",
        &format!("/api/rounds/by-date?date={today}"),
        Value::Null,
    )
    .await;
    assert_eq!(rounds, Value::Null);
}

#[tokio::test]
async fn period_filter_keeps_inactive_players_and_limits_recent_rounds() {
    let (app, pool) = setup_with_pool().await;
    let active_id = sqlx::query("INSERT INTO players (name) VALUES (?)")
        .bind("희재93")
        .execute(&pool)
        .await
        .unwrap()
        .last_insert_rowid();
    let inactive_id = sqlx::query("INSERT INTO players (name) VALUES (?)")
        .bind("도윤95")
        .execute(&pool)
        .await
        .unwrap()
        .last_insert_rowid();

    // 기간 바깥에만 기록이 있는 플레이어도 통계 목록에서 사라지면 안 됩니다.
    let old_id = sqlx::query(
        "INSERT INTO rounds (date, created_at) VALUES ('2026-08-01', '2026-08-01 20:00:00')",
    )
    .execute(&pool)
    .await
    .unwrap()
    .last_insert_rowid();
    sqlx::query("INSERT INTO round_results (round_id, player_id, rank) VALUES (?, ?, 4)")
        .bind(old_id)
        .bind(inactive_id)
        .execute(&pool)
        .await
        .unwrap();

    // 동일 시각이라도 높은 round_id가 최근 기록에서 먼저 나오는지 검사합니다.
    let mut last_id = 0;
    for index in 0..55 {
        last_id = sqlx::query(
            "INSERT INTO rounds (date, created_at) VALUES ('2026-09-18', '2026-09-18 18:00:00')",
        )
        .execute(&pool)
        .await
        .unwrap()
        .last_insert_rowid();
        sqlx::query("INSERT INTO round_results (round_id, player_id, rank) VALUES (?, ?, ?)")
            .bind(last_id)
            .bind(active_id)
            .bind(index % 4 + 1)
            .execute(&pool)
            .await
            .unwrap();
    }

    let (status, stats) = request(
        &app,
        "GET",
        "/api/stats/all?start_date=2026-09-01&end_date=2026-09-30",
        Value::Null,
    )
    .await;
    assert_eq!(status, StatusCode::OK);
    let stats_rows = stats.as_array().unwrap();
    assert_eq!(stats_rows.len(), 2);
    assert_eq!(
        stats_rows
            .iter()
            .find(|s| s["player_id"] == active_id)
            .unwrap()["games"],
        55
    );
    assert_eq!(
        stats_rows
            .iter()
            .find(|s| s["player_id"] == inactive_id)
            .unwrap()["games"],
        0
    );

    let (_, history) = request(
        &app,
        "GET",
        &format!("/api/stats/players/{active_id}/history"),
        Value::Null,
    )
    .await;
    assert_eq!(history["summary"]["games"], 55);
    assert_eq!(history["daily"].as_array().unwrap().len(), 1);
    assert_eq!(history["recent_rounds"].as_array().unwrap().len(), 50);
    assert_eq!(history["recent_rounds"][0]["round_id"], last_id);
    assert_eq!(history["recent_rounds"][0]["rank"], 3);
}

#[tokio::test]
async fn deleting_round_cascades_to_round_results() {
    let (app, pool) = setup_with_pool().await;
    // 사용자가 정상적인 API 흐름으로 만든 라운드가 삭제 시 자식 테이블에서 제거됩니다.
    let mut player_ids = Vec::new();
    for name in ["희재93", "도윤95", "지호99", "인섭02"] {
        let (_, player) = request(&app, "POST", "/api/players", json!({"name": name})).await;
        player_ids.push(player["id"].as_i64().unwrap());
    }
    let (_, created) = request(
        &app,
        "POST",
        "/api/rounds",
        json!({"date": today_kst(), "ranking": player_ids}),
    )
    .await;
    let id = created["round_id"].as_i64().unwrap();
    let (status, _) = request(&app, "DELETE", &format!("/api/rounds/{id}"), Value::Null).await;
    assert_eq!(status, StatusCode::OK);
    let count: i64 = sqlx::query_scalar("SELECT count(*) FROM round_results WHERE round_id = ?")
        .bind(id)
        .fetch_one(&pool)
        .await
        .unwrap();
    assert_eq!(count, 0);
}

#[tokio::test]
async fn typed_query_structs_keep_legacy_missing_and_empty_semantics() {
    let app = setup().await;

    // `?date=`와 날짜 미지정 모두 동일한 JSON 오류를 내려줘야 합니다.
    for url in ["/api/players/by-date", "/api/players/by-date?date="] {
        let (status, body) = request(&app, "GET", url, Value::Null).await;
        assert_eq!(status, StatusCode::BAD_REQUEST);
        assert_eq!(body, json!({"error":"date query is required"}));
    }

    // 미지정 날짜 범위와 명시적으로 비운 날짜 범위는 둘 다 무제한 검색입니다.
    let (status, stats) = request(
        &app,
        "GET",
        "/api/stats/all?start_date=&end_date=",
        Value::Null,
    )
    .await;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(stats, json!([]));

    let (status, body) = request(
        &app,
        "GET",
        "/api/stats/all?start_date=2026-10-08&end_date=2026-10-07",
        Value::Null,
    )
    .await;
    assert_eq!(status, StatusCode::BAD_REQUEST);
    assert_eq!(
        body,
        json!({"error":"start_date must not be after end_date"})
    );
}

#[tokio::test]
async fn invalid_db_timestamp_returns_error_instead_of_fabricated_date() {
    let (app, pool) = setup_with_pool().await;

    // 일반적인 API로는 만들어질 수 없지만, 오래된 DB의 수동 변경·손상을 가정합니다.
    sqlx::query("INSERT INTO players (name, created_at) VALUES (?, ?)")
        .bind("희재93")
        .bind("this-is-not-a-date")
        .execute(&pool)
        .await
        .unwrap();

    let (status, players) = request(&app, "GET", "/api/players", Value::Null).await;
    assert_eq!(status, StatusCode::INTERNAL_SERVER_ERROR);
    assert!(players["error"].as_str().is_some());
    assert!(!players.to_string().contains("0001-01-01"));
}

#[tokio::test]
async fn timestamp_with_timezone_remains_unchanged_in_player_json() {
    let (app, pool) = setup_with_pool().await;
    sqlx::query("INSERT INTO players (name, created_at) VALUES (?, ?)")
        .bind("희재93")
        .bind("2026-10-08T15:30:00+09:00")
        .execute(&pool)
        .await
        .unwrap();

    let (status, players) = request(&app, "GET", "/api/players", Value::Null).await;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(players[0]["created_at"], "2026-10-08T15:30:00+09:00");
}

#[tokio::test]
async fn date_lookup_keeps_original_non_validating_behavior() {
    let app = setup().await;
    // 기존 Go API에서 날짜별 조회는 빈 문자열만 거절하며 날짜 형식 자체는 검증하지 않습니다.
    // 타입을 도입해도 이 동작을 변경하지 않아야 합니다.
    let (status, rounds) = request(
        &app,
        "GET",
        "/api/rounds/by-date?date=not-a-date",
        Value::Null,
    )
    .await;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(rounds, Value::Null);
}
