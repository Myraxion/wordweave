package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Myraxion/wordweave/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dataDir := flag.String("data-dir", os.Getenv("WORDWEAVE_DATA_DIR"), "数据目录（优先于 WORDWEAVE_DATA_DIR；默认程序旁的 data）")
	address := flag.String("listen", "127.0.0.1:8080", "HTTP 监听地址")
	flag.Parse()
	if *dataDir == "" {
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		*dataDir = filepath.Join(filepath.Dir(executable), "data")
	}
	absDir, err := filepath.Abs(*dataDir)
	if err != nil {
		return err
	}
	a, err := app.Open(absDir)
	if err != nil {
		return err
	}
	defer a.Close()
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: a, ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()
	fmt.Printf("Wordweave: http://%s\n数据目录: %s\n按 Ctrl+C 停止程序。\n", listener.Addr(), absDir)
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		<-shutdownDone
		return nil
	}
	return err
}
