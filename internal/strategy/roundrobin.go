package strategy

import (
	"sync/atomic"
	"github.com/GiaBao0510/go_loadbalance/internal/backend"
)
/*
• Định nghĩa: gửi request lần lượt theo vòng tròn: 1, 2, 3, 1, 2, 3, ...
• Đặc điểm: không cần biết tải hay cấu hình từng server; trạng thái chỉ là một bộ đếm.
• Ưu điểm: đơn giản, nhanh, công bằng về số lượng request. Nhược điểm: coi mọi server và mọi request là như nhau; không phù hợp khi server cấu hình khác nhau hoặc request nặng nhẹ chênh lệch lớn.
• Khi nào dùng: cụm server đồng nhất, request có chi phí xấp xỉ nhau.
*/
type RoundRobin struct {
	counter atomic.Uint64
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

func(r *RoundRobin) Next(c []backend.BackEnd) backend.BackEnd {
	if len(c) == 0 {
		return nil
	}

	n := r.counter.Add(1) - 1 // nguyên tử; uint64 nên không có chỉ số âm
	return c[n % uint64(len(c))] // vòng tròn
}