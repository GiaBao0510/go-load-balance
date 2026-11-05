package strategy

import (
	"sync"

	"github.com/GiaBao0510/go_loadbalance/internal/backend"
)

/*
• Định nghĩa: Round Robin có trọng số: server trọng số 3 nhận số request gấp 3 lần server trọng số 1.
• Đặc điểm: mỗi server giữ một "điểm hiện tại" (current). Mỗi lượt: cộng trọng số vào điểm của mọi server, chọn server có điểm cao nhất, rồi trừ tổng trọng số khỏi điểm của server được chọn. Đây là biến thể smooth của NGINX.
• Ưu điểm: phân phối đúng tỷ lệ và rải đều. Với trọng số A=3, B=1 dãy chọn là A, A, B, A (bản "cổ điển" cho A, A, A, B gây dồn cục).
• Nhược điểm: cần trạng thái có khóa; vẫn không phản ứng với tải thực tế.
• Khi nào dùng: cụm server không đồng nhất (máy mạnh yếu khác nhau), hoặc khi triển khai canary với tỷ lệ nhỏ.
*/
type WeightRoundRobin struct {
	mu      sync.Mutex     // Cần mutex vì mỗi lần chọn là một chuỗi đọc-sửa-ghi trên nhiều phần tử.
	current map[string]int // điểm hiện tại của từng server
}

// Hàm khởi tạo
func NewWeightRoundRobin() *WeightRoundRobin {
	return &WeightRoundRobin{
		current: make(map[string]int),
	}
}

func (w *WeightRoundRobin) Next(c []backend.BackEnd) backend.BackEnd {

	// Trường hợp không có backend nào
	if len(c) == 0 {
		return nil
	}

	// Khóa mutex để đảm bảo an toàn khi truy cập và sửa đổi trạng thái
	w.mu.Lock()
	defer w.mu.Unlock()

	var (
		best      backend.BackEnd // backend có điểm hiện tại cao nhất
		bestScore int             // điểm hiện tại cao nhất
		total     int             // tổng trọng số của tất cả backend
	)

	for _, b := range c {
		key := b.URL().String()
		w.current[key] += b.Weight() // cộng trọng số vào điểm hiện tại
		total += b.Weight()          // cộng tổng trọng số
		if best == nil || w.current[key] > bestScore {
			best, bestScore = b, w.current[key] // cập nhật backend tốt nhất và điểm cao nhất
		}
	}

	w.current[best.URL().String()] -= total // trừ tổng trọng số khỏi điểm của backend được chọn
	return best
}
