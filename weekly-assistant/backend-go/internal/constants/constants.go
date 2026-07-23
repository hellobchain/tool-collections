package constants

// ext
const (
	DocxExt = ".docx"
	PdfExt  = ".pdf"
	DocExt  = ".doc"
	HtmlExt = ".html"
	HtmExt  = ".htm"
	MdExt   = ".md"
	TxtExt  = ".txt"
	PngExt  = ".png"
	JpgExt  = ".jpg"
	JpegExt = ".jpeg"
)

const (
	LOCAL_SAVE_TYPE      = "local"
	OSS_S3_SAVE_TYPE     = "oss_s3"
	OSS_MINIO_SAVE_TYPE  = "oss_minio"
	OSS_ALIYUN_SAVE_TYPE = "oss_aliyun"
	MAX_FILE_SIZE        = 20 * 1024 * 1024
)

// 合同审查状态
const (
	ContractReviewStatusPending   = "pending"   // 待审查
	ContractReviewStatusRunning   = "running"   // 审查中
	ContractReviewStatusCompleted = "completed" // 审查完成
	ContractReviewStatusRejected  = "rejected"  // 审查被拒绝
	ContractReviewStatusCanceled  = "canceled"  // 审查取消
	ContractReviewStatusFailed    = "failed"    // 审查失败
	ContractDraftStatusParsing    = "parsing"   // 文档解析中
	ContractDraftStatusParsed     = "parsed"    // 文档解析完成
)

// 合同审查项状态
const (
	ContractReviewItemStatusOpen   = "open"   // 待处理
	ContractReviewItemStatusIgnore = "ignore" // 已忽略
	ContractReviewItemStatusFixed  = "fixed"  // 已修复
	ContractReviewItemStatusDone   = "done"   // 已处理
)

// 合同审查项状态描述
const (
	ContractReviewItemStatusOpenDesc   = "公开"
	ContractReviewItemStatusIgnoreDesc = "已忽略"
	ContractReviewItemStatusFixedDesc  = "已修复"
	ContractReviewItemStatusDoneDesc   = "已处理"
)

// 风险级别
const (
	RiskLevelHigh   = "high"   // 高风险
	RiskLevelMedium = "medium" // 中风险
	RiskLevelLow    = "low"    // 低风险
)

// 风险级别描述
const (
	RiskLevelHighDesc   = "高风险"
	RiskLevelMediumDesc = "中风险"
	RiskLevelLowDesc    = "低风险"
)

// 合同起草状态
const (
	ContractDraftStatusPending    = "pending"    // 待生成
	ContractDraftStatusGenerating = "generating" // 生成中
	ContractDraftStatusCompleted  = "completed"  // 生成完成
	ContractDraftStatusRejected   = "rejected"   // 生成被拒绝
	ContractDraftStatusCanceled   = "canceled"   // 生成取消
	ContractDraftStatusFailed     = "failed"     // 生成失败
)

// 合同起草状态描述
const (
	ContractDraftStatusPendingDesc    = "待生成"
	ContractDraftStatusGeneratingDesc = "生成中"
	ContractDraftStatusCompletedDesc  = "生成完成"
	ContractDraftStatusRejectedDesc   = "生成被拒绝"
	ContractDraftStatusCanceledDesc   = "生成取消"
	ContractDraftStatusFailedDesc     = "生成失败"
)

// 合同要素提取状态
const (
	ContractExtractStatusPending    = "pending"    // 待提取
	ContractExtractStatusExtracting = "extracting" // 提取中
	ContractExtractStatusCompleted  = "completed"  // 提取完成
	ContractExtractStatusRejected   = "rejected"   // 提取被拒绝
	ContractExtractStatusCanceled   = "canceled"   // 提取取消
	ContractExtractStatusFailed     = "failed"     // 提取失败
)

// 合同要素提取状态描述
const (
	ContractExtractStatusPendingDesc    = "待提取"
	ContractExtractStatusExtractingDesc = "提取中"
	ContractExtractStatusCompletedDesc  = "提取完成"
	ContractExtractStatusRejectedDesc   = "提取被拒绝"
	ContractExtractStatusCanceledDesc   = "提取取消"
	ContractExtractStatusFailedDesc     = "提取失败"
)

