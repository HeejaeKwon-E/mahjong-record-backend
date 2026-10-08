//! 기존 Gin의 API 계약을 유지하는 Axum 라우터 및 HTTP 핸들러.
//!
//! Query 파라미터와 응답은 명확한 Serde 구조체로 표현합니다.
//! 서비스/DB 오류는 이전 API의 상태 코드와 오류 문구 규칙에 맞춰 변환합니다.

use axum::{
    extract::{rejection::JsonRejection, Path, Query, State},
    http::{header, StatusCode, Uri},
    response::{IntoResponse, Response},
    routing::{delete, get, post},
    Json, Router,
};
use tower_http::services::ServeDir;

use crate::{
    domain::{
        CreatePlayerRequest, CreateRoundRequest, CreateRoundResponse, DateQuery, DateRangeQuery,
        ErrorResponse, Player, PlayerStatsHistory, PlayerTotalStats, Round, ServerDateResponse,
    },
    service::{today_kst, Service, ServiceError},
};

#[derive(Clone)]
struct AppState {
    service: Service,
}

/// 프론트엔드가 사용 중인 9개 API 경로/HTTP 메서드와 SPA 정적 파일 서빙을 유지합니다.
pub fn build_router(service: Service) -> Router {
    Router::new()
        .route("/api/server-date", get(get_server_date))
        .route("/api/players", get(get_players).post(create_player))
        .route("/api/players/by-date", get(get_players_by_date))
        .route("/api/rounds", post(create_round))
        .route("/api/rounds/by-date", get(get_rounds_by_date))
        .route("/api/rounds/{id}", delete(delete_round))
        .route("/api/stats/all", get(get_all_player_stats))
        .route(
            "/api/stats/players/{player_id}/history",
            get(get_player_stats_history),
        )
        .nest_service("/assets", ServeDir::new("web/dist/assets"))
        .fallback(spa_fallback)
        // 이전 서버는 /api/... 에 미등록 메서드로 접근해도 JSON 404를 내려줍니다.
        .method_not_allowed_fallback(spa_fallback)
        .with_state(AppState { service })
}

/// 외부 에러 포맷은 {"error":"..."}로 고정합니다.
/// 서비스 오류의 HTTP 매핑을 각 핸들러에서 분명히 구분하기 위한 타입입니다.
struct ApiError {
    status: StatusCode,
    message: String,
}

impl ApiError {
    fn new(status: StatusCode, message: impl Into<String>) -> Self {
        Self {
            status,
            message: message.into(),
        }
    }

    fn bad_request(message: impl Into<String>) -> Self {
        Self::new(StatusCode::BAD_REQUEST, message)
    }

    fn not_found(message: &'static str) -> Self {
        Self::new(StatusCode::NOT_FOUND, message)
    }

    fn internal(context: &'static str, error: impl std::fmt::Display) -> Self {
        tracing::error!(%error, context, "database request failed");
        Self::new(StatusCode::INTERNAL_SERVER_ERROR, context)
    }
}

impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        (
            self.status,
            Json(ErrorResponse {
                error: self.message,
            }),
        )
            .into_response()
    }
}

/// Axum의 기본 JSON 거절 메시지 대신 기존 API의 `invalid payload`를 사용합니다.
fn payload<T>(incoming: Result<Json<T>, JsonRejection>) -> Result<T, ApiError> {
    incoming
        .map(|json| json.0)
        .map_err(|_| ApiError::bad_request("invalid payload"))
}

fn required_date(query: &DateQuery) -> Result<&str, ApiError> {
    query
        .date
        .as_deref()
        .filter(|date| !date.is_empty())
        .ok_or_else(|| ApiError::bad_request("date query is required"))
}

fn date_range(query: &DateRangeQuery) -> (&str, &str) {
    (
        query.start_date.as_deref().unwrap_or(""),
        query.end_date.as_deref().unwrap_or(""),
    )
}

fn date_query_error(error: ServiceError) -> ApiError {
    // 기존 Go 날짜별 조회 엔드포인트는 DB 조회 실패까지 HTTP 400으로 반환합니다.
    ApiError::bad_request(error.to_string())
}

async fn get_server_date() -> Json<ServerDateResponse> {
    Json(ServerDateResponse { today: today_kst() })
}

