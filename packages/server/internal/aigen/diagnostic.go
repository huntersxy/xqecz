package aigen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

const diagnosticMaxRunes = 6000

var diagnosticSecrets = []struct {
	pattern *regexp.Regexp
	replace string
}{
	{regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)[^\r\n]+`), "${1}***"},
	{regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=\-]+`), "${1}***"},
	{regexp.MustCompile(`sk-[A-Za-z0-9_\-]{8,}`), "***"},
	{regexp.MustCompile(`(?i)((?:x-goog-api-key|x-api-key|api[_-]?key|access[_-]?token|client[_-]?secret|secret[_-]?key|token|password)\s*[:=]\s*)[^\s,;"'&]+`), "${1}***"},
	{regexp.MustCompile(`(?i)([?&](?:key|token|api_key|apikey|access_token|signature|sig|x-amz-signature|x-amz-credential|x-amz-security-token|auth)=)[^&\s"'<>]+`), "${1}***"},
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit < 1 {
		return ""
	}
	return string(runes[:limit-1]) + "…"
}

func redactDiagnostic(text string) string {
	for _, rule := range diagnosticSecrets {
		text = rule.pattern.ReplaceAllString(text, rule.replace)
	}
	return truncateRunes(text, diagnosticMaxRunes)
}

func appendDiagnostic(existing, block string) string {
	block = strings.TrimSpace(block)
	if block == "" {
		return existing
	}
	if strings.TrimSpace(existing) == "" {
		return truncateRunes(block, diagnosticMaxRunes)
	}
	return truncateRunes(existing+"\n\n"+block, diagnosticMaxRunes)
}

func failureCategory(err error) string {
	var response *ProviderResponseError
	switch {
	case err == nil:
		return "none"
	case errors.As(err, &response):
		return "provider_http"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "provider_error"
	}
}

func failureReason(err error) string {
	var known KnownFailure
	var response *ProviderResponseError
	switch {
	case err == nil:
		return "未知错误"
	case errors.As(err, &known):
		return known.Message
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("供应商请求超时（%d 分钟未返回）", int(generationAttemptTimeout/time.Minute))
	case errors.Is(err, context.Canceled):
		return "请求已中断"
	case errors.As(err, &response):
		return fmt.Sprintf("供应商返回 HTTP %d", response.StatusCode)
	default:
		return truncateRunes(err.Error(), 120)
	}
}

func refundNote(task Task) string {
	if task.Cost > 0 {
		return "额度已返还"
	}
	return "本次未扣除额度"
}

func generationFailureMessage(task Task, attempts int, err error) string {
	if attempts < 1 {
		return truncateRunes(fmt.Sprintf("生成失败：%s；%s（任务 #%d）", failureReason(err), refundNote(task), task.ID), 300)
	}
	return truncateRunes(fmt.Sprintf("生成失败：%s；已尝试 %d 次，%s（任务 #%d）", failureReason(err), attempts, refundNote(task), task.ID), 300)
}

func failureDiagnostic(taskID uint64, attempt int, provider ProviderConfig, err error, elapsed time.Duration, at time.Time) string {
	statusCode, snippet, message := 0, "", ""
	var response *ProviderResponseError
	if errors.As(err, &response) {
		statusCode, snippet = response.StatusCode, response.Snippet
	}
	if err != nil {
		message = err.Error()
	}
	attemptText := "preflight"
	switch {
	case attempt > 0:
		attemptText = fmt.Sprintf("%d/%d", attempt, generationAttempts)
	case attempt < 0:
		attemptText = "interrupted"
	}
	lines := []string{
		"time: " + at.UTC().Format(time.RFC3339),
		fmt.Sprintf("task: #%d", taskID),
		"attempt: " + attemptText,
		fmt.Sprintf("provider: #%d %s", provider.ID, provider.Name),
		"model: " + provider.Model,
		"timeout: " + generationAttemptTimeout.String(),
		"category: " + failureCategory(err),
		fmt.Sprintf("elapsed_ms: %d", elapsed.Milliseconds()),
		fmt.Sprintf("http_status: %d", statusCode),
		"error: " + message,
	}
	if snippet != "" {
		lines = append(lines, "response: "+snippet)
	}
	return redactDiagnostic(strings.Join(lines, "\n"))
}

func (s *Service) logGenerationFailure(taskID uint64, attempt int, provider ProviderConfig, elapsed time.Duration, err error) {
	slog.Error("AI generation attempt failed",
		"task_id", taskID,
		"attempt", attempt,
		"provider_id", provider.ID,
		"model", provider.Model,
		"elapsed_ms", elapsed.Milliseconds(),
		"err", err,
	)
}

func (s *Service) recordFailure(taskID uint64, snapshot *taskSnapshot, provider ProviderConfig, attempt int, elapsed time.Duration, err error) {
	snapshot.Diagnostic = appendDiagnostic(snapshot.Diagnostic, failureDiagnostic(taskID, attempt, provider, err, elapsed, time.Now()))
	s.persistSnapshot(taskID, *snapshot)
	s.logGenerationFailure(taskID, attempt, provider, elapsed, err)
}

func (s *Service) persistSnapshot(taskID uint64, snapshot taskSnapshot) {
	if s == nil || s.db == nil {
		return
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		slog.Warn("AI task snapshot encode failed", "task_id", taskID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.db.WithContext(ctx).Model(&Task{}).Where("id = ?", taskID).Updates(map[string]any{"snapshot": string(raw), "updated_at": time.Now()}).Error; err != nil {
		slog.Warn("AI task snapshot save failed", "task_id", taskID)
	}
}
