package pool

import (
	"github.com/GiaBao0510/go_loadbalance/internal/backend"
	"sync"
)

// Pool: bảo vệ slice backends bằng RWMutex để đảm bảo an toàn khi truy cập từ nhiều goroutine. 
type Pool struct {
	mu sync.RWMutex
	backends []backend.BackEnd
}

// Khởi tạo một Pool mới với các backend được cung cấp. Sử dụng append để tạo một slice mới, tránh việc thay đổi slice gốc từ bên ngoài.
func New(bs ...backend.BackEnd) *Pool {
	return &Pool{backends: append([]backend.BackEnd(nil), bs...)}
}

// Add thêm backend. 
func(p *Pool) Add(b backend.BackEnd) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.backends = append(p.backends, b)
}

// trả về bản sao danh sách
func(p *Pool) Snapshot() []backend.BackEnd {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]backend.BackEnd(nil), p.backends...)
}

// Healthy trả về danh sách backend dang sống
func(p *Pool) Healthy(exclude map[string]struct{}) []backend.BackEnd {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]backend.BackEnd, 0, len(p.backends))
	for _, b := range p.backends{
		if !b.IsAlive() {
			continue
		}
		if _, skip := exclude[b.URL().String()]; skip {
			continue
		}
		out = append(out, b)
	}
	return out
}