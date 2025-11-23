package service

import (
    "context"
    "errors"
    "math/rand"
    "time"

    "pr-review-assigner/internal/repo"
)

var (
    ErrTeamExists    = errors.New("team already exists")
    ErrPRExists      = errors.New("PR already exists") 
    ErrPRMerged      = errors.New("PR is merged")
    ErrNotAssigned   = errors.New("reviewer not assigned")
    ErrNoCandidate   = errors.New("no active candidate in team")
    ErrNotFound      = errors.New("resource not found")
)

type Service struct {
    Repo repo.RepoInterface  // Изменено на интерфейс
}

func New(r repo.RepoInterface) *Service {  // Принимает интерфейс
    rand.Seed(time.Now().UnixNano())
    return &Service{Repo: r}
}

// CreateTeam создает команду с участниками
func (s *Service) CreateTeam(ctx context.Context, teamName string, members []repo.TeamMember) error {
    exists, err := s.Repo.TeamExists(ctx, teamName)
    if err != nil {
        return err
    }
    if exists {
        return ErrTeamExists
    }

    teamID, err := s.Repo.CreateTeam(ctx, teamName)
    if err != nil {
        return err
    }

    for _, member := range members {
        // Создаем/обновляем пользователя
        if err := s.Repo.CreateUser(ctx, member.UserID, member.Username); err != nil {
            return err
        }

        // Устанавливаем активность
        if err := s.Repo.SetUserActive(ctx, member.UserID, member.IsActive); err != nil {
            return err
        }

        // Добавляем в команду
        if err := s.Repo.AddMember(ctx, teamID, member.UserID); err != nil {
            return err
        }
    }

    return nil
}

// GetTeam возвращает команду с участниками
func (s *Service) GetTeam(ctx context.Context, teamName string) (*repo.Team, []repo.User, error) {
    team, err := s.Repo.GetTeamByName(ctx, teamName)
    if err != nil {
        return nil, nil, ErrNotFound
    }

    members, err := s.Repo.GetTeamMembers(ctx, teamName)
    if err != nil {
        return nil, nil, err
    }

    return team, members, nil
}

// SetUserActive устанавливает флаг активности пользователя
func (s *Service) SetUserActive(ctx context.Context, userID string, active bool) (*repo.User, error) {
    user, err := s.Repo.GetUserByID(ctx, userID)
    if err != nil {
        return nil, ErrNotFound
    }

    if err := s.Repo.SetUserActive(ctx, userID, active); err != nil {
        return nil, err
    }

    // Получаем команду пользователя
    teamName, _ := s.Repo.GetUserTeam(ctx, userID)

    user.TeamName = teamName
    user.IsActive = active

    return user, nil
}

// CreatePR создает PR и назначает ревьюверов
func (s *Service) CreatePR(ctx context.Context, prID, prName, authorID string) (*repo.PR, error) {
    exists, err := s.Repo.PRExists(ctx, prID)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, ErrPRExists
    }

    // Проверяем существование автора
    _, err = s.Repo.GetUserByID(ctx, authorID)
    if err != nil {
        return nil, ErrNotFound
    }

    // Получаем команду автора
    teamName, err := s.Repo.GetUserTeam(ctx, authorID)
    if err != nil {
        return nil, errors.New("author has no team")
    }

    // Создаем PR
    if err := s.Repo.CreatePRWithID(ctx, prID, prName, authorID); err != nil {
        return nil, err
    }

    // Назначаем ревьюверов
    reviewers, err := s.assignReviewers(ctx, teamName, authorID)
    if err != nil {
        // PR создан, но ревьюверы не назначены - это допустимо
    }

    // Добавляем ревьюверов в PR
    for _, reviewer := range reviewers {
        if err := s.Repo.AddReviewer(ctx, prID, reviewer.ID); err != nil {
            continue
        }
        s.Repo.AddAssignmentEvent(ctx, prID, reviewer.ID)
    }

    pr := &repo.PR{
        ID:        prID,
        Title:     prName,
        AuthorID:  authorID,
        Status:    "OPEN",
        Reviewers: reviewers,
    }

    return pr, nil
}

// assignReviewers назначает до 2 случайных активных ревьюверов из команды
func (s *Service) assignReviewers(ctx context.Context, teamName, excludeUserID string) ([]repo.User, error) {
    candidates, err := s.Repo.GetActiveTeamMembersExcept(ctx, teamName, excludeUserID)
    if err != nil {
        return nil, err
    }

    if len(candidates) == 0 {
        return []repo.User{}, nil
    }

    // Перемешиваем кандидатов
    rand.Shuffle(len(candidates), func(i, j int) {
        candidates[i], candidates[j] = candidates[j], candidates[i]
    })

    // Берем до 2 кандидатов
    limit := 2
    if len(candidates) < limit {
        limit = len(candidates)
    }

    return candidates[:limit], nil
}

