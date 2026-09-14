package web

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// signalContext 在收到 SIGINT/SIGTERM 时取消，用于优雅关停。
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
