# go-load-balance

Cây thư mục
```
go-load-balance/
  cmd/
    lb/main.go            # điểm vào của load balancer, chỉ làm wiring
    backend/main.go       # server giả lập để thực hành
  internal/
    backend/backend.go    # interface Backend + cài đặt (ReverseProxy, atomic state)
    pool/pool.go          # tập backend, an toàn đa luồng (RWMutex)
    strategy/             # strategy.go, roundrobin.go, weighted.go, leastconn.go
    health/checker.go     # active health check
    balancer/balancer.go  # http.Handler: chọn backend, failover, passive check
  go.mod
  README.md
```

### **Luồng xử lý như sau:**

- **Client** gửi request đến **http.Server**
- **Http.Server** gửi request của **Client** đến **Balancer.Http**

- Các bước xử lý:
1. pool.Healthy(triead) lọc tìm các backend đang còn hoạt động và chưa thử
2. Strategy.Next(...) chọn 1 backend
3. backend.Forward(...) chuyển tiếp bằng ReverseProxy

- Nếu thành công (200) -> trả response cho client
- Error -> SetAlive về false (passive check) -> quay về bước 1.

Goroutine nền: health.Checker --mỗi N giây--> GET /health từng backend --> SetAlive(true/false)  (active check)