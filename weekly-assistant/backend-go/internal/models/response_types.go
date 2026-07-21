package models

// ========== Auth ==========

type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type LoginUserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

type LoginResponse struct {
	Token        string        `json:"token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    int           `json:"expires_in"`
	User         LoginUserInfo `json:"user"`
}

// ========== Week ==========

type WeekStatusResp struct {
	WeekStart            string          `json:"week_start"`
	WeekEnd              string          `json:"week_end"`
	WeekNumber           string          `json:"week_number"`
	Fragments            []FragmentItem  `json:"fragments"`
	Carryover            []CarryoverItem `json:"carryover"`
	IsCarryoverConfirmed bool            `json:"is_carryover_confirmed"`
	NextWeekPlan         []CarryoverItem `json:"next_week_plan"`
	IsFinalized          bool            `json:"is_finalized"`
}

type FragmentItem struct {
	ID         string `json:"id"`
	Content    string `json:"content"`
	Date       string `json:"date"`
	OccurredAt string `json:"occurred_at"`
	IsCarried  bool   `json:"is_carried"`
}

type CarryoverItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type FragmentAddResponse struct {
	ID         string `json:"id"`
	Content    string `json:"content"`
	Date       string `json:"date"`
	WeekStart  string `json:"week_start"`
	OccurredAt string `json:"occurred_at"`
	IsCarried  bool   `json:"is_carried"`
}

type GitlabCommitResponse struct {
	ProjectID   string                   `json:"project_id"`
	ProjectName string                   `json:"project_name"`
	StartDate   string                   `json:"start_date"`
	EndDate     string                   `json:"end_date"`
	Commits     []map[string]interface{} `json:"commits"`
}

type DiffItem struct {
	Path     string      `json:"path"`
	Type     string      `json:"type"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}

type JsonCompareResponse struct {
	Differences []DiffItem `json:"differences"`
	Match       bool
}

type PromptCreateResponse struct {
	ID string `json:"id"`
}

type SummaryGenerateResponse struct {
	Quarter string `json:"quarter"`
	Year    string `json:"year"`
}

type WeekHistoryItem struct {
	ID            string `json:"id"`
	WeekStart     string `json:"week_start"`
	WeekEnd       string `json:"week_end"`
	WeekNumber    string `json:"week_number"`
	NarrativeType string `json:"narrative_type"`
	Content       string `json:"content"`
	CreatedAt     string `json:"created_at"`
}

type SummaryItem struct {
	ID          string `json:"id"`
	PeriodType  string `json:"period_type"`
	PeriodValue string `json:"period_value"`
	Content     string `json:"content"`
	CreatedAt   string `json:"created_at"`
}

// ========== Fragment ==========

type FragmentResponse struct {
	ID         string `json:"id"`
	Content    string `json:"content"`
	Date       string `json:"date"`
	WeekStart  string `json:"week_start"`
	OccurredAt string `json:"occurred_at"`
	IsCarried  bool   `json:"is_carried"`
}

// ========== Git Project ==========

type GitProjectItem struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	BaseURL     string `json:"base_url"`
	Branch      string `json:"branch"`
}

// ========== Prompt ==========

type PromptItem struct {
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

// ========== Skill ==========

type SkillItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	SortOrder   int    `json:"sort_order"`
}

// ========== Contract Upload ==========

type ContractUploadResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     string `json:"size"`
	Status   string `json:"status"`
	FileUUID string `json:"file_uuid"`
}

// ========== Contract Review ==========

type ReviewStartResponse struct {
	TaskID   string `json:"task_id"`
	ReportID string `json:"report_id"`
}

type ReviewProgressResponse struct {
	Percent     int    `json:"percent"`
	CurrentRule string `json:"current_rule"`
	HighRisk    int    `json:"high_risk"`
	MediumRisk  int    `json:"medium_risk"`
	LowRisk     int    `json:"low_risk"`
	Status      string `json:"status"`
}

