package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Struct này định nghĩa các server
type BackEnd struct {
	URL          *url.URL               // Địa chỉ server
	Alive        bool                   // Đánh dấu xem server có đang hoạt động hay không
	ReverseProxy *httputil.ReverseProxy // ReverseProxy để chuyển tiếp request đến server
}

// Struct này định nghĩa một pool các server
type ServerPool struct {
	backends []*BackEnd // Danh sách các server
	current  int64      // Chỉ số của server hiện tại trong danh sách
}

// Thêm một server vào pool
func (s *ServerPool) AddBackend(backend *BackEnd) {
	s.backends = append(s.backends, backend)
}

// Lấy chỉ số của server tiếp theo trong pool
func (s *ServerPool) NextIndex() int64 {
	s.current++
	return s.current % int64(len(s.backends)) // Lấy kết quả của phép chia dư => Áp dụng thuật toán Round Robin để chọn server tiếp theo
}

// Lấy server tiếp theo trong pool
func (s *ServerPool) GetNextBackend() *BackEnd {
	next := s.NextIndex()
	return s.backends[next] // lấy ra thứ tự của server tiếp theo và trả về server với thứ tự tương ứng nằm trong thuộc tính backends
}

func main() {

	var serverList string
	var port int

	flag.StringVar(&serverList, "backends", "", "Load balanced backeds, use commas to separate") // Đọc danh sách các biến truyền vào từ Terminal, ví dụ: go run main.go -backends=http://localhost:8080,http://localhost:8081
	flag.IntVar(&port, "port", 3000, "Port to serve")                                            // Đọc cổng mà server sẽ lắng nghe từ Terminal, ví dụ: go run main.go -port=3000
	flag.Parse()

	if len(serverList) == 0 {
		log.Fatal("Please provide one or more backends to load balance")
	}

	servers := strings.Split(serverList, ",") // Tách danh sách các server thành một mảng các chuỗi

	serverPool := ServerPool{current: -1} // Tạo một pool các server
	for _, s := range servers {
		serverURL, err := url.Parse(s) // Chuyển đổi chuỗi URL thành một đối tượng URL
		if err != nil {
			log.Fatal(err)
		}

		// Tạo một ReverseProxy để chuyển tiếp request đến server
		proxy := httputil.NewSingleHostReverseProxy(serverURL)

		// Thêm server vào pool
		serverPool.AddBackend(&BackEnd{
			URL:          serverURL,
			Alive:        true,
			ReverseProxy: proxy,
		})
	}

	// Tạo một server HTTP lắng nghe trên cổng được chỉ định và chuyển tiếp tất cả các request đến server tiếp theo trong pool
	server := http.Server{
		Addr: fmt.Sprintf(":%d", port), // Lắng nghe trên cổng được chỉ định
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // Tạo một handler để xử lý tất cả các request
			peer := serverPool.GetNextBackend() // Lấy server tiếp theo trong pool

			if peer != nil {
				peer.ReverseProxy.ServeHTTP(w, r) // Chuyển tiếp request đến server tiếp theo trong pool
				return
			}

			http.Error(w, "Service not available", http.StatusServiceUnavailable) // Nếu không có server nào trong pool, trả về lỗi 503
		}),
	}

	log.Printf("Load Balancer started at :%d\n", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
