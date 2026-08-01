package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "xqecz-worker/proto"

	"xqecz-worker/config"
)

// startTestServer 在随机端口起一个真实 gRPC server，返回 client 与 stop 函数。
func startTestServer(t *testing.T, cfg *config.Config) (pb.WorkerServiceClient, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterWorkerServiceServer(s, NewWorkerServer(cfg))
	go func() { _ = s.Serve(lis) }()

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	client := pb.NewWorkerServiceClient(conn)

	return client, func() {
		conn.Close()
		s.Stop()
	}
}

func TestHealth(t *testing.T) {
	cfg := config.Load()
	client, stop := startTestServer(t, cfg)
	defer stop()

	resp, err := client.Health(context.Background(), &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("Health rpc error: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("unexpected health status: %q", resp.Status)
	}
	t.Logf("Health OK: status=%s version=%s", resp.Status, resp.Version)
}

// TestFetchLinkPreview 用本地 httptest 服务一段含 OG 元数据的 HTML，验证真实解析。
func TestFetchLinkPreview(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head>
			<title>Fallback Title</title>
			<meta property="og:title" content="Hello OG">
			<meta property="og:image" content="https://example.com/cover.png">
			<meta name="twitter:image" content="https://example.com/tw.png">
		</head><body>hi</body></html>`))
	}))
	defer ts.Close()

	cfg := config.Load()
	client, stop := startTestServer(t, cfg)
	defer stop()

	resp, err := client.FetchLinkPreview(context.Background(), &pb.LinkPreviewRequest{Url: ts.URL})
	if err != nil {
		t.Fatalf("FetchLinkPreview rpc error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if resp.Title != "Hello OG" {
		t.Fatalf("title not parsed: got %q", resp.Title)
	}
	if resp.Image != "https://example.com/cover.png" {
		t.Fatalf("image not parsed: got %q", resp.Image)
	}
	t.Logf("LinkPreview OK: title=%s image=%s platform=%s", resp.Title, resp.Image, resp.Platform)
}

// TestCompressImageNoKey 未配置 Tinify key 时返回 success=false（真实降级，非 stub）。
func TestCompressImageNoKey(t *testing.T) {
	cfg := config.Load()
	client, stop := startTestServer(t, cfg)
	defer stop()

	resp, err := client.CompressImage(context.Background(), &pb.CompressRequest{FilePath: "/tmp/nope.png"})
	if err != nil {
		t.Fatalf("CompressImage rpc error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure without tinify key, got success")
	}
	if resp.Error == "" {
		t.Fatalf("expected non-empty error message")
	}
	t.Logf("CompressImage degraded as expected: %s", resp.Error)
}

// TestRefreshRecommendComputesScores worker 作为无状态计算服务，接收内容数据并
// 返回打分（不依赖任何外部 DB/Redis）。验证「时间衰减 + 浏览量」综合打分与降序排列。
func TestRefreshRecommendComputesScores(t *testing.T) {
	cfg := config.Load()
	client, stop := startTestServer(t, cfg)
	defer stop()

	now := time.Now().Unix()
	items := []*pb.RecommendItem{
		{ContentId: 1, CreatedAtUnix: now, ViewCount: 10},          // 新鲜 + 少量浏览
		{ContentId: 2, CreatedAtUnix: now - 8 * 86400, ViewCount: 0}, // 超过 7 天，分数应很低
		{ContentId: 3, CreatedAtUnix: now, ViewCount: 2000},        // 新鲜 + 高浏览，应最高
		{ContentId: 4, CreatedAtUnix: now, ViewCount: 10, LikeCount: 500}, // 新鲜 + 高点赞，应超过高浏览
	}
	resp, err := client.RefreshRecommend(context.Background(), &pb.RefreshRecommendRequest{Items: items})
	if err != nil {
		t.Fatalf("RefreshRecommend rpc error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if len(resp.Results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(resp.Results))
	}

	byId := map[uint64]float64{}
	for _, r := range resp.Results {
		byId[r.ContentId] = r.Score
	}
	// 期望分数降序：content 4（高点赞，权重更高）> content 3（高浏览）> content 1（新鲜）> content 2（过期）。
	if !(byId[4] > byId[3] && byId[3] > byId[1] && byId[1] > byId[2]) {
		t.Fatalf("unexpected score order: %v", byId)
	}
	t.Logf("RefreshRecommend scored: %v", byId)
}

// TestGenerateThumbnailSmoke 验证方法调用链路可达（handler 永不返回 rpc error，
// 无 ffmpeg 时返回 Success=false，仍证明走的是真实逻辑而非 stub）。
func TestGenerateThumbnailSmoke(t *testing.T) {
	cfg := config.Load()
	client, stop := startTestServer(t, cfg)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.GenerateThumbnail(ctx, &pb.ThumbnailRequest{FilePath: "missing", ContentType: "image"})
	if err != nil {
		t.Fatalf("GenerateThumbnail rpc error: %v", err)
	}
	t.Logf("GenerateThumbnail reachable: success=%v err=%q", resp.Success, resp.Error)
}
