package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	targetURL  = "http://localhost:8080/check"
	numWorkers = 200 // concurrent API keys
	duration   = 10 * time.Second
)

type workerResult struct {
	apiKey       string
	total        int
	allowed      int
	limited      int
	errors       int
	totalLatency time.Duration
}

func runWorker(apiKey string, stop <-chan struct{}, wg *sync.WaitGroup, results chan<- workerResult) {
	defer wg.Done()

	client := &http.Client{Timeout: 5 * time.Second}
	result := workerResult{apiKey: apiKey}

	for {
		select {
		case <-stop:
			results <- result
			return
		default:
			start := time.Now()

			req, err := http.NewRequest("GET", targetURL, nil)
			if err != nil {
				result.errors++
				result.total++
				continue
			}
			req.Header.Set("X-API-Key", apiKey)

			resp, err := client.Do(req)
			latency := time.Since(start)

			if err != nil {
				result.errors++
				result.total++
				continue
			}
			resp.Body.Close()

			result.total++
			result.totalLatency += latency

			switch resp.StatusCode {
			case http.StatusOK:
				result.allowed++
			case http.StatusTooManyRequests:
				result.limited++
			}
		}
	}
}

func main() {
	fmt.Println("=== Rate Limiter Load Test ===")
	fmt.Printf("Target:   %s\n", targetURL)
	fmt.Printf("Workers:  %d\n", numWorkers)
	fmt.Printf("Duration: %s\n\n", duration)

	stop := make(chan struct{})
	results := make(chan workerResult, numWorkers)

	var wg sync.WaitGroup
	var ready sync.WaitGroup

	// Countdown so all workers start at the same time
	ready.Add(numWorkers)
	var readyCount atomic.Int32

	for i := 0; i < numWorkers; i++ {
		apiKey := fmt.Sprintf("user-%d", i+1)
		wg.Add(1)
		go func(key string) {
			readyCount.Add(1)
			ready.Done()
			ready.Wait() // block until all workers are ready
			runWorker(key, stop, &wg, results)
		}(apiKey)
	}

	// Wait for all goroutines to be ready
	ready.Wait()
	fmt.Println("All workers ready — starting test...")

	time.Sleep(duration)
	close(stop)
	wg.Wait()
	close(results)

	// Aggregate results
	var (
		totalReqs    int
		totalAllowed int
		totalLimited int
		totalErrors  int
		totalLatency time.Duration
	)

	fmt.Println("\n--- Per Worker Results ---")
	fmt.Printf("%-12s %8s %8s %8s %8s %10s\n", "API Key", "Total", "200", "429", "Errors", "Avg Lat")
	fmt.Println("--------------------------------------------------------------")

	for r := range results {
		avgLat := time.Duration(0)
		if r.total > 0 {
			avgLat = r.totalLatency / time.Duration(r.total)
		}
		fmt.Printf("%-12s %8d %8d %8d %8d %10s\n",
			r.apiKey, r.total, r.allowed, r.limited, r.errors, avgLat.Round(time.Microsecond))

		totalReqs += r.total
		totalAllowed += r.allowed
		totalLimited += r.limited
		totalErrors += r.errors
		totalLatency += r.totalLatency
	}

	avgLatency := time.Duration(0)
	if totalReqs > 0 {
		avgLatency = totalLatency / time.Duration(totalReqs)
	}

	allowedPct := 0.0
	limitedPct := 0.0
	if totalReqs > 0 {
		allowedPct = float64(totalAllowed) / float64(totalReqs) * 100
		limitedPct = float64(totalLimited) / float64(totalReqs) * 100
	}

	fmt.Println("\n--- Summary ---")
	fmt.Printf("Total requests : %d\n", totalReqs)
	fmt.Printf("200 Allowed    : %d (%.1f%%)\n", totalAllowed, allowedPct)
	fmt.Printf("429 Limited    : %d (%.1f%%)\n", totalLimited, limitedPct)
	fmt.Printf("Errors         : %d\n", totalErrors)
	fmt.Printf("Avg latency    : %s\n", avgLatency.Round(time.Microsecond))
	fmt.Printf("Throughput     : %.0f req/s\n", float64(totalReqs)/duration.Seconds())
}
