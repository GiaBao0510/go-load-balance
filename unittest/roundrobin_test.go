package unittest

import (
	"sync"
	"testing"

	"github.com/GiaBao0510/go_loadbalance/internal/backend"
	"github.com/GiaBao0510/go_loadbalance/internal/strategy"
)

func TestRoundRobinConcurrent(t *testing.T) {
	var candidates []backend.BackEnd
	for _, u := range []string{"http://localhost:8080", "http://localhost:8081", "http://localhost:8082"} {
		b, err := backend.New(u, 1)
		if err != nil {
			t.Fatalf("failed to create backend: %v", err)
		}
		candidates = append(candidates, b)
	}

	rr := strategy.NewRoundRobin()
	var (
		mu     sync.Mutex
		counts = make(map[string]int)
		wg     sync.WaitGroup
	)

	for range 3000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			picked := rr.Next(candidates).URL().Host
			mu.Lock()
			counts[picked]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	for host, n := range counts {
		if n != 1000 {
			t.Errorf("host %s got %d requests, expected 1000", host, n)
		}
	}
}

type ServerPool struct {
	current int
}
