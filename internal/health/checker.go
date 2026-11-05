package health

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/GiaBao0510/go_loadbalance/internal/backend"
	"github.com/GiaBao0510/go_loadbalance/internal/pool"
)

// HealthChecker sẽ chạy nền để kiểm tra cho tới khi context bị hủy
type HealthChecker interface {
	Run(ctx context.Context)
}

// Checker thực hiện theo tiêu chí Active Health Check, nó sẽ gửi request tới các backend để kiểm tra trạng thái của chúng
type Checker struct {
	pool *pool.Pool			// pool chứa các backend cần kiểm tra
	interval time.Duration	// khoảng thời gian giữa các lần kiểm tra
	path string				// đường dẫn để gửi request kiểm tra
	client *http.Client		// client HTTP để gửi request kiểm tra
	logger *slog.Logger		// logger để ghi log
}

var _ HealthChecker = (*Checker)(nil) // Kiểm tra tị thời điểm biên dịch

// Hàm khởi tạo
func NewChecker(pool *pool.Pool, interval, timeout time.Duration, path string, logger *slog.Logger) *Checker {
	return &Checker {
		pool: pool,
		interval: interval,
		path: path,
		client: &http.Client{Timeout: timeout},
		logger: logger,
	}
}

// Thực hiện kiểm tra định kỳ 1 lần, nếu context bị hủy thì dừng kiểm tra
func(c *Checker) Run(ctx context.Context) {
	
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.checkAll(ctx) // kiểm tra ngay lần đầu tiên
	for {
		select {
		case <- ctx.Done():
			return
		case <- ticker.C:
			c.checkAll(ctx) // kiểm tra tất cả các backend
		}
	}

	// Sử dụng time.Ticker kết hợp với Select với ctx.Done() là khuôn mẫu chuẩn cho vòng lặp có thể dừng êm
}

// checkAll sẽ kiểm tra tất cả backend song song: nếu một backend treo sẽ không làm chậm các backend còn lại 
func (c *Checker) checkAll(ctx context.Context) {
	
	var wg sync.WaitGroup	// WaitGroup đảm bảo hai vòng kiểm tra không bị chồng chéo lên nhau
	for _, b := range c.pool.Snapshot() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.checkOne(ctx, b) // kiểm tra từng backend
		}()
	}

	wg.Wait() // đợi tất cả các backend kiểm tra xong
}

// checkOne sẽ kiểm tra một backend, nếu trạng thái thay đổi thì ghi log và cập nhật trạng thái
func(c *Checker) checkOne(ctx context.Context, b backend.BackEnd) {
	target := *b.URL() // sao chép giá trị để không sửa URL gốc
	target.Path = c.path // thay đổi đường dẫn để kiểm tra
	
	alive := c.probe(ctx, target.String()) 
	if b.IsAlive() != alive {
		c.logger.Info("backend status changed", "backend", b, "alive", alive)
	}

	b.SetAlive(alive) // cập nhật trạng thái backend
}

// probe trả true nếu backend phản hồi mã 2xx trước hạn timeout.
func(c *Checker) probe(ctx context.Context, target string) bool {
	
	// tạo request với context để có thể hủy nếu timeout
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false
	}

	// gửi request và kiểm tra phản hồi
	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body) // đọc hết body để tái sử dụng kết nối
	return resp.StatusCode >= 200 && resp.StatusCode < 300 // trả true nếu mã phản hồi là 2xx
}