package service

import (
    "context"
    "errors"
    "testing"

    "pr-review-assigner/internal/repo"
)

// Mock репозитория для тестирования
type mockRepo struct {
    users        map[string]*repo.User
    teams        map[string]*repo.Team
    teamMembers  map[string][]string // teamName -> userIDs
    prs          map[string]*repo.PR
    prReviewers  map[string][]string // prID -> reviewerIDs
    assignments  []struct{ prID, userID string }
}

func newMockRepo() *mockRepo {
    return &mockRepo{
        users:       make(map[string]*repo.User),
        teams:       make(map[string]*repo.Team),
        teamMembers: make(map[string][]string),
        prs:         make(map[string]*repo.PR),
        prReviewers: make(map[string][]string),
    }
}

func (m *mockRepo) CreateUser(ctx context.Context, userID, username string) error {
    m.users[userID] = &repo.User{
        ID:       userID,
        Name:     username,
        IsActive: true,
    }
    return nil
}

func (m *mockRepo) GetUserByID(ctx context.Context, userID string) (*repo.User, error) {
    user, exists := m.users[userID]
    if !exists {
        return nil, errors.New("user not found")
    }
    return user, nil
}

func (m *mockRepo) SetUserActive(ctx context.Context, userID string, active bool) error {
    user, exists := m.users[userID]
    if !exists {
        return errors.New("user not found")
    }
    user.IsActive = active
    return nil
}

func (m *mockRepo) TeamExists(ctx context.Context, name string) (bool, error) {
    _, exists := m.teams[name]
    return exists, nil
}

func (m *mockRepo) CreateTeam(ctx context.Context, name string) (int64, error) {
    if _, exists := m.teams[name]; exists {
        return 0, errors.New("team exists")
    }
    m.teams[name] = &repo.Team{
        ID:   int64(len(m.teams) + 1),
        Name: name,
    }
    return m.teams[name].ID, nil
}

func (m *mockRepo) AddMember(ctx context.Context, teamID int64, userID string) error {
    // Находим команду по ID
    var teamName string
    for name, team := range m.teams {
        if team.ID == teamID {
            teamName = name
            break
        }
    }
    if teamName == "" {
        return errors.New("team not found")
    }
    
    m.teamMembers[teamName] = append(m.teamMembers[teamName], userID)
    return nil
}

func (m *mockRepo) GetTeamByName(ctx context.Context, name string) (*repo.Team, error) {
    team, exists := m.teams[name]
    if !exists {
        return nil, errors.New("team not found")
    }
    return team, nil
}

func (m *mockRepo) GetTeamMembers(ctx context.Context, teamName string) ([]repo.User, error) {
    memberIDs := m.teamMembers[teamName]
    var users []repo.User
    for _, id := range memberIDs {
        if user, exists := m.users[id]; exists {
            users = append(users, *user)
        }
    }
    return users, nil
}

func (m *mockRepo) GetActiveTeamMembersExcept(ctx context.Context, teamName string, excludeUserID string) ([]repo.User, error) {
    memberIDs := m.teamMembers[teamName]
    var users []repo.User
    for _, id := range memberIDs {
        if user, exists := m.users[id]; exists && user.IsActive && user.ID != excludeUserID {
            users = append(users, *user)
        }
    }
    return users, nil
}

func (m *mockRepo) PRExists(ctx context.Context, prID string) (bool, error) {
    _, exists := m.prs[prID]
    return exists, nil
}

func (m *mockRepo) CreatePRWithID(ctx context.Context, prID, title, authorID string) error {
    if _, exists := m.prs[prID]; exists {
        return errors.New("PR exists")
    }
    m.prs[prID] = &repo.PR{
        ID:       prID,
        Title:    title,
        AuthorID: authorID,
        Status:   "OPEN",
    }
    return nil
}

func (m *mockRepo) GetPRByID(ctx context.Context, prID string) (*repo.PR, error) {
    pr, exists := m.prs[prID]
    if !exists {
        return nil, errors.New("PR not found")
    }
    return pr, nil
}

func (m *mockRepo) AddReviewer(ctx context.Context, prID, userID string) error {
    m.prReviewers[prID] = append(m.prReviewers[prID], userID)
    return nil
}

func (m *mockRepo) RemoveReviewer(ctx context.Context, prID, userID string) error {
    reviewers := m.prReviewers[prID]
    for i, id := range reviewers {
        if id == userID {
            m.prReviewers[prID] = append(reviewers[:i], reviewers[i+1:]...)
            return nil
        }
    }
    return errors.New("reviewer not found")
}

func (m *mockRepo) GetPRReviewers(ctx context.Context, prID string) ([]repo.User, error) {
    reviewerIDs := m.prReviewers[prID]
    var users []repo.User
    for _, id := range reviewerIDs {
        if user, exists := m.users[id]; exists {
            users = append(users, *user)
        }
    }
    return users, nil
}

