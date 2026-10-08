//! 서버 시작점. 실행 디렉터리의 config.json 및 mahjong.db를 사용합니다.

use std::{fs, net::SocketAddr, path::Path, time::Duration};

use mahjong_stat_back::{api, repository::Repository, schema, service::Service};
use serde::Deserialize;
use sqlx::sqlite::{SqliteConnectOptions, SqlitePoolOptions};
use tracing_subscriber::EnvFilter;

#[derive(Deserialize)]
struct Config {
    mode: String,
    port: u16,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 기존 Go 버전과 동일하게 현재 작업 디렉터리에서 설정 파일을 찾습니다.
    let config: Config = serde_json::from_slice(&fs::read("config.json")?)?;
    let default_level = if config.mode == "debug" {
        "debug"
    } else {
        "info"
    };
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new(default_level)),
        )
        .init();

    // 새 파일이면 생성하지만, 기존 DB가 있으면 구조나 데이터를 초기화하지 않습니다.
    // SQLite foreign_keys는 연결 단위 설정이므로 pool의 모든 연결에 적용합니다.
    let options = SqliteConnectOptions::new()
        .filename(Path::new("mahjong.db"))
        .create_if_missing(true)
        .foreign_keys(true)
        .busy_timeout(Duration::from_secs(5));
    let pool = SqlitePoolOptions::new()
        .max_connections(5)
        .connect_with(options)
        .await?;

    schema::init_schema(&pool).await?;
    let app = api::build_router(Service::new(Repository::new(pool)));
    let addr = SocketAddr::from(([0, 0, 0, 0], config.port));
    let listener = tokio::net::TcpListener::bind(addr).await?;
    tracing::info!(%addr, "mahjong stat Rust server started");
    axum::serve(listener, app).await?;
    Ok(())
}
