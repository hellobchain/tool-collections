package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GitProject struct {
	BaseModel
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
	ProjectID   string    `gorm:"not null"`
	ProjectName string    `gorm:"not null"`
	BaseURL     string    `gorm:"not null"`
	Token       string    `gorm:"not null"`
	Branch      string    `gorm:"default:'master'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (p *GitProject) BeforeCreate(tx *gorm.DB) error {
	if err := p.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}
	return nil
}

type GitProjectRequest struct {
	ProjectID   string `json:"project_id" binding:"required"`
	ProjectName string `json:"project_name" binding:"required"`
	BaseURL     string `json:"base_url" binding:"required"`
	Token       string `json:"token" binding:"required"`
	Branch      string `json:"branch"`
}

type GitProjectResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	BaseURL     string `json:"base_url"`
	Token       string `json:"token"`
	Branch      string `json:"branch"`
}
