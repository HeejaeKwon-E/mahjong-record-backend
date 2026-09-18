package service

import (
	"errors"
	"fmt"
	"mahjong-stat-back/internal/domain"
	"mahjong-stat-back/internal/repository"
	"regexp"
	"time"
)

var (
	// ErrInvalidPlayerID는 0 이하의 플레이어 ID가 전달됐을 때 반환합니다.
	ErrInvalidPlayerID = errors.New("invalid player id")
	// ErrInvalidDateFormat은 YYYY-MM-DD가 아닌 날짜 쿼리에 사용합니다.
	ErrInvalidDateFormat = errors.New("invalid date format; expected YYYY-MM-DD")
	// ErrInvalidDateRange는 시작일이 종료일보다 늦을 때 반환합니다.
	ErrInvalidDateRange = errors.New("start_date must not be after end_date")
)

const recentRoundLimit = 50

// Service는 마작 기록 도메인 로직을 담당합니다.
type Service struct {
	repo *repository.Repository
}

// NewService는 Service 인스턴스를 생성합니다.
//
// 매개변수:
//   - repo: 데이터 레포지토리
//
// 반환값:
//   - *Service: 서비스 인스턴스
func NewService(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// GetPlayers는 전체 플레이어 목록을 반환합니다.
func (s *Service) GetPlayers() ([]domain.Player, error) {
	return s.repo.GetAllPlayers()
}

func isValidPlayerName(name string) bool {
	re := regexp.MustCompile(`^[가-힣]{2}\d{2}$`)
	return re.MatchString(name)
}

// AddPlayer는 새 플레이어를 추가합니다.
//
// 비즈니스 룰:
//   - 이름은 공백일 수 없음
func (s *Service) AddPlayer(name string) (*domain.Player, error) {
	if name == "" {
		return &domain.Player{}, fmt.Errorf("name is required")
	}
	if !isValidPlayerName(name) {
		return &domain.Player{}, fmt.Errorf("형식: 한글이름 + 2자리 연도")
	}

	// TODO: 중복 이름에 대한 비즈니스 룰을 여기서 처리할 수도 있음.
	return s.repo.CreatePlayer(name)
}

// AddRound는 라운드를 추가합니다.
//
// 비즈니스 룰:
//   - ranking 길이는 4여야 함 (마작 4인 고정)
func (s *Service) AddRound(date string, ranking []int) (int64, error) {
	if len(ranking) != 4 {
		return 0, fmt.Errorf("ranking must have exactly 4 players")
	}
	if date == "" {
		return 0, fmt.Errorf("date is required")
	}
	// 오늘 날짜
	today := time.Now().Format("2006-01-02")

	// 🔹 클라이언트가 보낸 date가 오늘이 아니면 거부
	if date != today {
		return 0, fmt.Errorf("라운드는 오늘 날짜에만 저장할 수 있습니다")
	}
	// 🔹 아예 서버 쪽에서 date를 강제로 오늘로 맞춰도 됨
	date = today
	return s.repo.CreateRound(date, ranking)
}

// GetRoundsByDate는 특정 날짜의 라운드를 조회합니다.
//
// 매개변수:
//   - date: 조회할 날짜 (YYYY-MM-DD)
//
// 반환값:
//   - []Round: 라운드 목록
//   - error: 에러 정보
func (s *Service) GetRoundsByDate(date string) ([]domain.Round, error) {
	if date == "" {
		return nil, fmt.Errorf("date is required")
	}
	return s.repo.GetRoundsByDate(date)
}

// GetPlayersByDate는 특정 날짜에 플레이한 플레이어들을 반환합니다.
func (s *Service) GetPlayersByDate(date string) ([]domain.Player, error) {
	if date == "" {
		return nil, fmt.Errorf("date is required")
	}
	return s.repo.GetPlayersByDate(date)
}

// DeleteRound 는 라운드를 삭제하고 삭제된 라운드 정보를 반환합니다.
//
// 매개변수:
//   - id: 삭제할 라운드 ID
//
// 반환값:
//   - *domain.Round: 삭제된 라운드 정보
//   - error: 에러 정보 (존재하지 않는 ID일 경우 sql.ErrNoRows)
func (s *Service) DeleteRound(id int) (*domain.Round, error) {
	round, err := s.repo.DeleteRound(id)
	if err != nil {
		// 여기서 sql.ErrNoRows, DB 에러 등을 그대로 위로 올림
		return &domain.Round{}, err
	}
	return round, nil
}

// GetAllPlayerTotalStats 함수는 전체 플레이어 통계를 반환합니다.
//
// 매개변수:
//   - 없음
//
// 반환값:
//   - []domain.PlayerTotalStats: 플레이어 전체 통계
//   - error: 에러 정보
func (s *Service) GetAllPlayerTotalStats(startDate, endDate string) ([]domain.PlayerTotalStats, error) {
	if err := validateOptionalDate(startDate); err != nil {
		return nil, err
	}
	if err := validateOptionalDate(endDate); err != nil {
		return nil, err
	}
	if startDate != "" && endDate != "" && startDate > endDate {
		return nil, ErrInvalidDateRange
	}

	return s.repo.GetAllPlayerTotalStats(startDate, endDate)
}

// validateOptionalDate는 빈 문자열 또는 엄격한 YYYY-MM-DD 형식만 허용합니다.
// time.Parse만 사용하면 일부 비정상 입력이 의도치 않게 정규화될 수 있으므로
// 다시 포맷한 문자열까지 비교해 API 규격을 명확하게 유지합니다.
func validateOptionalDate(value string) error {
	if value == "" {
		return nil
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return ErrInvalidDateFormat
	}
	return nil
}

// addRank는 등수별 횟수를 누적합니다.
// DB의 rank 제약은 현재 스키마에 없으므로 1~4 이외 값은 통계에서 제외합니다.
func addRank(counts *domain.RankCounts, rank int) {
	switch rank {
	case 1:
		counts.First++
	case 2:
		counts.Second++
	case 3:
		counts.Third++
	case 4:
		counts.Fourth++
	}
}

// rankScore는 등수 횟수로부터 평균순위 계산용 합계를 만듭니다.
func rankScore(counts domain.RankCounts) int64 {
	return counts.First + counts.Second*2 + counts.Third*3 + counts.Fourth*4
}

// GetPlayerStatsHistory는 플레이어 한 명의 기간 요약과 시계열을 반환합니다.
// Repository에서 정렬된 원천 라운드를 한 번 순회해 전체/일별 통계를 동시에 계산합니다.
func (s *Service) GetPlayerStatsHistory(
	playerID int,
	startDate string,
	endDate string,
) (*domain.PlayerStatsHistory, error) {
	if playerID <= 0 {
		return nil, ErrInvalidPlayerID
	}
	if err := validateOptionalDate(startDate); err != nil {
		return nil, err
	}
	if err := validateOptionalDate(endDate); err != nil {
		return nil, err
	}
	if startDate != "" && endDate != "" && startDate > endDate {
		return nil, ErrInvalidDateRange
	}

	player, err := s.repo.GetPlayerByID(playerID)
	if err != nil {
		return nil, err
	}

	rounds, err := s.repo.GetPlayerRoundResults(playerID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	history := &domain.PlayerStatsHistory{
		Player:       *player,
		Daily:        make([]domain.DailyPlayerStats, 0),
		RecentRounds: make([]domain.PlayerRecentRound, 0),
	}

	// 날짜별 결과가 이미 오름차순이므로 마지막 항목만 확인해도 같은 날짜인지 알 수 있습니다.
	for _, round := range rounds {
		if round.Rank < 1 || round.Rank > 4 {
			continue
		}

		addRank(&history.RankCounts, round.Rank)

		if len(history.Daily) == 0 || history.Daily[len(history.Daily)-1].Date != round.Date {
			history.Daily = append(history.Daily, domain.DailyPlayerStats{Date: round.Date})
		}

		daily := &history.Daily[len(history.Daily)-1]
		daily.Games++
		switch round.Rank {
		case 1:
			daily.First++
		case 2:
			daily.Second++
		case 3:
			daily.Third++
		case 4:
			daily.Fourth++
		}
	}

	for index := range history.Daily {
		daily := &history.Daily[index]
		if daily.Games == 0 {
			continue
		}
		daily.AvgRank = float64(
			daily.First+daily.Second*2+daily.Third*3+daily.Fourth*4,
		) / float64(daily.Games)
	}

	history.Summary.Games = history.RankCounts.First +
		history.RankCounts.Second +
		history.RankCounts.Third +
		history.RankCounts.Fourth

	if history.Summary.Games > 0 {
		games := float64(history.Summary.Games)
		history.Summary.AvgRank = float64(rankScore(history.RankCounts)) / games
		history.Summary.FirstRate = float64(history.RankCounts.First) / games
		history.Summary.Top2Rate = float64(
			history.RankCounts.First+history.RankCounts.Second,
		) / games
		history.Summary.FourthRate = float64(history.RankCounts.Fourth) / games
	}

	// 프론트에서 최근 10/20/50게임을 즉시 전환할 수 있도록 최신순 최대 50개를 반환합니다.
	for index := len(rounds) - 1; index >= 0 && len(history.RecentRounds) < recentRoundLimit; index-- {
		round := rounds[index]
		if round.Rank < 1 || round.Rank > 4 {
			continue
		}
		history.RecentRounds = append(history.RecentRounds, domain.PlayerRecentRound{
			RoundID:   round.RoundID,
			Date:      round.Date,
			Rank:      round.Rank,
			CreatedAt: round.CreatedAt,
		})
	}

	return history, nil
}