type ReviewReportResponse struct {
	ID                string       `json:"id"`
	FileName          string       `json:"file_name"`
	ContractType      string       `json:"contract_type"`
	ContractTypeLabel string       `json:"contract_type_label"`
	Position          string       `json:"position"`
	PositionLabel     string       `json:"position_label"`
	StandardsLabel    string       `json:"standards_label"`
	Status            string       `json:"status"`
	Conclusion        string       `json:"conclusion"`
	TotalRules        int          `json:"total_rules"`
	RiskStats         RiskStats    `json:"risk_stats"`
	ReviewStartTime   string       `json:"review_start_time"`
	ReviewEndTime     string       `json:"review_end_time"`
	Reviewer          string       `json:"reviewer"`
	Items             []ReviewItem `json:"items"`
}

type ReviewHistoryItem struct {
	ID                string    `json:"id"`
	FileName          string    `json:"file_name"`
	ContractType      string    `json:"contract_type"`
	ContractTypeLabel string    `json:"contract_type_label"`
	Reviewer          string    `json:"reviewer"`
	ReviewStartTime   string    `json:"review_start_time"`
	ReviewEndTime     string    `json:"review_end_time"`
	RiskStats         RiskStats `json:"risk_stats"`
	TotalRisks        int       `json:"total_risks"`
	Conclusion        string    `json:"conclusion"`
	Status            string    `json:"status"`
	Progress          int       `json:"progress"`
}

type RiskStats struct {
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
}

type ReviewItem struct {
	ID           string `json:"id"`
	Level        string `json:"level"`
	Section      string `json:"section"`
	RuleName     string `json:"rule_name"`
	Description  string `json:"description"`
	Suggestion   string `json:"suggestion"`
	LawRef       string `json:"law_ref"`
	OriginalText string `json:"original_text"`
	Status       string `json:"status"`
	Comment      string `json:"comment"`
}

// ========== Contract Draft ==========

type DraftGenerateResponse struct {
	TaskID string `json:"task_id"`
}

type DraftProgressResponse struct {
	Percent     int    `json:"percent"`
	CurrentStep string `json:"current_step"`
	Status      string `json:"status"`
}

type DraftResultResponse struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	ChangeLog   string `json:"change_log"`
	GeneratedAt string `json:"generated_at"`
	FileName    string `json:"file_name"`
}

type DraftDetailResponse struct {
	ID           string `json:"id"`
	FileName     string `json:"file_name"`
	Requirements string `json:"requirements"`
	Content      string `json:"content"`
	ChangeLog    string `json:"change_log"`
	GeneratedAt  string `json:"generated_at"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
}

type DraftHistoryItem struct {
	ID           string `json:"id"`
	FileName     string `json:"file_name"`
	Requirements string `json:"requirements"`
	GeneratedAt  string `json:"generated_at"`
	ContentLen   int    `json:"content_len"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
}

// ========== Contract Extract ==========

type ExtractResultItem struct {
	ID       string                 `json:"id"`
	FileID   string                 `json:"file_id"`
	FileName string                 `json:"file_name"`
	Data     map[string]interface{} `json:"data"`
	Status   string                 `json:"status"`
	ErrorMsg string                 `json:"error_msg"`
}

type ExtractResultResponse struct {
	TaskID   string               `json:"task_id"`
	TaskName string               `json:"task_name"`
	Fields   []ExtractFieldConfig `json:"fields"`
	Results  []ExtractResultItem  `json:"results"`
	Status   string               `json:"status"`
	Progress int                  `json:"progress"`
}

type ExtractHistoryItem struct {
	ID         string `json:"id"`
	TaskName   string `json:"task_name"`
	FileCount  int    `json:"file_count"`
	FieldCount int    `json:"field_count"`
	Status     string `json:"status"`
	Progress   int    `json:"progress"`
	CreatedAt  string `json:"created_at"`
}

type ExtractStartResponse struct {
	TaskID string `json:"task_id"`
}

type ExtractProgressResponse struct {
	Percent int    `json:"percent"`
	Step    string `json:"step"`
	Status  string `json:"status"`
}
