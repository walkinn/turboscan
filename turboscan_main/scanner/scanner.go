package scanner

import (
    "fmt"
    "sync"
    "time"
)

type Config struct {
    BaseURL     string
    Threads     int
    Timeout     int
    StatusCodes []int
    Verbose     bool

    Rate       int
    MaxRetries int

    Recursive  bool
    MaxDepth   int
    Extensions []string

    WordlistPath string
}

type Result struct {
    URL        string        `json:"url"`
    StatusCode int           `json:"status"`
    Size       int64         `json:"size"`
    Time       time.Duration `json:"time"`
}

type Stats struct {
    Total     int
    Success   int
    Failed    int
    StartTime time.Time
    EndTime   time.Time
}

type Scanner struct {
    config      Config
    client      *HTTPClient
    results     []Result
    stats       Stats
    mutex       sync.Mutex
    rateLimiter *RateLimiter
    filter      *SmartFilter
}

func NewScanner(config Config) *Scanner {
    s := &Scanner{
        config:  config,
        client:  NewHTTPClient(config),
        results: make([]Result, 0),
        stats: Stats{
            StartTime: time.Now(),
        },
        filter: &SmartFilter{},
    }

    if config.Rate > 0 {
        s.rateLimiter = NewRateLimiter(config.Rate)
    }

    s.calibrate()

    return s
}

func (s *Scanner) Scan(words []string) []Result {
    jobs := make(chan string, len(words))
    results := make(chan Result, len(words))

    var wg sync.WaitGroup

    for i := 0; i < s.config.Threads; i++ {
        wg.Add(1)
        go s.worker(jobs, results, &wg)
    }

    go s.collector(results)

    s.stats.Total = len(words)

    for _, word := range words {
        jobs <- word
    }
    close(jobs)

    wg.Wait()
    close(results)

    s.stats.EndTime = time.Now()
    return s.results
}

func (s *Scanner) ScanWithExtensions(words []string, extensions []string) []Result {
    var allWords []string
    allWords = make([]string, 0, len(words)*(len(extensions)+1))

    for _, w := range words {
        allWords = append(allWords, w)
        for _, ext := range extensions {
            allWords = append(allWords, w+"."+ext)
        }
    }

    return s.Scan(allWords)
}

func (s *Scanner) ScanRecursive(baseURL string, words []string, depth, maxDepth int) []Result {
    if depth > maxDepth {
        return nil
    }

    fmt.Printf("[*] Recursive scan: depth=%d base=%s\n", depth, baseURL)

    oldBase := s.config.BaseURL
    s.config.BaseURL = baseURL

    s.mutex.Lock()
    s.results = nil
    s.stats = Stats{
        StartTime: time.Now(),
    }
    s.mutex.Unlock()

    res := s.Scan(words)

    all := make([]Result, 0, len(res))
    all = append(all, res...)

    for _, r := range res {
        if r.StatusCode == 301 || r.StatusCode == 302 {
            sub := s.ScanRecursive(r.URL, words, depth+1, maxDepth)
            all = append(all, sub...)
        }
    }

    s.config.BaseURL = oldBase
    return all
}

func (s *Scanner) shouldReport(statusCode int) bool {
    for _, code := range s.config.StatusCodes {
        if code == statusCode {
            return true
        }
    }
    return false
}

func (s *Scanner) incrementSuccess() {
    s.mutex.Lock()
    s.stats.Success++
    s.mutex.Unlock()
}

func (s *Scanner) incrementFailed() {
    s.mutex.Lock()
    s.stats.Failed++
    s.mutex.Unlock()
}

func (s *Scanner) PrintStats() {
    duration := s.stats.EndTime.Sub(s.stats.StartTime)
    if duration <= 0 {
        duration = time.Since(s.stats.StartTime)
    }
    reqPerSec := 0.0
    if duration.Seconds() > 0 {
        reqPerSec = float64(s.stats.Total) / duration.Seconds()
    }

    fmt.Println("\n[*] Scan Statistics:")
    fmt.Printf("    Total Requests:  %d\n", s.stats.Total)
    fmt.Printf("    Successful:      %d\n", s.stats.Success)
    fmt.Printf("    Failed:          %d\n", s.stats.Failed)
    fmt.Printf("    Duration:        %v\n", duration)
    fmt.Printf("    Req/sec:         %.2f\n", reqPerSec)
}

func (s *Scanner) notify(result Result) {

    if result.StatusCode == 200 {
        // sendTelegramMessage(fmt.Sprintf("Found: %s", result.URL))
        // output only:
        // fmt.Printf("[!] NOTIFY: %s\n", result.URL)
    }
}