func (m *mockRepo) SetPRStatus(ctx context.Context, prID string, status string) error {
    pr, exists := m.prs[prID]
    if !exists {
        return errors.New("PR not found")
    }
    pr.Status = status
    return nil
}

func (m *mockRepo) GetPRsByReviewer(ctx context.Context, userID string) ([]repo.PR, error) {
    var result []repo.PR
    for prID, reviewers := range m.prReviewers {
        for _, reviewerID := range reviewers {
            if reviewerID == userID {
                if pr, exists := m.prs[prID]; exists {
                    result = append(result, *pr)
                }
                break
            }
        }
    }
    return result, nil
}

func (m *mockRepo) GetUserTeam(ctx context.Context, userID string) (string, error) {
    for teamName, members := range m.teamMembers {
        for _, memberID := range members {
            if memberID == userID {
                return teamName, nil
            }
        }
    }
    return "", errors.New("user not in any team")
}

func (m *mockRepo) GetRandomActiveTeamMember(ctx context.Context, teamName, excludeUserID string) (*repo.User, error) {
    members, err := m.GetActiveTeamMembersExcept(ctx, teamName, excludeUserID)
    if err != nil {
        return nil, err
    }
    if len(members) == 0 {
        return nil, errors.New("no active members")
    }
    // Возвращаем первого доступного (в реальности должен быть случайный)
    return &members[0], nil
}

func (m *mockRepo) AddAssignmentEvent(ctx context.Context, prID, userID string) error {
    m.assignments = append(m.assignments, struct{ prID, userID string }{prID, userID})
    return nil
}

func (m *mockRepo) GetAssignmentStats(ctx context.Context) (map[string]int, error) {
    stats := make(map[string]int)
    for _, assignment := range m.assignments {
        stats[assignment.userID]++
    }
    return stats, nil
}

func (m *mockRepo) DeactivateTeamMembers(ctx context.Context, teamID int64) error {
    // Находим команду по ID
    var teamName string
    for name, team := range m.teams {
        if team.ID == teamID {
            teamName = name
            break
        }
    }
    if teamName == "" {
        return errors.New("team not found")
    }

    // Деактивируем всех членов команды
    for _, userID := range m.teamMembers[teamName] {
        if user, exists := m.users[userID]; exists {
            user.IsActive = false
        }
    }
    return nil
}

func (m *mockRepo) GetOpenPRsWithReviewersByUserIDs(ctx context.Context, userIDs []string) ([]repo.PR, error) {
    var result []repo.PR
    for prID, pr := range m.prs {
        if pr.Status != "OPEN" {
            continue
        }
        for _, reviewerID := range m.prReviewers[prID] {
            for _, targetID := range userIDs {
                if reviewerID == targetID {
                    result = append(result, *pr)
                    break
                }
            }
        }
    }
    return result, nil
}

// Тесты

func TestCreateTeam(t *testing.T) {
    mockRepo := newMockRepo()
    service := New(mockRepo)
    ctx := context.Background()

    members := []repo.TeamMember{
        {UserID: "u1", Username: "Alice", IsActive: true},
        {UserID: "u2", Username: "Bob", IsActive: true},
    }

    // Успешное создание команды
    err := service.CreateTeam(ctx, "dev-team", members)
    if err != nil {
        t.Fatalf("CreateTeam failed: %v", err)
    }

    // Попытка создать дубликат команды
    err = service.CreateTeam(ctx, "dev-team", members)
    if err != ErrTeamExists {
        t.Errorf("Expected ErrTeamExists, got %v", err)
    }
}

