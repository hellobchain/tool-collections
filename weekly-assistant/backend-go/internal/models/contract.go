package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContractFile struct {
	BaseModel
	UserID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	FileName     string         `gorm:"size:512;not null" json:"name"`
	FileSize     int64          `json:"size"`
	FileUUID     string         `gorm:"size:128;not null;index" json:"file_uuid"`
	FileSavePath string         `gorm:"size:512;not null;index" json:"file_save_path"`
	Bucket       string         `gorm:"size:128" json:"bucket"`
	FileType     string         `gorm:"size:32" json:"file_type"`
	Status       string         `gorm:"size:32;default:parsed" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (f *ContractFile) BeforeCreate(tx *gorm.DB) error {
	return f.BaseModel.BeforeCreate(tx)
}

type ContractReview struct {
	BaseModel
	UserID            uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	FileName          string         `gorm:"size:512" json:"file_name"`
	FileIDs           string         `gorm:"type:text" json:"-"`
	ContractType      string         `gorm:"size:64" json:"contract_type"`
	ContractTypeLabel string         `gorm:"size:128" json:"contract_type_label"`
	Position          string         `gorm:"size:32" json:"position"`
	PositionLabel     string         `gorm:"size:64" json:"position_label"`
	Standards         string         `gorm:"type:text" json:"-"`
	StandardsLabel    string         `gorm:"size:256" json:"standards_label"`
	Status            string         `gorm:"size:32;default:pending" json:"status"`
	Progress          int            `gorm:"default:0" json:"progress"`
	CurrentRule       string         `gorm:"size:256" json:"current_rule"`
	HighRisk          int            `gorm:"default:0" json:"high_risk"`
	MediumRisk        int            `gorm:"default:0" json:"medium_risk"`
	LowRisk           int            `gorm:"default:0" json:"low_risk"`
	TotalRules        int            `gorm:"default:0" json:"total_rules"`
	Conclusion        string         `gorm:"size:128" json:"conclusion"`
	Reviewer          string         `gorm:"size:64" json:"reviewer"`
	ReviewStartTime   string         `gorm:"size:32" json:"review_start_time"`
	ReviewEndTime     string         `gorm:"size:32" json:"review_end_time"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (r *ContractReview) BeforeCreate(tx *gorm.DB) error {
	return r.BaseModel.BeforeCreate(tx)
}

type ContractReviewItem struct {
	BaseModel
	ReviewID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"review_id"`
	Level        string         `gorm:"size:16" json:"level"`
	Section      string         `gorm:"size:64" json:"section"`
	RuleName     string         `gorm:"size:256" json:"rule_name"`
	Description  string         `gorm:"type:text" json:"description"`
	Suggestion   string         `gorm:"type:text" json:"suggestion"`
	LawRef       string         `gorm:"type:text" json:"law_ref"`
	OriginalText string         `gorm:"type:text" json:"original_text"`
	Status       string         `gorm:"size:32;default:open" json:"status"`
	Comment      string         `gorm:"type:text" json:"comment"`
	SortOrder    int            `gorm:"default:0" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (i *ContractReviewItem) BeforeCreate(tx *gorm.DB) error {
	return i.BaseModel.BeforeCreate(tx)
}
