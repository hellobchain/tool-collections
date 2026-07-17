package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WeeklyReport struct {
	BaseModel
	UserID            uuid.UUID `gorm:"type:uuid;not null;index"`
	WeekStart         time.Time `gorm:"not null;index"`
	Content           string    `gorm:"type:text;not null"`
	NarrativeType     string    `gorm:"default:'攻坚'"`
	CarryoverFromPrev string    `gorm:"type:jsonb;default:'[]'"`
	NextWeekPlan      string    `gorm:"type:jsonb;default:'[]'"`
	CreatedAt         time.Time
}

func (w *WeeklyReport) BeforeCreate(tx *gorm.DB) error {
	return w.BaseModel.BeforeCreate(tx)
}

type ConfirmCarryoverRequest struct {
	KeptIDs    []string `json:"kept_ids"`
	DroppedIDs []string `json:"dropped_ids"`
}

type GenerateDraftRequest struct {
	NarrativeType string `json:"narrative_type" binding:"oneof=攻坚 协作 稳健"`
	TemplateID    string `json:"template_id"`
	WeekStart     string `json:"week_start"`
}

type FinalizeRequest struct {
	Content       string `json:"content" binding:"required"`
	NarrativeType string `json:"narrative_type"`
	WeekStart     string `json:"week_start"`
}

type WeekStatusResponse struct {
	WeekStart            string              `json:"week_start"`
	Fragments            []FragmentResponse  `json:"fragments"`
	Carryover            []CarryoverResponse `json:"carryover"`
	IsCarryoverConfirmed bool                `json:"is_carryover_confirmed"`
	IsFinalized          bool                `json:"is_finalized"`
	NextWeekPlan         []CarryoverResponse `json:"next_week_plan"`
}

type DraftResponse struct {
	Content   string `json:"content"`
	WeekStart string `json:"week_start"`
}

type PaginatedResponse struct {
	List       interface{} `json:"list"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

type HistoryQueryRequest struct {
	WeekStart string `json:"week_start" form:"week_start"`
	WeekEnd   string `json:"week_end" form:"week_end"`
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"page_size" form:"page_size"`
}

type HistoryItemResponse struct {
	ID            string `json:"id"`
	WeekStart     string `json:"week_start"`
	Content       string `json:"content"`
	NarrativeType string `json:"narrative_type"`
	CreatedAt     string `json:"created_at"`
}
