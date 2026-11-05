package strategy

import (
	"github.com/GiaBao0510/go_loadbalance/internal/backend"
)

/*
• Định nghĩa: chọn server đang có ít kết nối (request đang xử lý) nhất.
• Đặc điểm: dựa trên tải thực tế đo được qua ActiveConns(); không cần trạng thái riêng trong strategy.
• Ưu điểm: thích nghi tự nhiên khi request có thời gian xử lý chênh lệch (upload, báo cáo nặng, WebSocket).
• Nhược điểm: có "cửa sổ race" nhỏ giữa lúc đọc bộ đếm và lúc request thực sự bắt đầu, nên hai request đến cùng lúc có thể chọn cùng một server (chấp nhận được vì đây là heuristic); chi phí O(n) mỗi lượt chọn.
• Khi nào dùng: thời gian xử lý mỗi request biến động mạnh.
*/
type LeastConnection struct{}

// Hàm khoi tạo
func NewLeastConnection() *LeastConnection {
	return &LeastConnection{}
}

func (l *LeastConnection) Next(c []backend.BackEnd) backend.BackEnd {

	var (
		best   backend.BackEnd // backend có ít kết nối nhất
		fewest int64           // số kết nối ít nhất
	)

	for _, b := range c {
		n := b.ActiveConns() // số kết nối đang hoạt động
		if best == nil || n < fewest {
			best, fewest = b, n // cập nhật backend tốt nhất và số kết nối ít nhất
		}
	}

	return best // nil nếu c đúng
}
