package scanner

import (
	"io"
	"math/rand"
	"net/http"
	"time"
)

type SmartFilter struct {
	notFoundSize    int64
	notFoundPattern string
	initialized     bool
}

func (s *Scanner) calibrate() {
	if s.filter == nil {
		return
	}

	rand.Seed(time.Now().UnixNano())
	randomPath := generateRandomString(32)

	resp, err := s.client.Get(s.config.BaseURL + "/" + randomPath)
	if err != nil || resp == nil {
		return
	}
	defer resp.Body.Close()

	s.filter.notFoundSize = resp.ContentLength

	body, err := io.ReadAll(resp.Body)
	if err == nil {
		s.filter.notFoundPattern = string(body)
	}

	s.filter.initialized = true
}

func (s *Scanner) isRealResult(resp *http.Response) bool {
	// if s.filter == nil || !s.filter.initialized {
	//     return true
	// }

	// if resp.ContentLength == s.filter.notFoundSize {
	//     return false
	// }

	return true
}

func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