func TestCreatePR(t *testing.T) {
    mockRepo := newMockRepo()
    service := New(mockRepo)
    ctx := context.Background()

    // Подготовка: создаем команду и пользователей
    members := []repo.TeamMember{
        {UserID: "author1", Username: "Author", IsActive: true},
        {UserID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {UserID: "reviewer2", Username: "Reviewer2", IsActive: true},
    }
    service.CreateTeam(ctx, "dev-team", members)

    // Создание PR
    pr, err := service.CreatePR(ctx, "pr-1", "Test PR", "author1")
    if err != nil {
        t.Fatalf("CreatePR failed: %v", err)
    }

    if pr.ID != "pr-1" {
        t.Errorf("Expected PR ID 'pr-1', got '%s'", pr.ID)
    }

    if pr.Status != "OPEN" {
        t.Errorf("Expected status OPEN, got %s", pr.Status)
    }

    // Проверяем что назначены ревьюверы (исключая автора)
    if len(pr.Reviewers) == 0 {
        t.Error("Expected reviewers to be assigned")
    }

    for _, reviewer := range pr.Reviewers {
        if reviewer.ID == "author1" {
            t.Error("Author should not be self-assigned")
        }
    }

    // Попытка создать дубликат PR
    _, err = service.CreatePR(ctx, "pr-1", "Duplicate PR", "author1")
    if err != ErrPRExists {
        t.Errorf("Expected ErrPRExists, got %v", err)
    }
}

func TestMergePR(t *testing.T) {
    mockRepo := newMockRepo()
    service := New(mockRepo)
    ctx := context.Background()

    // Создаем PR
    members := []repo.TeamMember{
        {UserID: "author1", Username: "Author", IsActive: true},
    }
    service.CreateTeam(ctx, "dev-team", members)
    service.CreatePR(ctx, "pr-1", "Test PR", "author1")

    // Мержим PR
    pr, err := service.MergePR(ctx, "pr-1")
    if err != nil {
        t.Fatalf("MergePR failed: %v", err)
    }

    if pr.Status != "MERGED" {
        t.Errorf("Expected status MERGED, got %s", pr.Status)
    }

    // Проверяем идемпотентность
    pr2, err := service.MergePR(ctx, "pr-1")
    if err != nil {
        t.Errorf("Second MergePR should be idempotent, got error: %v", err)
    }

    if pr2.Status != "MERGED" {
        t.Errorf("Second call should return MERGED status, got %s", pr2.Status)
    }
}

func TestReassignReviewer(t *testing.T) {
    mockRepo := newMockRepo()
    service := New(mockRepo)
    ctx := context.Background()

    // Подготовка: команда с несколькими пользователями
    members := []repo.TeamMember{
        {UserID: "author1", Username: "Author", IsActive: true},
        {UserID: "reviewer1", Username: "Reviewer1", IsActive: true},
        {UserID: "reviewer2", Username: "Reviewer2", IsActive: true},
        {UserID: "reviewer3", Username: "Reviewer3", IsActive: true},
    }
    service.CreateTeam(ctx, "dev-team", members)

    // Создаем PR (автоматически назначит ревьюверов)
    service.CreatePR(ctx, "pr-1", "Test PR", "author1")

    // Получаем текущих ревьюверов через mock repo
    reviewers, _ := mockRepo.GetPRReviewers(ctx, "pr-1")
    
    if len(reviewers) == 0 {
        t.Fatal("No reviewers assigned, cannot test reassignment")
    }

    oldReviewer := reviewers[0].ID

    // Переназначаем ревьювера
    updatedPR, newReviewerID, err := service.ReassignReviewer(ctx, "pr-1", oldReviewer)
    if err != nil {
        t.Fatalf("ReassignReviewer failed: %v", err)
    }

    if newReviewerID == oldReviewer {
        t.Error("New reviewer should be different from old reviewer")
    }

    // Проверяем что старый ревьювер заменен
    foundOld := false
    foundNew := false
    for _, reviewer := range updatedPR.Reviewers {
        if reviewer.ID == oldReviewer {
            foundOld = true
        }
        if reviewer.ID == newReviewerID {
            foundNew = true
        }
    }

    if foundOld {
        t.Error("Old reviewer should be removed from PR")
    }
    if !foundNew {
        t.Error("New reviewer should be added to PR")
    }
}

func TestBulkDeactivateTeam(t *testing.T) {
    mockRepo := newMockRepo()
    service := New(mockRepo)
    ctx := context.Background()

    // Создаем команду с активными пользователями
    members := []repo.TeamMember{
        {UserID: "user1", Username: "User1", IsActive: true},
        {UserID: "user2", Username: "User2", IsActive: true},
    }
    service.CreateTeam(ctx, "team-to-deactivate", members)

    // Деактивируем команду
    err := service.BulkDeactivateTeam(ctx, "team-to-deactivate", false)
    if err != nil {
        t.Fatalf("BulkDeactivateTeam failed: %v", err)
    }

    // Проверяем что пользователи деактивированы
    u1, _ := mockRepo.GetUserByID(ctx, "user1")
    u2, _ := mockRepo.GetUserByID(ctx, "user2")
    
    if u1.IsActive {
        t.Error("User1 should be deactivated")
    }
    if u2.IsActive {
        t.Error("User2 should be deactivated")
    }
}

func TestSetUserActive(t *testing.T) {
    mockRepo := newMockRepo()
    service := New(mockRepo)
    ctx := context.Background()

    // Создаем пользователя через создание команды
    members := []repo.TeamMember{
        {UserID: "user1", Username: "TestUser", IsActive: true},
    }
    service.CreateTeam(ctx, "test-team", members)

    // Деактивируем пользователя
    user, err := service.SetUserActive(ctx, "user1", false)
    if err != nil {
        t.Fatalf("SetUserActive failed: %v", err)
    }

    if user.IsActive {
        t.Error("User should be deactivated")
    }

    // Используем поле Name вместо Username
    if user.Name != "TestUser" {
        t.Errorf("Expected username 'TestUser', got '%s'", user.Name)
    }
}