package aigen

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestRedactDiagnostic(t *testing.T) {
	raw := strings.Join([]string{
		"error: provider said apikey=super-secret-1",
		"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.token-value",
		"x-goog-api-key: gemini-secret-2",
		"url https://example.com/result.png?key=query-secret-3&token=query-secret-4&api_key=query-secret-5&signature=query-secret-6&keep=ok",
		"saw sk-liveABCDEFGH1234 in the body",
	}, "\n")
	got := redactDiagnostic(raw)
	for _, secret := range []string{
		"super-secret-1",
		"eyJhbGciOiJIUzI1NiJ9.token-value",
		"gemini-secret-2",
		"query-secret-3",
		"query-secret-4",
		"query-secret-5",
		"query-secret-6",
		"sk-liveABCDEFGH1234",
	} {
		if strings.Contains(got, secret) {
			t.Fatalf("diagnostic leaked %q:\n%s", secret, got)
		}
	}
	if !strings.Contains(got, "keep=ok") {
		t.Fatalf("non-secret query parameter was dropped: %s", got)
	}
	if !strings.Contains(got, "***") {
		t.Fatalf("redaction placeholder missing: %s", got)
	}
}

func TestRedactDiagnosticTruncates(t *testing.T) {
	got := redactDiagnostic(strings.Repeat("中", diagnosticMaxRunes+500))
	if n := len([]rune(got)); n != diagnosticMaxRunes {
		t.Fatalf("diagnostic length = %d, want %d", n, diagnosticMaxRunes)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatal("truncated diagnostic should end with an ellipsis")
	}
}

func TestReadResponseSnippetIsBounded(t *testing.T) {
	if got := readResponseSnippet(nil); got != "" {
		t.Fatalf("nil body snippet = %q", got)
	}
	long := strings.Repeat("x", 2000)
	if got := readResponseSnippet(strings.NewReader(long)); len(got) != providerFailureSnippetLimit {
		t.Fatalf("snippet length = %d, want %d", len(got), providerFailureSnippetLimit)
	}
}

