package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/MeteorsLiu/FastPaimon/httpServer"
)

func main() {
	ctx, cancel := context.WithCancel()
	defer cancel()
	go httpServer.New(ctx, "0.0.0.0:8081")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