async fn get_players(State(state): State<AppState>) -> Result<Json<Option<Vec<Player>>>, ApiError> {
    let players = state.service.get_players().await.map_err(|error| {
        tracing::error!(%error, "failed to load players");
        ApiError::new(StatusCode::INTERNAL_SERVER_ERROR, error.to_string())
    })?;
    // Go의 nil slice -> JSON null 규칙을 Repository에서 그대로 유지합니다.
    Ok(Json(players))
}

async fn create_player(
    State(state): State<AppState>,
    incoming: Result<Json<CreatePlayerRequest>, JsonRejection>,
) -> Result<Json<Player>, ApiError> {
    let request = payload(incoming)?;
    // Go에서는 UNIQUE 제약 충돌 등의 등록 실패도 400으로 반환했습니다.
    let player = state
        .service
        .add_player(&request.name)
        .await
        .map_err(|error| ApiError::bad_request(error.to_string()))?;
    Ok(Json(player))
}

async fn create_round(
    State(state): State<AppState>,
    incoming: Result<Json<CreateRoundRequest>, JsonRejection>,
) -> Result<Json<CreateRoundResponse>, ApiError> {
    let request = payload(incoming)?;
    let round_id = state
        .service
        .add_round(&request.date, &request.ranking)
        .await
        .map_err(|error| ApiError::bad_request(error.to_string()))?;
    Ok(Json(CreateRoundResponse { round_id }))
}

async fn get_rounds_by_date(
    State(state): State<AppState>,
    Query(query): Query<DateQuery>,
) -> Result<Json<Option<Vec<Round>>>, ApiError> {
    let date = required_date(&query)?;
    let rounds = state
        .service
        .get_rounds_by_date(date)
        .await
        .map_err(date_query_error)?;
    Ok(Json(rounds))
}

async fn get_players_by_date(
    State(state): State<AppState>,
    Query(query): Query<DateQuery>,
) -> Result<Json<Option<Vec<Player>>>, ApiError> {
    let date = required_date(&query)?;
    let players = state
        .service
        .get_players_by_date(date)
        .await
        .map_err(date_query_error)?;
    Ok(Json(players))
}

async fn delete_round(
    State(state): State<AppState>,
    Path(raw_id): Path<String>,
) -> Result<Json<Round>, ApiError> {
    let id = raw_id
        .parse::<i64>()
        .ok()
        .filter(|id| *id > 0)
        .ok_or_else(|| ApiError::bad_request("invalid round id"))?;
    let round = state
        .service
        .delete_round(id)
        .await
        .map_err(|error| match error {
            ServiceError::NotFound => ApiError::not_found("round not found"),
            other => ApiError::internal("failed to delete round", other),
        })?;
    Ok(Json(round))
}

async fn get_all_player_stats(
    State(state): State<AppState>,
    Query(query): Query<DateRangeQuery>,
) -> Result<Json<Vec<PlayerTotalStats>>, ApiError> {
    let (start, end) = date_range(&query);
    let stats = state
        .service
        .get_all_player_total_stats(start, end)
        .await
        .map_err(|error| match error {
            ServiceError::BadRequest(message) => ApiError::bad_request(message),
            other => ApiError::internal("failed to load stats", other),
        })?;
    Ok(Json(stats))
}

async fn get_player_stats_history(
    State(state): State<AppState>,
    Path(raw_id): Path<String>,
    Query(query): Query<DateRangeQuery>,
) -> Result<Json<PlayerStatsHistory>, ApiError> {
    let id = raw_id
        .parse::<i64>()
        .ok()
        .filter(|id| *id > 0)
        .ok_or_else(|| ApiError::bad_request("invalid player id"))?;
    let (start, end) = date_range(&query);
    let history = state
        .service
        .get_player_stats_history(id, start, end)
        .await
        .map_err(|error| match error {
            ServiceError::NotFound => ApiError::not_found("player not found"),
            ServiceError::BadRequest(message) => ApiError::bad_request(message),
            other => ApiError::internal("failed to load player stats", other),
        })?;
    Ok(Json(history))
}

/// SPA 경로는 React의 index.html로 연결하고, 미등록 API는 JSON 404를 반환합니다.
async fn spa_fallback(uri: Uri) -> Response {
    if uri.path().starts_with("/api/") {
        return ApiError::not_found("not found").into_response();
    }
    match tokio::fs::read("web/dist/index.html").await {
        Ok(html) => (
            StatusCode::OK,
            [(header::CONTENT_TYPE, "text/html; charset=utf-8")],
            html,
        )
            .into_response(),
        Err(_) => ApiError::not_found("web/dist/index.html not found; build frontend first")
            .into_response(),
    }
}
