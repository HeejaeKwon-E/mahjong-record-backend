package service

import (
	"fmt"
	"mahjong-stat-back/internal/domain"
	"mahjong-stat-back/internal/repository"
	"regexp"
	"time"
)

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
	return s.repo.GetAllPlayerTotalStats(startDate, endDate)
}
