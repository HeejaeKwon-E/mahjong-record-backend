//! Go service.go의 비즈니스 규칙과 통계 계산을 그대로 옮긴 계층입니다.
use chrono::{NaiveDate, Utc};
use chrono_tz::Asia::Seoul;

use crate::{
    domain::{
        DailyPlayerStats, DateRange, Player, PlayerRecentRound, PlayerStatsHistory,
        PlayerStatsSummary, PlayerTotalStats, RankCounts, Round,
    },
    repository::Repository,
};

const RECENT_ROUND_LIMIT: usize = 50;

/// 비즈니스 검증 오류, 조회 실패 및 DB 오류를 타입으로 구분합니다.
///
/// `thiserror`가 Display/Error/From 구현을 생성하여 직접 구현하던 반복 코드를 제거합니다.
#[derive(Debug, thiserror::Error)]
pub enum ServiceError {
    #[error("{0}")]
    BadRequest(&'static str),
    #[error("not found")]
    NotFound,
    #[error(transparent)]
    Database(#[from] sqlx::Error),
}

#[derive(Clone)]
pub struct Service {
    repo: Repository,
}

impl Service {
    pub fn new(repo: Repository) -> Self {
        Self { repo }
    }

    pub async fn get_players(&self) -> Result<Option<Vec<Player>>, ServiceError> {
        Ok(self.repo.get_all_players().await?)
    }

    pub async fn add_player(&self, name: &str) -> Result<Player, ServiceError> {
        if name.is_empty() {
            return Err(ServiceError::BadRequest("name is required"));
        }
        // Go 정규식 ^[가-힣]{2}\d{2}$: 한글 완성형 두 글자 + ASCII 숫자 두 자리.
        let chars: Vec<char> = name.chars().collect();
        let valid = chars.len() == 4
            && chars[..2].iter().all(|c| ('가'..='힣').contains(c))
            && chars[2..].iter().all(|c| c.is_ascii_digit());
        if !valid {
            return Err(ServiceError::BadRequest("형식: 한글이름 + 2자리 연도"));
        }
        Ok(self.repo.create_player(name).await?)
    }

    pub async fn add_round(&self, date: &str, ranking: &[i64]) -> Result<i64, ServiceError> {
        // 검사 순서는 Go 구현과 동일합니다(길이 → 날짜 공백 → 오늘만 허용).
        if ranking.len() != 4 {
            return Err(ServiceError::BadRequest(
                "ranking must have exactly 4 players",
            ));
        }
        if date.is_empty() {
            return Err(ServiceError::BadRequest("date is required"));
        }
        // 검증은 NaiveDate로 수행하되 기존 API의 동일한 문자열 형식도 엄격히 검사합니다.
        let today = today_kst_date();
        let canonical_date = today.format("%Y-%m-%d").to_string();
        let parsed_date = NaiveDate::parse_from_str(date, "%Y-%m-%d").ok();
        if parsed_date != Some(today) || date != canonical_date {
            return Err(ServiceError::BadRequest(
                "라운드는 오늘 날짜에만 저장할 수 있습니다",
            ));
        }
        // SQLite 스키마의 TEXT 컬럼 형식은 바꾸지 않습니다.
        Ok(self.repo.create_round(&canonical_date, ranking).await?)
    }

    pub async fn get_rounds_by_date(&self, date: &str) -> Result<Option<Vec<Round>>, ServiceError> {
        if date.is_empty() {
            return Err(ServiceError::BadRequest("date is required"));
        }
        Ok(self.repo.get_rounds_by_date(date).await?)
    }

    pub async fn get_players_by_date(
        &self,
        date: &str,
    ) -> Result<Option<Vec<Player>>, ServiceError> {
        if date.is_empty() {
            return Err(ServiceError::BadRequest("date is required"));
        }
        Ok(self.repo.get_players_by_date(date).await?)
    }

    pub async fn delete_round(&self, id: i64) -> Result<Round, ServiceError> {
        self.repo
            .delete_round(id)
            .await?
            .ok_or(ServiceError::NotFound)
    }

    pub async fn get_all_player_total_stats(
        &self,
        start_date: &str,
        end_date: &str,
    ) -> Result<Vec<PlayerTotalStats>, ServiceError> {
        let range = parse_date_range(start_date, end_date)?;
        Ok(self.repo.get_all_player_total_stats(&range).await?)
    }

    pub async fn get_player_stats_history(
        &self,
        player_id: i64,
        start_date: &str,
        end_date: &str,
    ) -> Result<PlayerStatsHistory, ServiceError> {
        if player_id <= 0 {
            return Err(ServiceError::BadRequest("invalid player id"));
        }
        let range = parse_date_range(start_date, end_date)?;
        let player = self
            .repo
            .get_player_by_id(player_id)
            .await?
            .ok_or(ServiceError::NotFound)?;
        let rounds = self
            .repo
            .get_player_round_results(player_id, &range)
            .await?;
        let mut history = PlayerStatsHistory {
            player,
            summary: PlayerStatsSummary::default(),
            rank_counts: RankCounts::default(),
            daily: Vec::new(),
            recent_rounds: Vec::new(),
        };

        // 정렬된 데이터 한 번만 순회해 날짜별 집계와 전체 등수 집계를 계산합니다.
        for round in &rounds {
            if !(1..=4).contains(&round.rank) {
                continue;
            }
            match round.rank {
                1 => history.rank_counts.first += 1,
                2 => history.rank_counts.second += 1,
                3 => history.rank_counts.third += 1,
                4 => history.rank_counts.fourth += 1,
                _ => unreachable!(),
            }
            if history
                .daily
                .last()
                .is_none_or(|daily| daily.date != round.date)
            {
                history.daily.push(DailyPlayerStats {
                    date: round.date.clone(),
                    games: 0,
                    first: 0,
                    second: 0,
                    third: 0,
                    fourth: 0,
                    avg_rank: 0.0,
                });
            }
            let daily = history.daily.last_mut().expect("just added a daily row");
            daily.games += 1;
            match round.rank {
                1 => daily.first += 1,
                2 => daily.second += 1,
                3 => daily.third += 1,
                4 => daily.fourth += 1,
                _ => unreachable!(),
            }
        }
        for daily in &mut history.daily {
            if daily.games > 0 {
                daily.avg_rank =
                    (daily.first + daily.second * 2 + daily.third * 3 + daily.fourth * 4) as f64
                        / daily.games as f64;
            }
        }

        let counts = &history.rank_counts;
        let games = counts.first + counts.second + counts.third + counts.fourth;
        history.summary.games = games;
        if games > 0 {
            let g = games as f64;
            history.summary.avg_rank =
                (counts.first + counts.second * 2 + counts.third * 3 + counts.fourth * 4) as f64
                    / g;
            history.summary.first_rate = counts.first as f64 / g;
            history.summary.top2_rate = (counts.first + counts.second) as f64 / g;
            history.summary.fourth_rate = counts.fourth as f64 / g;
        }

        // Go 코드의 역순 배열 순회 규칙(동일 타임스탬프일 때 높은 ID 우선)을 유지.
        history.recent_rounds = rounds
            .iter()
            .rev()
            .filter(|r| (1..=4).contains(&r.rank))
            .take(RECENT_ROUND_LIMIT)
            .map(|r| PlayerRecentRound {
                round_id: r.round_id,
                date: r.date.clone(),
                rank: r.rank,
                created_at: r.created_at.clone(),
            })
            .collect();
        Ok(history)
    }
}

/// 서비스 내부에서는 날짜를 문자열 대신 `NaiveDate`로 계산합니다.
pub fn today_kst_date() -> NaiveDate {
    Utc::now().with_timezone(&Seoul).date_naive()
}

/// 기존 `GET /api/server-date` JSON의 `YYYY-MM-DD` 문자열을 유지합니다.
pub fn today_kst() -> String {
    today_kst_date().format("%Y-%m-%d").to_string()
}

/// 요청 문자열을 엄격한 날짜 타입으로 변환합니다.
///
/// 빈 문자열은 경계를 지정하지 않은 것으로 해석합니다. 오류 문구는 기존 Go API와
/// 동일하게 유지하고, 이후 계층은 검증된 `NaiveDate`만 전달받습니다.
pub fn parse_date_range(start_date: &str, end_date: &str) -> Result<DateRange, ServiceError> {
    fn parse_one(value: &str) -> Result<Option<NaiveDate>, ServiceError> {
        if value.is_empty() {
            return Ok(None);
        }
        let parsed = NaiveDate::parse_from_str(value, "%Y-%m-%d")
            .map_err(|_| ServiceError::BadRequest("invalid date format; expected YYYY-MM-DD"))?;
        // chrono가 축약된 날짜 등을 허용해도 외부 계약은 YYYY-MM-DD 10자리로 고정.
        if parsed.format("%Y-%m-%d").to_string() != value {
            return Err(ServiceError::BadRequest(
                "invalid date format; expected YYYY-MM-DD",
            ));
        }
        Ok(Some(parsed))
    }

    let start = parse_one(start_date)?;
    let end = parse_one(end_date)?;
    if let (Some(start), Some(end)) = (start, end) {
        if start > end {
            return Err(ServiceError::BadRequest(
                "start_date must not be after end_date",
            ));
        }
    }
    Ok(DateRange { start, end })
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn dates_reject_bad_values_and_reversed_ranges() {
        assert!(parse_date_range("", "").is_ok());
        assert!(parse_date_range("2026-02-29", "").is_err());
        assert!(parse_date_range("2026/10/01", "").is_err());
        assert!(parse_date_range("2026-10-08", "2026-10-07").is_err());
        let range = parse_date_range("2026-10-07", "2026-10-08").unwrap();
        assert_eq!(range.start.unwrap().to_string(), "2026-10-07");
        assert_eq!(range.end.unwrap().to_string(), "2026-10-08");
    }
}
