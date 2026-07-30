package server

import (
	"context"
	"log/slog"

	pb "xqecz-worker/proto"

	"xqecz-worker/config"
	"xqecz-worker/linkpreview"
	"xqecz-worker/media"
)

// WorkerServer implements the gRPC WorkerService.
// 文件处理 / 推荐刷新逻辑在阶段 2 从 xqecz-golang / xqecz-nodejs 迁移而来。
type WorkerServer struct {
	pb.UnimplementedWorkerServiceServer
	cfg *config.Config
}

func NewWorkerServer(cfg *config.Config) *WorkerServer {
	return &WorkerServer{cfg: cfg}
}

// Health returns the worker status.
func (s *WorkerServer) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "ok", Version: "0.2.0"}, nil
}

// GenerateThumbnail 用 ffmpeg 为图片/视频生成 800px 宽 webp 缩略图。
func (s *WorkerServer) GenerateThumbnail(ctx context.Context, req *pb.ThumbnailRequest) (*pb.ThumbnailResponse, error) {
	slog.Info("GenerateThumbnail", "file", req.FilePath, "type", req.ContentType)
	thumbPath, err := media.GenerateThumbnail(ctx, req.FilePath, req.ContentType, s.cfg.Server.ThumbDir)
	if err != nil {
		return &pb.ThumbnailResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.ThumbnailResponse{ThumbPath: thumbPath, Success: true}, nil
}

// CompressImage 用 Tinify API 压缩图片，输出 webp 到 images 目录。
func (s *WorkerServer) CompressImage(ctx context.Context, req *pb.CompressRequest) (*pb.CompressResponse, error) {
	slog.Info("CompressImage", "file", req.FilePath)
	compressed, err := media.TinifyCompress(req.FilePath, s.cfg.Server.ImagesDir, s.cfg.Tinify.APIKey)
	if err != nil {
		return &pb.CompressResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.CompressResponse{CompressedPath: compressed, Success: true}, nil
}

// FetchLinkPreview 抓取外部链接的 OG/Twitter Card 元数据。
func (s *WorkerServer) FetchLinkPreview(ctx context.Context, req *pb.LinkPreviewRequest) (*pb.LinkPreviewResponse, error) {
	slog.Info("FetchLinkPreview", "url", req.Url)
	res, ok := linkpreview.Fetch(ctx, req.Url)
	if !ok {
		return &pb.LinkPreviewResponse{Success: false}, nil
	}
	return &pb.LinkPreviewResponse{
		Title:    res.Title,
		Image:    res.Image,
		Platform: res.Platform,
		Success:  true,
	}, nil
}

// RefreshRecommend 作为无状态计算服务：接收由 NestJS 传入的内容数据，
// 纯计算打分后将结果返回（不访问任何 DB/Redis，架构约束见 AGENTS.md）。
// NestJS 拿到结果后负责写入 Redis ZSet。
func (s *WorkerServer) RefreshRecommend(_ context.Context, req *pb.RefreshRecommendRequest) (*pb.RefreshRecommendResponse, error) {
	results := computeRecommend(req.GetItems())
	return &pb.RefreshRecommendResponse{Success: true, Results: results}, nil
}
