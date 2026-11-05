package balancer

import (
	"log/slog"
	"net/http"

	"github.com/GiaBao0510/go_loadbalance/internal/backend"
	"github.com/GiaBao0510/go_loadbalance/internal/pool"
	"github.com/GiaBao0510/go_loadbalance/internal/strategy"
)

// LoadBalancer là http.Handler để quản lý tập backend
type LoadBalancer interface {
	http.Handler
	AddBackend(b backend.BackEnd)
}

// Balancer là cài đặt mặc định của LoadBalancer
type Balancer struct {
	pool *pool.Pool
	strategy strategy.Strategy
	maxAttempts int
	logger *slog.Logger
}

//var _ LoadBalancer = (*Balancer)(nil) // Kiểm tra tị thời điểm biên dịch

// Hàm khởi tạo
func New(
	p *pool.Pool,
	s strategy.Strategy,
	maxAttempts int,
	logger *slog.Logger,
) *Balancer {
	return &Balancer{
		pool: p,
		strategy: s,
		maxAttempts: maxAttempts,
		logger: logger,
	}
}

// AddBackend thêm backend vào pool
func(lb *Balancer) AddBackend(b backend.BackEnd) {lb.pool.Add(b)}

// ServerHTTP chọn backend để chuyển tiếp request và tự chuyển sang backend khác khi backend hiện tại không phản hồi
func(lb *Balancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	
	attempts := lb.maxAttempts
	if !isRetryable(r) {
		attempts = 1 // không retryable thì chỉ thử 1 lần
	}
	tried := make(map[string]struct{}, attempts) // 1. Danh sách tried: bảo đảm một request không bị gửi lại cho cùng một backend vừa lỗi

	// 2. giới hạn số lần thử, tránh khuếch đại tải (retry storm) khi cả cụm đang gặp sự cố.
	for range attempts {
		be := lb.strategy.Next(lb.pool.Healthy(tried))	// Lấy danh sách backend còn sống và chưa thử
		if be == nil {
			break // không còn backend nào để thử
		}

		key := be.URL().String()
		tried[key] = struct{}{} // đánh dấu backend đã thử

		err := be.Forward(w, r)
		if err == nil {
			return // request đã được xử lý thành công
		}
		if r.Context().Err() != nil {
			return // client đã hủy request, không cần retry
		}

		// Passive health check: đánh dấu chết ngay, Active check sẽ cứu sống lại khi hồi phục.
		be.SetAlive(false)
		lb.logger.Warn("backend failed, marked unhealthy", "backend", key, "error", err)
	}

	if len(tried) == 0 {
		// khi không có backend nào sống để thử
		http.Error(w, "no healthy backend available", http.StatusServiceUnavailable)
		return
	}

	//  khi đã thử mà tất cả đều lỗi.
	http.Error(w, "all attempted backends failed", http.StatusBadGateway)
}

// isRetryable chỉ cho phép thử lại request nếu method là GET, HEAD hoặc OPTIONS và không có body.
//  Body của request chỉ đọc được 1 lần nên khi gửi đi thì không thể phát lại
func isRetryable(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return r.Body == nil || r.Body == http.NoBody 		// chỉ retryable nếu không có body
	default:
		return false
	}
}
