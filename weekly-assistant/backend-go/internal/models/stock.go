package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StockAnalysisTask struct {
	BaseModel
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	StockCode    string     `gorm:"size:32;not null;index" json:"stock_code"`
	StockName    string     `gorm:"size:128" json:"stock_name"`
	ReportType   string     `gorm:"size:32;default:detailed" json:"report_type"`
	Status       string     `gorm:"size:32;default:pending;index" json:"status"`
	Progress     int        `gorm:"default:0" json:"progress"`
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`
	Result       string     `gorm:"type:jsonb" json:"result,omitempty"`
	QueryID      string     `gorm:"size:64;index" json:"query_id"`
	TraceID      string     `gorm:"size:64" json:"trace_id"`
	Phase        string     `gorm:"size:32;default:auto" json:"phase"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

func (t *StockAnalysisTask) BeforeCreate(tx *gorm.DB) error {
	return t.BaseModel.BeforeCreate(tx)
}

type StockAnalysisReport struct {
	BaseModel
	UserID          uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	QueryID         string    `gorm:"size:64;index" json:"query_id"`
	StockCode       string    `gorm:"size:32;not null;index" json:"stock_code"`
	StockName       string    `gorm:"size:128" json:"stock_name"`
	ReportType      string    `gorm:"size:32" json:"report_type"`
	SentimentScore  *int      `json:"sentiment_score"`
	OperationAdvice string    `gorm:"size:512" json:"operation_advice"`
	TrendPrediction string    `gorm:"size:512" json:"trend_prediction"`
	AnalysisSummary string    `gorm:"type:text" json:"analysis_summary"`
	IdealBuy        string    `gorm:"size:128" json:"ideal_buy"`
	SecondaryBuy    string    `gorm:"size:128" json:"secondary_buy"`
	StopLoss        string    `gorm:"size:128" json:"stop_loss"`
	TakeProfit      string    `gorm:"size:128" json:"take_profit"`
	CurrentPrice    *float64  `json:"current_price"`
	ChangePct       *float64  `json:"change_pct"`
	ModelUsed       string    `gorm:"size:128" json:"model_used"`
	ReportLanguage  string    `gorm:"size:8;default:zh" json:"report_language"`
	RawResult       string    `gorm:"type:jsonb" json:"raw_result,omitempty"`
	ContextSnapshot string    `gorm:"type:jsonb" json:"context_snapshot,omitempty"`
	NewsContent     string    `gorm:"type:text" json:"news_content,omitempty"`
	Action          string    `gorm:"size:32" json:"action"`
	ActionLabel     string    `gorm:"size:64" json:"action_label"`
	CreatedAt       time.Time `json:"created_at"`
}

func (r *StockAnalysisReport) BeforeCreate(tx *gorm.DB) error {
	return r.BaseModel.BeforeCreate(tx)
}

type StockWatchlist struct {
	BaseModel
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_watchlist_user" json:"user_id"`
	StockCode string    `gorm:"size:32;not null;index:idx_watchlist_user" json:"stock_code"`
	StockName string    `gorm:"size:128" json:"stock_name"`
	Market    string    `gorm:"size:16" json:"market"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PortfolioAccount struct {
	BaseModel
	UserID       uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Broker       string    `gorm:"size:64" json:"broker"`
	Market       string    `gorm:"size:16" json:"market"`
	BaseCurrency string    `gorm:"size:8;default:CNY" json:"base_currency"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PortfolioTrade struct {
	BaseModel
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	AccountID uuid.UUID `gorm:"type:uuid;not null;index" json:"account_id"`
	Symbol    string    `gorm:"size:32;not null" json:"symbol"`
	TradeDate string    `gorm:"size:16;not null" json:"trade_date"`
	Side      string    `gorm:"size:8;not null" json:"side"`
	Quantity  float64   `gorm:"not null" json:"quantity"`
	Price     float64   `gorm:"not null" json:"price"`
	Fee       float64   `gorm:"default:0" json:"fee"`
	Tax       float64   `gorm:"default:0" json:"tax"`
	Market    string    `gorm:"size:16" json:"market"`
	Currency  string    `gorm:"size:8;default:CNY" json:"currency"`
	TradeUID  string    `gorm:"size:64;uniqueIndex" json:"trade_uid"`
	Note      string    `gorm:"size:512" json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type PortfolioCashLedger struct {
	BaseModel
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	AccountID uuid.UUID `gorm:"type:uuid;not null;index" json:"account_id"`
	EventDate string    `gorm:"size:16;not null" json:"event_date"`
	Direction string    `gorm:"size:8;not null" json:"direction"`
	Amount    float64   `gorm:"not null" json:"amount"`
	Currency  string    `gorm:"size:8;default:CNY" json:"currency"`
	Note      string    `gorm:"size:512" json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type PortfolioCorporateAction struct {
	BaseModel
	UserID               uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	AccountID            uuid.UUID `gorm:"type:uuid;not null;index" json:"account_id"`
	Symbol               string    `gorm:"size:32;not null" json:"symbol"`
	EffectiveDate        string    `gorm:"size:16;not null" json:"effective_date"`
	ActionType           string    `gorm:"size:32;not null" json:"action_type"`
	Market               string    `gorm:"size:16" json:"market"`
	Currency             string    `gorm:"size:8;default:CNY" json:"currency"`
	CashDividendPerShare *float64  `json:"cash_dividend_per_share"`
	SplitRatio           *float64  `json:"split_ratio"`
	Note                 string    `gorm:"size:512" json:"note"`
	CreatedAt            time.Time `json:"created_at"`
}

type AgentChatSession struct {
	BaseModel
	UserID       string    `gorm:"size:128;not null;index" json:"user_id"`
	SessionID    string    `gorm:"size:64;not null;uniqueIndex" json:"session_id"`
	Title        string    `gorm:"size:256" json:"title"`
	MessageCount int       `gorm:"default:0" json:"message_count"`
	LastActive   time.Time `json:"last_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type AgentChatMessage struct {
	BaseModel
	SessionID string    `gorm:"size:64;not null;index" json:"session_id"`
	Role      string    `gorm:"size:16;not null" json:"role"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Metadata  string    `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type SystemConfig struct {
	BaseModel
	Key           string    `gorm:"size:128;not null;uniqueIndex" json:"key"`
	Value         string    `gorm:"type:text" json:"value"`
	Description   string    `gorm:"size:512" json:"description"`
	Category      string    `gorm:"size:64" json:"category"`
	IsSecret      bool      `gorm:"default:false" json:"is_secret"`
	ConfigVersion int       `gorm:"default:1" json:"config_version"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type BacktestResult struct {
	BaseModel
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	StockCode      string    `gorm:"size:32;not null;index" json:"stock_code"`
	StockName      string    `gorm:"size:128" json:"stock_name"`
	ReportID       uuid.UUID `gorm:"type:uuid;index" json:"report_id"`
	QueryID        string    `gorm:"size:64;index" json:"query_id"`
	AnalysisDate   string    `gorm:"size:16" json:"analysis_date"`
	Action         string    `gorm:"size:32" json:"action"`
	EntryPrice     float64   `json:"entry_price"`
	EntryDate      string    `gorm:"size:16" json:"entry_date"`
	ExitPrice      float64   `json:"exit_price"`
	ExitDate       string    `gorm:"size:16" json:"exit_date"`
	ReturnPct      float64   `json:"return_pct"`
	MaxDrawdownPct float64   `json:"max_drawdown_pct"`
	HoldDays       int       `json:"hold_days"`
	EvalWindowDays int       `gorm:"default:10" json:"eval_window_days"`
	EngineVersion  string    `gorm:"size:32;default:v1" json:"engine_version"`
	CreatedAt      time.Time `json:"created_at"`
}
