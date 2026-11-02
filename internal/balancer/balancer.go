package balancer

import (
	"github.com/GiaBao0510/go_loadbalance/internal/backend"
	"net/http"
)

type LoadBalancer interface {
	http.Handler
	AddBackend(b backend.BackEnd)
}