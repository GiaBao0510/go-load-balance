package backend

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

// BackEnd: interface định nghĩa các phương thức mà một backend cần triển khai.
type BackEnd interface {
	URL() *url.URL
	Weight() int
	IsAlive() bool
	SetAlive(alive bool)
	ActiveConns() int64
	Forward(w http.ResponseWriter, r *http.Request) error
}

// errSlotKey:  được sử dụng để khóa context để ErrorHandler báo lỗi ngược về Forward.
type errSlotKey struct{}

// sharedTransport: được sử dụng để chia sẻ transport giữa các backend, giúp tiết kiệm tài nguyên và cải thiện hiệu suất.
var sharedTransport = &http.Transport{
	DialContext: (&net.Dialer{ // thiêt lập DialContext với các thông số timeout và keep-alive.
		Timeout:   3 * time.Second,  // timeout cho việc kết nối đến backend.
		KeepAlive: 30 * time.Second, // thời gian giữ kết nối sống.
	}).DialContext, // sử dụng DialContext của net.Dialer để thiết lập kết nối.
	ResponseHeaderTimeout: 10 * time.Second, // timeout cho việc nhận header từ backend.
	MaxConnsPerHost:       100,              // giới hạn số lượng kết nối tối đa cho mỗi host.
	IdleConnTimeout:       90 * time.Second, // thời gian giữ kết nối idle trước khi đóng.
}

// httpBackend: là cài đặt mặc định. Kiểu không xuất (unexported): bên ngoài chỉ làm việc qua interface Backend
type httpBackend struct {
	url    *url.URL               // url của backend
	weight int                    // weight của backend
	alive  atomic.Bool            // alive: trạng thái sống của backend, sử dụng atomic để đảm bảo an toàn khi truy cập từ nhiều goroutine.
	conns  atomic.Int64           // conns: số lượng kết nối hiện tại đến backend, sử dụng atomic để đảm bảo an toàn khi truy cập từ nhiều goroutine.
	proxy  *httputil.ReverseProxy // proxy: reverse proxy để chuyển tiếp các request đến backend.
}

func New(rawURL string, weight int) (BackEnd, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("backend: parse url %q: %w", rawURL, err)
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("backend: url %q needs http/https scheme and a host", rawURL)
	}

	if weight < 1 {
		weight = 1
	}

	b := &httpBackend{url: u, weight: weight}
	b.alive.Store(true)	// mặc định backend được coi là alive khi khởi tạo.

	proxy := httputil.NewSingleHostReverseProxy(u)	// tạo một reverse proxy cho backend.
	proxy.Transport = sharedTransport // sử dụng transport chia sẻ để tiết kiệm tài nguyên.

	// Ở đây không ghi response. Chỉ lúu lỗi để Forward trả về cho Balancer, nhờ vậy Balancer quyết định retry hay báo lỗi cho client 
	proxy.ErrorHandler = func(_ http.ResponseWriter, r *http.Request, err error ) {
		if slot, ok := r.Context().Value(errSlotKey{}).(*error); ok {
			*slot = err
		}
	}

	b.proxy = proxy
	return b, nil
}

func(h *httpBackend) URL() *url.URL { return h.url}
func(h *httpBackend) Weight() int { return h.weight }
func(h *httpBackend) IsAlive() bool {return h.alive.Load()}
func(h *httpBackend) SetAlive(alive bool) { h.alive.Store(alive)}
func(h *httpBackend) ActiveConns() int64 { return h.conns.Load()}

// Forward: chuyển request tới backend. ReverseProxy gọi ErrorHandler đồng bộ trong cùng goroutine, nên biến này không cần khóa
func(h *httpBackend) Forward(w http.ResponseWriter, r *http.Request) error {
	h.conns.Add(1)
	defer h.conns.Add(-1)

	var proxyErr error
	ctx := context.WithValue(r.Context(), errSlotKey{}, &proxyErr)
	h.proxy.ServeHTTP(w, r.WithContext(ctx))

	return proxyErr
}


