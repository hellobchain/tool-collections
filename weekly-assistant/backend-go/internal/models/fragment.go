package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Fragment struct {
	BaseModel
	UserID     uuid.UUID `gorm:"type:uuid;not null;index"`
	WeekStart  time.Time `gorm:"not null;index"`
	Content    string    `gorm:"type:text;not null"`
	Source     string    `gorm:"default:'manual'"`
	OccurredAt *time.Time
	IsCarried  bool `gorm:"default:false"`
	CreatedAt  time.Time
}

func (f *Fragment) BeforeCreate(tx *gorm.DB) error {
	return f.BaseModel.BeforeCreate(tx)
}

type AddFragmentRequest struct {
	Content    string     `json:"content" binding:"required"`
	Date       string     `json:"date"`
	OccurredAt *time.Time `json:"occurred_at"`
}

type ListFragmentsQuery struct {
	WeekStart string `json:"week_start" form:"week_start"`
	Date      string `json:"date" form:"date"`
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"page_size" form:"page_size"`
}

type CarryoverResponse struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	FromWeek string `json:"from_week"`
}
