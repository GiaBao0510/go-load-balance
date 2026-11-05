package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/GiaBao0510/go_loadbalance/internal/backend"
	"github.com/GiaBao0510/go_loadbalance/internal/balancer"
	"github.com/GiaBao0510/go_loadbalance/internal/health"
	"github.com/GiaBao0510/go_loadbalance/internal/pool"
	"github.com/GiaBao0510/go_loadbalance/internal/strategy"
)

func main() {
	slog.Info("load balancer started")
	if err := run(); err != nil {
		slog.Error("load balancer stopped", "error", err)
		os.Exit(1)
	}
}

// run chứa toàn bộ logic để main chỉ xử lý lỗi một chỗ.
func run() error {
	var(
		backends = flag.String("backends", "", "danh sách backend, phân tách bằng dấu phẩy: url hoặc url|weight")
		port = flag.Int("port", 3000, "Cổng lắng nghe")
		algo = flag.String("algo", "round-robin", "Thuật toán cân bằng tải: round-robin, least-connection, weight-round-robin")
		interval = flag.Duration("health-interval", 5*time.Second, "Khoảng thời gian giữa các lần kiểm tra sức khỏe của backend")
		path = flag.String("health-path", "/health", "Đường dẫn để kiểm tra sức khỏe của backend")
	)
	flag.Parse()

	if strings.TrimSpace(*backends) == ""{
		return errors.New("flag -backends is required")
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	p := pool.New()
	// Duyệt các backend được cung cấp trong flag -backends, phân tách bằng dấu phẩy.
	for _, spec := range strings.Split(*backends, ",") {
		b, err := parseBackends(strings.TrimSpace(spec))
		if err != nil {
			return fmt.Errorf("invalid backend %q: %w", spec, err)
		}
		p.Add(b)
	}

	// Nhận biết thuật toán áp dụng
	strategy, err := strategy.New(*algo)
	if err != nil {
		return err
	}
	lb := balancer.New(p, strategy, 3 ,logger)
	checker := health.NewChecker(p, *interval, 2*time.Second, *path, logger)

	// CTX bị hủy khi nhận ctr + c hoặc sigterm
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop() 
	go checker.Run(ctx)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", *port),
		Handler: lb,
		ReadTimeout: 5 * time.Second, // chống slowloris
		IdleTimeout: 60 * time.Second,
		// Ở đây không đặt WriteTimeout để không cắt các response dạng streaming
	}

	// Hiển thị danh sách lỗi nếu có
	errCh := make(chan error, 1)
	go func() {errCh <- srv.ListenAndServe()}()
	logger.Info("load balancer started", "addr", srv.Addr, "algo", *algo)

	select {
	case err := <-errCh:
		return err // Lỗi khi khởi động
	case <-ctx.Done():
	}

	// Cho các request đang xử lý tối đa 10 giây để hoàn tất
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// parseBackends đọc "http://host:port" hoặc "http://host:port|weight". 
func parseBackends(spec string) (backend.BackEnd, error) {
	rawURL, weightStr, hasWeight := strings.Cut(spec, "|") // tách lấy weight nếu có
	weight := 1

	// Nếu có trọng số
	if hasWeight {
		w, err := strconv.Atoi(weightStr)
		if err != nil {
			return nil, fmt.Errorf("invalid weight %q: %w", weightStr, err)
		}
		weight = w
	}
	return backend.New(rawURL, weight)
}