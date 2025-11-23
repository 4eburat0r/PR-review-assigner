package repo

import (
    "context"
    "github.com/jmoiron/sqlx"
)

// RepoInterface определяет контракт для репозитория
type RepoInterface interface {
    // Users
    CreateUser(ctx context.Context, userID, username string) error
    GetUserByID(ctx context.Context, userID string) (*User, error)
    SetUserActive(ctx context.Context, userID string, active bool) error
    
    // Teams
    TeamExists(ctx context.Context, name string) (bool, error)
    CreateTeam(ctx context.Context, name string) (int64, error)
    AddMember(ctx context.Context, teamID int64, userID string) error
    GetTeamByName(ctx context.Context, name string) (*Team, error)
    GetTeamMembers(ctx context.Context, teamName string) ([]User, error)
    GetActiveTeamMembersExcept(ctx context.Context, teamName string, excludeUserID string) ([]User, error)
    
    // PRs
    PRExists(ctx context.Context, prID string) (bool, error)
    CreatePRWithID(ctx context.Context, prID, title, authorID string) error
    GetPRByID(ctx context.Context, prID string) (*PR, error)
    AddReviewer(ctx context.Context, prID, userID string) error
    RemoveReviewer(ctx context.Context, prID, userID string) error
    GetPRReviewers(ctx context.Context, prID string) ([]User, error)
    SetPRStatus(ctx context.Context, prID string, status string) error
    GetPRsByReviewer(ctx context.Context, userID string) ([]PR, error)
    GetUserTeam(ctx context.Context, userID string) (string, error)
    GetRandomActiveTeamMember(ctx context.Context, teamName, excludeUserID string) (*User, error)
    
    // Assignment events
    AddAssignmentEvent(ctx context.Context, prID, userID string) error
    GetAssignmentStats(ctx context.Context) (map[string]int, error)
    
    // Bulk operations
    DeactivateTeamMembers(ctx context.Context, teamID int64) error
    GetOpenPRsWithReviewersByUserIDs(ctx context.Context, userIDs []string) ([]PR, error)
}

type Repo struct {
    db *sqlx.DB
}

func New(db *sqlx.DB) *Repo {
    return &Repo{db: db}
}


// Структуры данных
type User struct {
    ID       string `json:"user_id" db:"id"`
    Name     string `json:"username" db:"name"`
    IsActive bool   `json:"is_active" db:"is_active"`
    TeamName string `json:"team_name,omitempty" db:"-"`
}

type Team struct {
    ID   int64  `json:"-" db:"id"`
    Name string `json:"team_name" db:"name"`
}

type TeamMember struct {
    UserID   string `json:"user_id" db:"user_id"`
    Username string `json:"username" db:"username"`
    IsActive bool   `json:"is_active" db:"is_active"`
}

type PR struct {
    ID        string `json:"pull_request_id" db:"id"`
    Title     string `json:"pull_request_name" db:"title"`
    AuthorID  string `json:"author_id" db:"author_id"`
    Status    string `json:"status" db:"status"`
    Reviewers []User `json:"assigned_reviewers,omitempty" db:"-"`
}

// Users
func (r *Repo) CreateUser(ctx context.Context, userID, username string) error {
    _, err := r.db.ExecContext(ctx, 
        "INSERT INTO users (id, name) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name = $2", 
        userID, username)
    return err
}

func (r *Repo) GetUserByID(ctx context.Context, userID string) (*User, error) {
    var u User
    err := r.db.GetContext(ctx, &u, "SELECT id, name, is_active FROM users WHERE id=$1", userID)
    if err != nil {
        return nil, err
    }
    return &u, nil
}

func (r *Repo) SetUserActive(ctx context.Context, userID string, active bool) error {
    _, err := r.db.ExecContext(ctx, "UPDATE users SET is_active=$1 WHERE id=$2", active, userID)
    return err
}

// Teams
func (r *Repo) TeamExists(ctx context.Context, name string) (bool, error) {
    var count int
    err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM teams WHERE name = $1", name)
    return count > 0, err
}

func (r *Repo) CreateTeam(ctx context.Context, name string) (int64, error) {
    var id int64
    err := r.db.QueryRowContext(ctx, "INSERT INTO teams (name) VALUES ($1) RETURNING id", name).Scan(&id)
    return id, err
}

func (r *Repo) AddMember(ctx context.Context, teamID int64, userID string) error {
    _, err := r.db.ExecContext(ctx, 
        "INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", 
        teamID, userID)
    return err
}

func (r *Repo) GetTeamByName(ctx context.Context, name string) (*Team, error) {
    var t Team
    err := r.db.GetContext(ctx, &t, "SELECT id, name FROM teams WHERE name=$1", name)
    if err != nil {
        return nil, err
    }
    return &t, nil
}

func (r *Repo) GetTeamMembers(ctx context.Context, teamName string) ([]User, error) {
    var users []User
    err := r.db.SelectContext(ctx, &users, `
        SELECT u.id, u.name, u.is_active 
        FROM users u 
        JOIN team_members tm ON u.id = tm.user_id 
        JOIN teams t ON t.id = tm.team_id 
        WHERE t.name = $1
    `, teamName)
    return users, err
}

func (r *Repo) GetActiveTeamMembersExcept(ctx context.Context, teamName string, excludeUserID string) ([]User, error) {
    var users []User
    err := r.db.SelectContext(ctx, &users, `
        SELECT u.id, u.name, u.is_active 
        FROM users u 
        JOIN team_members tm ON u.id = tm.user_id 
        JOIN teams t ON t.id = tm.team_id 
        WHERE t.name = $1 AND u.is_active = true AND u.id != $2
    `, teamName, excludeUserID)
    return users, err
}