func TestProvidersExposeHTTPStatusAndSnippetOnUnknown(t *testing.T) {
	const body = `{"error":{"message":"upstream exploded"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()
	cfg := ProviderConfig{ID: 7, Name: "test", Endpoint: server.URL + "/generate", Model: "m", TimeoutSeconds: 10}
	input := Input{Prompt: "p", Count: 1}
	for _, tc := range []struct {
		name     string
		provider Provider
	}{
		{"openai", OpenAIProvider{Client: server.Client()}},
		{"openai_chat", OpenAIChatProvider{Client: server.Client()}},
		{"gemini", GeminiProvider{Client: server.Client()}},
		{"codex", CodexResponsesProvider{Client: server.Client()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.provider.Generate(context.Background(), cfg, "secret-key", input, nil)
			if err == nil {
				t.Fatal("expected a failure")
			}
			var response *ProviderResponseError
			if !errors.As(err, &response) {
				t.Fatalf("error %T does not carry the HTTP status: %v", err, err)
			}
			if response.StatusCode != http.StatusBadGateway {
				t.Fatalf("status = %d", response.StatusCode)
			}
			if !strings.Contains(response.Snippet, "upstream exploded") {
				t.Fatalf("snippet = %q", response.Snippet)
			}
			if strings.Contains(err.Error(), "upstream exploded") {
				t.Fatalf("response body leaked into Error(): %v", err)
			}
			var known KnownFailure
			if errors.As(err, &known) {
				t.Fatal("a 5xx response must stay an unknown outcome, not a known failure")
			}
		})
	}
}

func quietSlog(t *testing.T) {
	t.Helper()
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
}

func TestFailureWordingDiagnosticAndRefund(t *testing.T) {
	quietSlog(t)
	for _, tc := range []struct {
		name         string
		err          error
		wording      string
		category     string
		httpStatus   string
		leakedSecret string
	}{
		{"timeout", context.DeadlineExceeded, "生成失败：供应商请求超时（30 分钟未返回）", "deadline_exceeded", "http_status: 0", ""},
		{"canceled", context.Canceled, "生成失败：请求已中断", "canceled", "http_status: 0", ""},
		{"http_5xx", &ProviderResponseError{StatusCode: 503, Snippet: `{"error":"upstream unavailable","api_key":"sk-liveABCDEFGH1234"}`}, "生成失败：供应商返回 HTTP 503", "provider_http", "http_status: 503", "sk-liveABCDEFGH1234"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, fake, p, session := testService(t)
			grant(t, s, 1, 10)
			task, err := s.Submit(context.Background(), 1, inputFor(p, session, "failed-key-"+tc.name))
			if err != nil {
				t.Fatal(err)
			}
			fake.err = tc.err
			fake.failures.Store(generationAttempts)
			s.runOne(context.Background())
			if err = s.db.First(&task, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			if task.Status != "failed" {
				t.Fatalf("status = %q, want failed", task.Status)
			}
			if task.Attempts != generationAttempts {
				t.Fatalf("attempts = %d", task.Attempts)
			}
			if !strings.Contains(task.Error, tc.wording) {
				t.Fatalf("user message = %q, want substring %q", task.Error, tc.wording)
			}
			if !strings.Contains(task.Error, "额度已返还") && !strings.Contains(task.Error, "本次未扣除额度") {
				t.Fatalf("user message lacks the refund note: %q", task.Error)
			}
			if a := balance(t, s, 1); a.Available != 10 || a.Reserved != 0 {
				t.Fatalf("quota not refunded: %+v", a)
			}
			var snapshot taskSnapshot
			if err = json.Unmarshal([]byte(task.Snapshot), &snapshot); err != nil {
				t.Fatal(err)
			}
			if snapshot.Diagnostic == "" {
				t.Fatal("diagnostic was not persisted into the snapshot")
			}
			for _, want := range []string{
				"task: #" + strconv.FormatUint(task.ID, 10),
				"attempt: 1/4",
				"attempt: 4/4",
				"provider: #" + strconv.FormatUint(p.ID, 10),
				"model: documented-model",
				"timeout: 30m0s",
				"category: " + tc.category,
				"elapsed_ms: ",
				tc.httpStatus,
				"error: ",
			} {
				if !strings.Contains(snapshot.Diagnostic, want) {
					t.Fatalf("diagnostic lacks %q:\n%s", want, snapshot.Diagnostic)
				}
			}
			if tc.leakedSecret != "" && strings.Contains(snapshot.Diagnostic, tc.leakedSecret) {
				t.Fatalf("diagnostic leaked the provider secret:\n%s", snapshot.Diagnostic)
			}
			if len([]rune(snapshot.Diagnostic)) > diagnosticMaxRunes {
				t.Fatalf("diagnostic is too long: %d", len([]rune(snapshot.Diagnostic)))
			}
			if snapshot.Input.Prompt != "A pink circle" || snapshot.Provider.Name != "test" || snapshot.CipherKey == "" || snapshot.Input.Key == "" {
				t.Fatalf("diagnostic update corrupted the snapshot: %+v", snapshot)
			}
		})
	}
}

func TestKnownFailureRecordsDiagnosticAndRefunds(t *testing.T) {
	quietSlog(t)
	s, fake, p, session := testService(t)
	grant(t, s, 1, 10)
	task, err := s.Submit(context.Background(), 1, inputFor(p, session, "known-key-000001"))
	if err != nil {
		t.Fatal(err)
	}
	fake.err = KnownFailure{"provider rejected the request"}
	fake.failures.Store(generationAttempts)
	s.runOne(context.Background())
	if err = s.db.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if task.Status != "failed" {
		t.Fatalf("status = %q", task.Status)
	}
	var snapshot taskSnapshot
	if err = json.Unmarshal([]byte(task.Snapshot), &snapshot); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"attempt: 4/4", "category: provider_error", "error: provider rejected the request"} {
		if !strings.Contains(snapshot.Diagnostic, want) {
			t.Fatalf("diagnostic lacks %q:\n%s", want, snapshot.Diagnostic)
		}
	}
	if a := balance(t, s, 1); a.Available != 10 || a.Reserved != 0 {
		t.Fatalf("known failure should release the reservation: %+v", a)
	}
}

func TestHTTPFailureRecordsEveryAttempt(t *testing.T) {
	quietSlog(t)
	s, _, p, session := testService(t)
	grant(t, s, 1, 10)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `{"error":{"message":"upstream exploded","token":"sk-liveABCDEFGH1234"}}`)
	}))
	defer server.Close()
	task, err := s.Submit(context.Background(), 1, inputFor(p, session, "diagnostic-key-0001"))
	if err != nil {
		t.Fatal(err)
	}
	var snapshot taskSnapshot
	if err = json.Unmarshal([]byte(task.Snapshot), &snapshot); err != nil {
		t.Fatal(err)
	}
	snapshot.Provider.Endpoint = server.URL + "/v1/images/generations"
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.db.Model(&Task{}).Where("id = ?", task.ID).Update("snapshot", string(raw)).Error; err != nil {
		t.Fatal(err)
	}
	if err = s.db.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	s.provider = OpenAIProvider{Client: server.Client()}
	s.runOne(context.Background())
	if err = s.db.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if task.Status != "failed" {
		t.Fatalf("status = %q, want failed", task.Status)
	}
	if !strings.Contains(task.Error, "生成失败：供应商返回 HTTP 502") {
		t.Fatalf("user message = %q", task.Error)
	}
	if a := balance(t, s, 1); a.Available != 10 || a.Reserved != 0 {
		t.Fatalf("quota not refunded: %+v", a)
	}
	if err = json.Unmarshal([]byte(task.Snapshot), &snapshot); err != nil {
		t.Fatal(err)
	}
	diagnostic := snapshot.Diagnostic
	for _, want := range []string{
		"attempt: 1/4",
		"attempt: 4/4",
		"http_status: 502",
		"category: provider_http",
		"response: ",
		"upstream exploded",
		"provider: #" + strconv.FormatUint(p.ID, 10),
		"model: documented-model",
		"elapsed_ms: ",
		"error: 供应商结果不明（HTTP 502）",
	} {
		if !strings.Contains(diagnostic, want) {
			t.Fatalf("diagnostic lacks %q:\n%s", want, diagnostic)
		}
	}
	if strings.Count(diagnostic, "attempt: ") != generationAttempts {
		t.Fatalf("expected %d attempt records:\n%s", generationAttempts, diagnostic)
	}
	if strings.Contains(diagnostic, "sk-liveABCDEFGH1234") {
		t.Fatalf("diagnostic leaked the response credential:\n%s", diagnostic)
	}
	if strings.Contains(diagnostic, "fake-secret-key") {
		t.Fatalf("diagnostic leaked the decrypted provider key:\n%s", diagnostic)
	}
	if snapshot.Input.Prompt != "A pink circle" || snapshot.Provider.Name != "test" || snapshot.CipherKey == "" {
		t.Fatalf("diagnostic update corrupted the snapshot: %+v", snapshot)
	}
}
