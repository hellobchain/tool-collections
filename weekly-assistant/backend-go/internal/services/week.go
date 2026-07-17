package services

import (
	"fmt"
	"time"
)

// GetWeekStart 获取本周一日期（始终返回 Local 时区 00:00:00）
func GetWeekStart(t time.Time) time.Time {
	t = t.In(time.Local)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	offset := weekday - 1
	return time.Date(t.Year(), t.Month(), t.Day()-offset, 0, 0, 0, 0, time.Local)
}

// FormatDate 格式化日期为 YYYY-MM-DD
func FormatDate(t time.Time) string {
	return t.In(time.Local).Format("2006-01-02")
}

// ParseDate 解析日期字符串
func ParseDate(layout, s string) (time.Time, error) {
	return time.ParseInLocation(layout, s, time.Local)
}

// GetISOWeekNumber 获取ISO周数（如 "2025年第3周"）
func GetISOWeekNumber(t time.Time) string {
	year, week := t.In(time.Local).ISOWeek()
	return fmt.Sprintf("%d年第%d周", year, week)
}

// GetWeekEnd 获取周日日期
func GetWeekEnd(t time.Time) time.Time {
	return t.AddDate(0, 0, 6)
}
