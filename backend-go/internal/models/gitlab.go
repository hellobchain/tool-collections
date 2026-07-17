package models

type GitLabCommitsRequest struct {
	BaseURL     string `json:"base_url" binding:"required"`
	Token       string `json:"token" binding:"required"`
	ProjectID   string `json:"project_id" binding:"required"`
	ProjectName string `json:"project_name" binding:"required"`
	Branch      string `json:"branch" binding:"required"`
	Email       string `json:"email"`
	StartDate   string `json:"start_date" binding:"required"`
	EndDate     string `json:"end_date" binding:"required"`
}

type GitLabCommit struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Message      string `json:"message"`
	AuthoredDate string `json:"authored_date"`
	WebURL       string `json:"web_url"`
}

type GitlabCommitRaw struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	AuthoredDate string `json:"authored_date"`
	WebURL      string `json:"web_url"`
	AuthorEmail string `json:"author_email"`
}
