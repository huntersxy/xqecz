package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "xqecz-worker/proto"
	"xqecz-worker/config"
	"xqecz-worker/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()

	port := cfg.Worker.Port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second, // 客户端两次 ping 最小间隔
			PermitWithoutStream: true,              // 允许无活跃流的 ping
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,  // 空闲连接 5 分钟后关闭
			MaxConnectionAge:      30 * time.Minute, // 连接最大存活 30 分钟
			MaxConnectionAgeGrace: 10 * time.Second, // 关闭前宽限期
			Time:                  2 * time.Minute,  // 2 分钟没收到 ping 就发 ping
			Timeout:               20 * time.Second, // ping 超时
		}),
	)
	workerServer := server.NewWorkerServer(cfg)
	pb.RegisterWorkerServiceServer(s, workerServer)

	reflection.Register(s)

	// worker 是无状态计算/文件处理服务，不持有 cron 调度（推荐刷新由
	// NestJS 侧按自己的节奏触发 RefreshRecommend）。主线程由 gRPC Serve 阻塞。
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down worker...")
		s.GracefulStop()
	}()

	log.Printf("[worker] gRPC server listening on :%s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