// MergePR помечает PR как мерженный
func (s *Service) MergePR(ctx context.Context, prID string) (*repo.PR, error) {
    pr, err := s.Repo.GetPRByID(ctx, prID)
    if err != nil {
        return nil, ErrNotFound
    }

    if pr.Status == "MERGED" {
        // Идемпотентность - возвращаем текущее состояние
        reviewers, _ := s.Repo.GetPRReviewers(ctx, prID)
        pr.Reviewers = reviewers
        return pr, nil
    }

    if err := s.Repo.SetPRStatus(ctx, prID, "MERGED"); err != nil {
        return nil, err
    }

    reviewers, err := s.Repo.GetPRReviewers(ctx, prID)
    if err != nil {
        reviewers = []repo.User{}
    }

    mergedPR := &repo.PR{
        ID:        pr.ID,
        Title:     pr.Title,
        AuthorID:  pr.AuthorID,
        Status:    "MERGED",
        Reviewers: reviewers,
    }

    return mergedPR, nil
}

// ReassignReviewer переназначает ревьювера
func (s *Service) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*repo.PR, string, error) {
    // Проверяем PR
    pr, err := s.Repo.GetPRByID(ctx, prID)
    if err != nil {
        return nil, "", ErrNotFound
    }

    if pr.Status == "MERGED" {
        return nil, "", ErrPRMerged
    }

    // Проверяем что старый ревьювер назначен
    reviewers, err := s.Repo.GetPRReviewers(ctx, prID)
    if err != nil {
        return nil, "", err
    }

    oldUserAssigned := false
    for _, reviewer := range reviewers {
        if reviewer.ID == oldUserID {
            oldUserAssigned = true
            break
        }
    }

    if !oldUserAssigned {
        return nil, "", ErrNotAssigned
    }

    // Получаем команду старого ревьювера
    teamName, err := s.Repo.GetUserTeam(ctx, oldUserID)
    if err != nil {
        return nil, "", errors.New("old reviewer has no team")
    }

    // Ищем замену из команды старого ревьювера
    newReviewer, err := s.Repo.GetRandomActiveTeamMember(ctx, teamName, oldUserID)
    if err != nil {
        return nil, "", ErrNoCandidate
    }

    // Выполняем замену
    if err := s.Repo.RemoveReviewer(ctx, prID, oldUserID); err != nil {
        return nil, "", err
    }

    if err := s.Repo.AddReviewer(ctx, prID, newReviewer.ID); err != nil {
        return nil, "", err
    }

    // Записываем событие назначения
    s.Repo.AddAssignmentEvent(ctx, prID, newReviewer.ID)

    // Получаем обновленный список ревьюверов
    updatedReviewers, _ := s.Repo.GetPRReviewers(ctx, prID)

    updatedPR := &repo.PR{
        ID:        pr.ID,
        Title:     pr.Title,
        AuthorID:  pr.AuthorID,
        Status:    pr.Status,
        Reviewers: updatedReviewers,
    }

    return updatedPR, newReviewer.ID, nil
}

// GetUserReviews возвращает PR где пользователь ревьювер
func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]repo.PR, error) {
    _, err := s.Repo.GetUserByID(ctx, userID)
    if err != nil {
        return nil, ErrNotFound
    }

    prs, err := s.Repo.GetPRsByReviewer(ctx, userID)
    if err != nil {
        return nil, err
    }

    return prs, nil
}

// GetStats возвращает статистику назначений
func (s *Service) GetStats(ctx context.Context) (map[string]interface{}, error) {
    stats, err := s.Repo.GetAssignmentStats(ctx)
    if err != nil {
        return nil, err
    }

    result := map[string]interface{}{
        "assignment_stats": stats,
        "timestamp":        time.Now().Format(time.RFC3339),
    }

    return result, nil
}

// BulkDeactivateTeam массово деактивирует пользователей команды
func (s *Service) BulkDeactivateTeam(ctx context.Context, teamName string, reassign bool) error {
    team, err := s.Repo.GetTeamByName(ctx, teamName)
    if err != nil {
        return ErrNotFound
    }

    // Деактивируем пользователей
    if err := s.Repo.DeactivateTeamMembers(ctx, team.ID); err != nil {
        return err
    }

    return nil
}