// ============ 分页 ============
const (
	DefaultPage      = 1
	DefaultPageSize  = 1
	FragmentPageSize = 50
	FragmentMaxSize  = 200
	HistoryPageSize  = 20
	HistoryMaxSize   = 200
)

// ============ 日期格式 ============
const (
	DateFormatDate       = "2006-01-02"
	DateFormatDateHHMM   = "2006-01-02 15:04"
	DateFormatTimeHHMMSS = "2006-01-02 15:04:05"
)

// 合同比对
const (
	JsonCompareTypeAdded   = "added"
	JsonCompareTypeRemoved = "removed"
	JsonCompareTypeChanged = "changed"
)

// 提示词类型
const (
	PromptTypeWeekly  = "weekly"  // 周报
	PromptTypeQuarter = "quarter" // 季度
	PromptTypeYearly  = "year"    // 年度
)

// ============ LLM ============
const (
	LLMTemperature     = 0.3
	LLMExtractTemp     = 0.1
	LLMMaxFallbackFrag = 5
	LLMMaxExtractItems = 3
)

// ============ JWT ============
const (
	JWTDefaultTTL = 7  // days for access token
	JWTRefreshTTL = 30 // days for refresh token
)

// ============ GitLab ============
const (
	GitLabPerPage = 1000
)

// ============ SSE ============
const (
	SSEDoneMarker = "[DONE]"
)

// ============ Prompt 模板 ============
const (
	DefaultPromptSortOrder = 100
	DefaultSkillSortOrder  = 0
)

// 默认提示词
const (
	// 下周计划系统提示词
	DefaultNextWeekPlanSystemPrompt = `你是一个周报助手。从用户提供的周报内容中，提取"下周计划"部分的具体待办事项。`
	DefaultNextWeekPlanUserPrompt   = `请从以下周报内容中提取下周的待办事项（最多3条）。只输出JSON数组，不要任何其他文字。每条包含一个字段"content"。

周报内容：
%s

输出格式示例：
[{"content":"事项1"},{"content":"事项2"}]`

	// Summary
	DefaultSummarySystemPrompt = `你是一个专业的周报汇总助手，擅长对多篇周报进行归纳总结。请根据以下%s的周报内容，生成一份结构化的阶段总结。`
	DefaultSummaryUserPrompt   = `请根据以下%s的周报内容，生成一份总结报告。

周报内容：
%s

请按照以下格式输出：

## %s工作总结

### 一、关键产出与成果
（列出最重要的3-5项产出，标注所属周）

### 二、重点工作方向
（归纳持续投入的主要方向）

### 三、能力成长与经验
（技术提升、踩坑经验等）

### 四、存在的问题与风险
（从各周风险中提炼共性问题）

### 五、下阶段规划
（基于各周计划的汇总建议）`

	// 周报草稿系统提示词
	DefaultWeeklyDraftSystemPrompt = `你是一个专业的周报撰写助手，擅长把零散信息组织成结构化、有重点的工作汇报。`
	DefaultWeeklyDraftUserPrompt   = `请根据以下碎片和继承事项，生成一份周报。输出必须包含三个平行板块，每个板块用\"### \"开头。\n\n碎片列表：{fragments}\n继承事项：{carryover}\n\n输出结构如下：\n### 攻坚\n（从\"攻克、突破、解决\"等角度组织本周的关键产出，每条用\"- \"开头）\n\n### 协作\n（从\"推动、对齐、拉通\"等角度组织常规事项与协作工作，每条用\"- \"开头）\n\n### 稳健\n（从\"保障、规范、优化\"等角度组织风险控制与稳定性工作，每条用\"- \"开头）\n\n要求：\n1. 每个板块至少2-3条\n2. 语言专业、客观，每条包含动作+对象+结果\n3. 禁止出现\"可能\"\"大概\"\"似乎\"等弱化词`
)
