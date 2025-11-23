package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "pr-review-assigner/internal/repo"
    "pr-review-assigner/internal/service"
)

type Handler struct {
    svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *chi.Mux) {
    r.Get("/health", h.HealthCheck)
    
    // Teams
    r.Post("/team/add", h.CreateTeam)
    r.Get("/team/get", h.GetTeam)
    
    // Users
    r.Post("/users/setIsActive", h.SetUserActive)
    r.Get("/users/getReview", h.GetUserReviews)
    
    // Pull Requests
    r.Post("/pullRequest/create", h.CreatePR)
    r.Post("/pullRequest/merge", h.MergePR)
    r.Post("/pullRequest/reassign", h.ReassignReviewer)
    
    // Additional endpoints
    r.Get("/stats", h.GetStats)
    r.Post("/teams/{team}/deactivate", h.BulkDeactivateTeam)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
    var req struct {
        TeamName string              `json:"team_name"`
        Members  []repo.TeamMember   `json:"members"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "BAD_REQUEST", "Invalid request body", http.StatusBadRequest)
        return
    }
    
    if err := h.svc.CreateTeam(r.Context(), req.TeamName, req.Members); err != nil {
        switch err {
        case service.ErrTeamExists:
            h.sendError(w, "TEAM_EXISTS", "team_name already exists", http.StatusBadRequest)
        default:
            h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        }
        return
    }
    
    team, members, err := h.svc.GetTeam(r.Context(), req.TeamName)
    if err != nil {
        h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        return
    }
    
    response := map[string]interface{}{
        "team": map[string]interface{}{
            "team_name": team.Name,
            "members":   members,
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
    teamName := r.URL.Query().Get("team_name")
    if teamName == "" {
        h.sendError(w, "BAD_REQUEST", "team_name is required", http.StatusBadRequest)
        return
    }
    
    team, members, err := h.svc.GetTeam(r.Context(), teamName)
    if err != nil {
        switch err {
        case service.ErrNotFound:
            h.sendError(w, "NOT_FOUND", "team not found", http.StatusNotFound)
        default:
            h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        }
        return
    }
    
    response := map[string]interface{}{
        "team_name": team.Name,
        "members":   members,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
    var req struct {
        UserID   string `json:"user_id"`
        IsActive bool   `json:"is_active"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "BAD_REQUEST", "Invalid request body", http.StatusBadRequest)
        return
    }
    
    user, err := h.svc.SetUserActive(r.Context(), req.UserID, req.IsActive)
    if err != nil {
        switch err {
        case service.ErrNotFound:
            h.sendError(w, "NOT_FOUND", "user not found", http.StatusNotFound)
        default:
            h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        }
        return
    }
    
    response := map[string]interface{}{
        "user": user,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) CreatePR(w http.ResponseWriter, r *http.Request) {
    var req struct {
        PullRequestID   string `json:"pull_request_id"`
        PullRequestName string `json:"pull_request_name"`
        AuthorID        string `json:"author_id"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "BAD_REQUEST", "Invalid request body", http.StatusBadRequest)
        return
    }
    
    pr, err := h.svc.CreatePR(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
    if err != nil {
        switch err {
        case service.ErrPRExists:
            h.sendError(w, "PR_EXISTS", "PR id already exists", http.StatusConflict)
        case service.ErrNotFound:
            h.sendError(w, "NOT_FOUND", "author/team not found", http.StatusNotFound)
        default:
            h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        }
        return
    }
    
    // Convert reviewers to user IDs
    reviewerIDs := make([]string, len(pr.Reviewers))
    for i, reviewer := range pr.Reviewers {
        reviewerIDs[i] = reviewer.ID
    }
    
    response := map[string]interface{}{
        "pr": map[string]interface{}{
            "pull_request_id":   pr.ID,
            "pull_request_name": pr.Title,
            "author_id":         pr.AuthorID,
            "status":            pr.Status,
            "assigned_reviewers": reviewerIDs,
            "createdAt":         nil, // Можно добавить при необходимости
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) MergePR(w http.ResponseWriter, r *http.Request) {
    var req struct {
        PullRequestID string `json:"pull_request_id"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "BAD_REQUEST", "Invalid request body", http.StatusBadRequest)
        return
    }
    
    pr, err := h.svc.MergePR(r.Context(), req.PullRequestID)
    if err != nil {
        switch err {
        case service.ErrNotFound:
            h.sendError(w, "NOT_FOUND", "PR not found", http.StatusNotFound)
        default:
            h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        }
        return
    }
    
    // Convert reviewers to user IDs
    reviewerIDs := make([]string, len(pr.Reviewers))
    for i, reviewer := range pr.Reviewers {
        reviewerIDs[i] = reviewer.ID
    }
    
    response := map[string]interface{}{
        "pr": map[string]interface{}{
            "pull_request_id":   pr.ID,
            "pull_request_name": pr.Title,
            "author_id":         pr.AuthorID,
            "status":            pr.Status,
            "assigned_reviewers": reviewerIDs,
            "mergedAt":          nil, // Можно добавить при необходимости
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
    var req struct {
        PullRequestID string `json:"pull_request_id"`
        OldUserID     string `json:"old_user_id"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "BAD_REQUEST", "Invalid request body", http.StatusBadRequest)
        return
    }
    
    pr, newUserID, err := h.svc.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
    if err != nil {
        switch err {
        case service.ErrNotFound:
            h.sendError(w, "NOT_FOUND", "PR or user not found", http.StatusNotFound)
        case service.ErrPRMerged:
            h.sendError(w, "PR_MERGED", "cannot reassign on merged PR", http.StatusConflict)
        case service.ErrNotAssigned:
            h.sendError(w, "NOT_ASSIGNED", "reviewer is not assigned to this PR", http.StatusConflict)
        case service.ErrNoCandidate:
            h.sendError(w, "NO_CANDIDATE", "no active replacement candidate in team", http.StatusConflict)
        default:
            h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        }
        return
    }
    
    // Convert reviewers to user IDs
    reviewerIDs := make([]string, len(pr.Reviewers))
    for i, reviewer := range pr.Reviewers {
        reviewerIDs[i] = reviewer.ID
    }
    
    response := map[string]interface{}{
        "pr": map[string]interface{}{
            "pull_request_id":   pr.ID,
            "pull_request_name": pr.Title,
            "author_id":         pr.AuthorID,
            "status":            pr.Status,
            "assigned_reviewers": reviewerIDs,
        },
        "replaced_by": newUserID,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("user_id")
    if userID == "" {
        h.sendError(w, "BAD_REQUEST", "user_id is required", http.StatusBadRequest)
        return
    }
    
    prs, err := h.svc.GetUserReviews(r.Context(), userID)
    if err != nil {
        switch err {
        case service.ErrNotFound:
            h.sendError(w, "NOT_FOUND", "user not found", http.StatusNotFound)
        default:
            h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        }
        return
    }
    
    // Convert to short PR format
    prShorts := make([]map[string]interface{}, len(prs))
    for i, pr := range prs {
        prShorts[i] = map[string]interface{}{
            "pull_request_id":   pr.ID,
            "pull_request_name": pr.Title,
            "author_id":         pr.AuthorID,
            "status":            pr.Status,
        }
    }
    
    response := map[string]interface{}{
        "user_id":        userID,
        "pull_requests": prShorts,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
    stats, err := h.svc.GetStats(r.Context())
    if err != nil {
        h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func (h *Handler) BulkDeactivateTeam(w http.ResponseWriter, r *http.Request) {
    teamName := chi.URLParam(r, "team")
    var req struct {
        Reassign bool `json:"reassign_open_prs"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, "BAD_REQUEST", "Invalid request body", http.StatusBadRequest)
        return
    }
    
    if err := h.svc.BulkDeactivateTeam(r.Context(), teamName, req.Reassign); err != nil {
        switch err {
        case service.ErrNotFound:
            h.sendError(w, "NOT_FOUND", "team not found", http.StatusNotFound)
        default:
            h.sendError(w, "INTERNAL_ERROR", err.Error(), http.StatusInternalServerError)
        }
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) sendError(w http.ResponseWriter, code, message string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]interface{}{
            "code":    code,
            "message": message,
        },
    })
}