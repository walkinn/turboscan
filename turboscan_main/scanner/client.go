package scanner

import (
    "crypto/tls"
    "net"
    "net/http"
    "time"
)

type HTTPClient struct {
    client *http.Client
    config Config
}

func NewHTTPClient(config Config) *HTTPClient {
    transport := &http.Transport{
        MaxIdleConns:        1000,
        MaxIdleConnsPerHost: 500,
        IdleConnTimeout:     90 * time.Second,

        DialContext: (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,

        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: true,
        },

        DisableCompression: false,
        DisableKeepAlives:  false,
        ForceAttemptHTTP2:  true,

        MaxResponseHeaderBytes: 4096,

        ResponseHeaderTimeout: time.Duration(config.Timeout) * time.Second,
        TLSHandshakeTimeout:   10 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
    }

    client := &http.Client{
        Transport: transport,
        Timeout:   time.Duration(config.Timeout) * time.Second,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse
        },
    }

    return &HTTPClient{
        client: client,
        config: config,
    }
}

func (c *HTTPClient) Get(url string) (*http.Response, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("User-Agent", "TurboScan/1.0")
    req.Header.Set("Accept", "*/*")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }

    return resp, nil
}

func (c *HTTPClient) GetWithRetry(url string, maxRetries int) (*http.Response, error) {
    if maxRetries <= 0 {
        return c.Get(url)
    }

    var resp *http.Response
    var err error

    for i := 0; i < maxRetries; i++ {
        resp, err = c.Get(url)
        if err == nil {
            return resp, nil
        }

        if isTemporaryError(err) {
            time.Sleep(time.Duration(i+1) * time.Second) // exponential backoff
            continue
        }

        return nil, err
    }

    return nil, err
}

func isTemporaryError(err error) bool {
    if netErr, ok := err.(net.Error); ok {
        return netErr.Timeout() || netErr.Temporary()
    }
    return false
}
