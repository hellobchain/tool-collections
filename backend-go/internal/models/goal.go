package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Goal struct {
	BaseModel
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Quarter   string    `gorm:"not null"`
	Content   string    `gorm:"type:text;not null"`
	Status    string    `gorm:"default:'active'"`
	CreatedAt time.Time
}

func (g *Goal) BeforeCreate(tx *gorm.DB) error {
	return g.BaseModel.BeforeCreate(tx)
}
