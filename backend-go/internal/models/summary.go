package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Summary struct {
	BaseModel
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
	PeriodType  string    `gorm:"not null"`
	PeriodValue string    `gorm:"not null"`
	Content     string    `gorm:"type:text;not null"`
	CreatedAt   time.Time
}

func (s *Summary) BeforeCreate(tx *gorm.DB) error {
	return s.BaseModel.BeforeCreate(tx)
}
