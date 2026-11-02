package strategy

import (
	"fmt"
	"github.com/GiaBao0510/go_loadbalance/internal/backend"
)

// chọn một backend trong danh sách ứng viên đã lọc sẵn.
type Strategy interface {
	Next(candidates []backend.BackEnd) backend.BackEnd
}

func New(name string) (Strategy, error) {
	switch name{
	case "rr":
		return NewRoundRobin(), nil
	case "wrr":
		return NewWeightRoundRobin(), nil
	case "lc":
		return NewLeastConnection(), nil
	default:
		return nil, fmt.Errorf("strategy: unknown algorithm %q (want rr, wrr or lc)", name)
	}
}