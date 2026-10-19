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

// diagnosticMaxRunes bounds what is persisted in the task snapshot. Snapshots
// are longtext, but the admin list ships every diagnostic on a page, so keep
// each one small.
const diagnosticMaxRunes = 900

// diagnosticSecrets masks credential-looking fragments before a provider
// failure is persisted. Order matters: header forms first, then generic token
// shapes, then assignment/query forms.
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

// redactDiagnostic removes credentials from a provider failure report and caps
// its total length. It never sees the decrypted provider key: the snapshot
// carries CipherKey, not the plaintext.
func redactDiagnostic(text string) string {
	for _, rule := range diagnosticSecrets {
		text = rule.pattern.ReplaceAllString(text, rule.replace)
	}
	runes := []rune(text)
	if len(runes) <= diagnosticMaxRunes {
		return text
	}
	// Redaction runs first, so a truncated tail can never expose a partial
	// secret that the rules would otherwise have masked.
	return string(runes[:diagnosticMaxRunes-1]) + "…"
}

// failureCategory labels the failure for administrators without changing the
// user-visible wording.
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

// unknownFailureMessage keeps the polite wording but adds the task number so a
// user and an administrator can correlate the message with the task list. The
// task stays in the unknown state and its quota reservation is untouched.
func unknownFailureMessage(taskID uint64, provider ProviderConfig, err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("供应商请求超时（%d 秒未返回），结果不明；额度仍预留（任务 #%d）", provider.TimeoutSeconds, taskID)
	case errors.Is(err, context.Canceled):
		return fmt.Sprintf("请求已中断，结果不明；额度仍预留（任务 #%d）", taskID)
	default:
		return fmt.Sprintf("供应商结果不明；额度仍预留，请联系管理员核查（任务 #%d）", taskID)
	}
}

// failureDiagnostic builds the admin-only report stored inside the task
// snapshot. It carries everything an operator needs to investigate: when,
// which provider/model, how long the call took, the error category, the HTTP
// status, the provider response prefix and the full error chain.
func failureDiagnostic(taskID uint64, provider ProviderConfig, err error, elapsed time.Duration, at time.Time) string {
	statusCode, snippet, message := 0, "", ""
	var response *ProviderResponseError
	if errors.As(err, &response) {
		statusCode, snippet = response.StatusCode, response.Snippet
	}
	if err != nil {
		message = err.Error()
	}
	lines := []string{
		"time: " + at.UTC().Format(time.RFC3339),
		fmt.Sprintf("task: #%d", taskID),
		fmt.Sprintf("provider: #%d %s", provider.ID, provider.Name),
		"model: " + provider.Model,
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

// logUnknownFailure makes every unknown outcome visible in the admin runtime
// log instead of only in the fixed user-facing sentence.
func (s *Service) logUnknownFailure(taskID uint64, provider ProviderConfig, elapsed time.Duration, err error) {
	slog.Error("AI generation ended with unknown provider result",
		"task_id", taskID,
		"provider_id", provider.ID,
		"model", provider.Model,
		"elapsed_ms", elapsed.Milliseconds(),
		"status", "unknown",
		"err", err,
	)
}

// persistDiagnostic writes the redacted report back into the task snapshot
// with a separate UPDATE, preserving the original snapshot contents. A failed
// write must not undo the task settlement, so problems are only logged.
func (s *Service) persistDiagnostic(taskID uint64, rawSnapshot, diagnostic string) {
	if s == nil || s.db == nil || strings.TrimSpace(diagnostic) == "" {
		return
	}
	var snapshot taskSnapshot
	if err := json.Unmarshal([]byte(rawSnapshot), &snapshot); err != nil {
		slog.Warn("AI task diagnostic snapshot unreadable", "task_id", taskID)
		return
	}
	snapshot.Diagnostic = diagnostic
	raw, err := json.Marshal(snapshot)
	if err != nil {
		slog.Warn("AI task diagnostic snapshot encode failed", "task_id", taskID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.db.WithContext(ctx).Model(&Task{}).Where("id = ?", taskID).Update("snapshot", string(raw)).Error; err != nil {
		slog.Warn("AI task diagnostic snapshot save failed", "task_id", taskID)
	}
}
