package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PromptTemplate struct {
	BaseModel
	UserID             *uuid.UUID `gorm:"type:uuid;index"`
	Name               string     `gorm:"not null"`
	Category           string     `gorm:"default:'custom'"`
	PromptType         string     `gorm:"default:'weekly'"`
	SystemPrompt       string     `gorm:"type:text;not null"`
	UserPromptTemplate string     `gorm:"type:text;not null"`
	Description        string     `gorm:"type:text"`
	IsActive           bool       `gorm:"default:true"`
	SortOrder          int        `gorm:"default:0"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (p *PromptTemplate) BeforeCreate(tx *gorm.DB) error {
	return p.BaseModel.BeforeCreate(tx)
}

type CreatePromptTemplateRequest struct {
	Name               string `json:"name" binding:"required"`
	SystemPrompt       string `json:"system_prompt" binding:"required"`
	UserPromptTemplate string `json:"user_prompt_template" binding:"required"`
	Description        string `json:"description"`
	PromptType         string `json:"prompt_type"`
}

type UpdatePromptTemplateRequest struct {
	Name               string `json:"name"`
	SystemPrompt       string `json:"system_prompt"`
	UserPromptTemplate string `json:"user_prompt_template"`
	Description        string `json:"description"`
	IsActive           *bool  `json:"is_active"`
	SortOrder          *int   `json:"sort_order"`
	PromptType         string `json:"prompt_type"`
}

type PromptTemplateResponse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Category           string `json:"category"`
	PromptType         string `json:"prompt_type"`
	SystemPrompt       string `json:"system_prompt"`
	UserPromptTemplate string `json:"user_prompt_template"`
	Description        string `json:"description"`
	IsActive           bool   `json:"is_active"`
	SortOrder          int    `json:"sort_order"`
	IsSystem           bool   `json:"is_system"`
}
