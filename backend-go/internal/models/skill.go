package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Skill struct {
	BaseModel
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"not null"`
	Description string    `gorm:"type:text;not null"`
	IsActive    bool      `gorm:"default:true"`
	SortOrder   int       `gorm:"default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *Skill) BeforeCreate(tx *gorm.DB) error {
	return s.BaseModel.BeforeCreate(tx)
}

type SkillRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	IsActive    *bool  `json:"is_active"`
	SortOrder   *int   `json:"sort_order"`
}

type SkillResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	SortOrder   int    `json:"sort_order"`
}
