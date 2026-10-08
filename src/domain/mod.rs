//! API 요청·응답 모델 및 검증된 내부 조회 조건.
//!
//! 외부 JSON의 필드명과 `null`/`[]` 처리 방식은 기존 Go 서버와 같습니다.
//! `date` 및 `created_at` 응답은 과거 데이터·프론트엔드 호환을 위해 문자열을 유지합니다.
//! 반면, 통계 검색에 사용되는 날짜 범위는 `NaiveDate`로 검증하여 전달합니다.

use chrono::NaiveDate;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Player {
    pub id: i64,
    pub name: String,
    /// DB의 UTC 시각을 RFC3339로 변환합니다. JSON 필드 형태는 기존과 동일합니다.
    pub created_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Round {
    pub id: i64,
    pub date: String,
    pub created_at: Option<String>,
    /// 순위 순서의 플레이어 ID (1위부터 4위까지).
    pub ranking: Vec<i64>,
}

#[derive(Debug, Deserialize)]
pub struct CreatePlayerRequest {
    pub name: String,
}

#[derive(Debug, Deserialize)]
pub struct CreateRoundRequest {
    /// 누락/비어 있는 날짜에 대한 기존 오류 메시지를 유지하므로 요청은 문자열로 받습니다.
    pub date: String,
    pub ranking: Vec<i64>,
}

/// `GET /api/.../by-date` 쿼리. 미지정 시 `None`, 빈 값도 동일하게 오류 처리합니다.
#[derive(Debug, Default, Deserialize)]
pub struct DateQuery {
    pub date: Option<String>,
}

/// 통계 API의 원본 쿼리. 유효한 날짜인지는 서비스 계층에서 검사합니다.
#[derive(Debug, Default, Deserialize)]
pub struct DateRangeQuery {
    pub start_date: Option<String>,
    pub end_date: Option<String>,
}

/// 타입 검사가 완료된 통계 검색 조건. `None`은 해당 경계가 없음을 뜻합니다.
///
/// SQL에 사용할 때에만 ISO 날짜 문자열로 직렬화합니다. 이 구조체를 서비스→DB 간에
/// 전달함으로써 잘못된 날짜가 DB 쿼리에 그대로 들어가는 것을 방지합니다.
#[derive(Debug, Clone, Copy, Default)]
pub struct DateRange {
    pub start: Option<NaiveDate>,
    pub end: Option<NaiveDate>,
}

#[derive(Debug, Serialize)]
pub struct ServerDateResponse {
    pub today: String,
}

#[derive(Debug, Serialize)]
pub struct CreateRoundResponse {
    pub round_id: i64,
}

#[derive(Debug, Serialize)]
pub struct ErrorResponse {
    pub error: String,
}

#[derive(Debug, Serialize)]
pub struct PlayerTotalStats {
    pub player_id: i64,
    pub name: String,
    pub games: i64,
    pub first: i64,
    pub second: i64,
    pub third: i64,
    pub fourth: i64,
    pub first_rate: f64,
    pub top2_rate: f64,
    pub fourth_rate: f64,
    pub avg_rank: f64,
}

#[derive(Debug, Default, Serialize)]
pub struct PlayerStatsSummary {
    pub games: i64,
    pub avg_rank: f64,
    pub first_rate: f64,
    pub top2_rate: f64,
    pub fourth_rate: f64,
}

#[derive(Debug, Default, Serialize)]
pub struct RankCounts {
    pub first: i64,
    pub second: i64,
    pub third: i64,
    pub fourth: i64,
}

#[derive(Debug, Serialize)]
pub struct DailyPlayerStats {
    pub date: String,
    pub games: i64,
    pub first: i64,
    pub second: i64,
    pub third: i64,
    pub fourth: i64,
    pub avg_rank: f64,
}

/// DB 조회 후 서비스에서 집계하는 내부 데이터.
#[derive(Debug)]
pub struct PlayerRoundResult {
    pub round_id: i64,
    pub date: String,
    pub rank: i64,
    pub created_at: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct PlayerRecentRound {
    pub round_id: i64,
    pub date: String,
    pub rank: i64,
    pub created_at: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct PlayerStatsHistory {
    pub player: Player,
    pub summary: PlayerStatsSummary,
    pub rank_counts: RankCounts,
    /// 날짜 오름차순. 비어 있으면 JSON `[]`.
    pub daily: Vec<DailyPlayerStats>,
    /// 최신순 최대 50개. 비어 있으면 JSON `[]`.
    pub recent_rounds: Vec<PlayerRecentRound>,
}
