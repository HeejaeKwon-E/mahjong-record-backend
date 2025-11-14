package service

import (
	"fmt"
	"mahjong-stat-back/internal/domain"
	"mahjong-stat-back/internal/repository"
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

// AddPlayer는 새 플레이어를 추가합니다.
//
// 비즈니스 룰:
//   - 이름은 공백일 수 없음
func (s *Service) AddPlayer(name string) (domain.Player, error) {
	if name == "" {
		return domain.Player{}, fmt.Errorf("name is required")
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
