//! 마작 기록 서버의 HTTP/API/도메인/DB 계층.
//!
//! Go 버전에서 쓰는 JSON 형식과 SQLite 스키마를 그대로 재사용하기 위해
//! 각 계층을 분리합니다. integration test에서 build_router()를 직접 사용합니다.

pub mod api;
pub mod domain;
pub mod repository;
pub mod schema;
pub mod service;