// PRs
func (r *Repo) PRExists(ctx context.Context, prID string) (bool, error) {
    var count int
    err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM prs WHERE id = $1", prID)
    return count > 0, err
}

func (r *Repo) CreatePRWithID(ctx context.Context, prID, title, authorID string) error {
    _, err := r.db.ExecContext(ctx, 
        "INSERT INTO prs (id, title, author_id) VALUES ($1, $2, $3)", 
        prID, title, authorID)
    return err
}

func (r *Repo) GetPRByID(ctx context.Context, prID string) (*PR, error) {
    var p PR
    err := r.db.GetContext(ctx, &p, 
        "SELECT id, title, author_id, status FROM prs WHERE id = $1", prID)
    if err != nil {
        return nil, err
    }
    return &p, nil
}

func (r *Repo) AddReviewer(ctx context.Context, prID, userID string) error {
    _, err := r.db.ExecContext(ctx, 
        "INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", 
        prID, userID)
    return err
}

func (r *Repo) RemoveReviewer(ctx context.Context, prID, userID string) error {
    _, err := r.db.ExecContext(ctx, 
        "DELETE FROM pr_reviewers WHERE pr_id = $1 AND user_id = $2", 
        prID, userID)
    return err
}

func (r *Repo) GetPRReviewers(ctx context.Context, prID string) ([]User, error) {
    var users []User
    err := r.db.SelectContext(ctx, &users, `
        SELECT u.id, u.name, u.is_active 
        FROM pr_reviewers pr 
        JOIN users u ON u.id = pr.user_id 
        WHERE pr.pr_id = $1
    `, prID)
    return users, err
}

func (r *Repo) SetPRStatus(ctx context.Context, prID string, status string) error {
    _, err := r.db.ExecContext(ctx, "UPDATE prs SET status=$1 WHERE id=$2", status, prID)
    return err
}

func (r *Repo) GetPRsByReviewer(ctx context.Context, userID string) ([]PR, error) {
    var prs []PR
    err := r.db.SelectContext(ctx, &prs, `
        SELECT p.id, p.title, p.author_id, p.status 
        FROM prs p 
        JOIN pr_reviewers pr ON p.id = pr.pr_id 
        WHERE pr.user_id = $1
    `, userID)
    return prs, err
}

func (r *Repo) GetUserTeam(ctx context.Context, userID string) (string, error) {
    var teamName string
    err := r.db.GetContext(ctx, &teamName, `
        SELECT t.name 
        FROM teams t 
        JOIN team_members tm ON t.id = tm.team_id 
        WHERE tm.user_id = $1 
        LIMIT 1
    `, userID)
    if err != nil {
        return "", err
    }
    return teamName, nil
}

func (r *Repo) GetRandomActiveTeamMember(ctx context.Context, teamName, excludeUserID string) (*User, error) {
    var user User
    err := r.db.GetContext(ctx, &user, `
        SELECT u.id, u.name, u.is_active 
        FROM users u 
        JOIN team_members tm ON u.id = tm.user_id 
        JOIN teams t ON t.id = tm.team_id 
        WHERE t.name = $1 AND u.is_active = true AND u.id != $2
        ORDER BY RANDOM()
        LIMIT 1
    `, teamName, excludeUserID)
    if err != nil {
        return nil, err
    }
    return &user, nil
}

// Assignment events
func (r *Repo) AddAssignmentEvent(ctx context.Context, prID, userID string) error {
    _, err := r.db.ExecContext(ctx, 
        "INSERT INTO assignment_events (pr_id, user_id) VALUES ($1, $2)", 
        prID, userID)
    return err
}

func (r *Repo) GetAssignmentStats(ctx context.Context) (map[string]int, error) {
    stats := make(map[string]int)
    
    type userStats struct {
        UserID string `db:"user_id"`
        Count  int    `db:"assignment_count"`
    }
    var userStatsList []userStats
    
    err := r.db.SelectContext(ctx, &userStatsList, `
        SELECT user_id, COUNT(*) as assignment_count 
        FROM assignment_events 
        GROUP BY user_id
    `)
    if err != nil {
        return nil, err
    }
    
    for _, stat := range userStatsList {
        stats[stat.UserID] = stat.Count
    }
    
    return stats, nil
}

// Bulk operations
func (r *Repo) DeactivateTeamMembers(ctx context.Context, teamID int64) error {
    _, err := r.db.ExecContext(ctx, 
        "UPDATE users SET is_active = false WHERE id IN (SELECT user_id FROM team_members WHERE team_id=$1)", 
        teamID)
    return err
}

func (r *Repo) GetOpenPRsWithReviewersByUserIDs(ctx context.Context, userIDs []string) ([]PR, error) {
    if len(userIDs) == 0 {
        return []PR{}, nil
    }
    
    query, args, err := sqlx.In(`
        SELECT DISTINCT p.id, p.title, p.author_id, p.status 
        FROM prs p 
        JOIN pr_reviewers pr ON p.id = pr.pr_id 
        WHERE p.status = 'OPEN' AND pr.user_id IN (?)
    `, userIDs)
    if err != nil {
        return nil, err
    }
    
    query = r.db.Rebind(query)
    var prs []PR
    err = r.db.SelectContext(ctx, &prs, query, args...)
    return prs, err
}