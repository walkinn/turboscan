package scanner

import (
    "fmt"
    "net/http"
    "sync"
    "time"
)

type RateLimiter struct {
    rate   int
    ticker *time.Ticker
    tokens chan struct{}
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
    if requestsPerSecond <= 0 {
        return nil
    }

    rl := &RateLimiter{
        rate:   requestsPerSecond,
        tokens: make(chan struct{}, requestsPerSecond),
    }

    for i := 0; i < requestsPerSecond; i++ {
        rl.tokens <- struct{}{}
    }

    rl.ticker = time.NewTicker(time.Second / time.Duration(requestsPerSecond))

    go func() {
        for range rl.ticker.C {
            select {
            case rl.tokens <- struct{}{}:
            default:
            }
        }
    }()

    return rl
}

func (rl *RateLimiter) Wait() {
    if rl == nil {
        return
    }
    <-rl.tokens
}

func (s *Scanner) worker(jobs <-chan string, results chan<- Result, wg *sync.WaitGroup) {
    defer wg.Done()

    for word := range jobs {
        if s.rateLimiter != nil {
            s.rateLimiter.Wait()
        }

        url := s.config.BaseURL + "/" + word

        start := time.Now()
        var resp *http.Response
        var err error

        if s.config.MaxRetries > 0 {
            resp, err = s.client.GetWithRetry(url, s.config.MaxRetries)
        } else {
            resp, err = s.client.Get(url)
        }

        duration := time.Since(start)

        if err != nil {
            s.incrementFailed()
            if s.config.Verbose {
                fmt.Printf("[-] %s - Error: %v\n", url, err)
            }
            continue
        }

        if resp.Body != nil {
            resp.Body.Close()
        }

        if !s.isRealResult(resp) {
            s.incrementFailed()
            continue
        }

        if s.shouldReport(resp.StatusCode) {
            result := Result{
                URL:        url,
                StatusCode: resp.StatusCode,
                Size:       resp.ContentLength,
                Time:       duration,
            }
            results <- result
            s.incrementSuccess()
        } else {
            s.incrementFailed()
        }
    }
}

func (s *Scanner) collector(results <-chan Result) {
    for result := range results {
        s.mutex.Lock()
        s.results = append(s.results, result)
        s.mutex.Unlock()

        fmt.Printf("[+] %d - %s [Size: %d] [Time: %v]\n",
            result.StatusCode,
            result.URL,
            result.Size,
            result.Time,
        )

        s.notify(result)
    }
}
