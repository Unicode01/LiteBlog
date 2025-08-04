package utils

import (
	"net/http"
	"sync"
	"sync/atomic"
)

type SnifferManager struct {
	allRequestCounter   atomic.Uint64
	pathRequestCounter  *sync.Map
	responseCodeCounter *sync.Map
}

func NewSnifferManager() *SnifferManager {
	sm := &SnifferManager{
		pathRequestCounter:  &sync.Map{},
		responseCodeCounter: &sync.Map{},
	}
	return sm
}

func (sm *SnifferManager) AllRequestCount() uint64 {
	return sm.allRequestCounter.Load()
}

func (sm *SnifferManager) PathRequestCount(path string) uint64 {
	if counter, exists := sm.pathRequestCounter.Load(path); exists {
		return counter.(*atomic.Uint64).Load()
	}
	return 0
}

func (sm *SnifferManager) ResponseCodeCount(code int) uint64 {
	if counter, exists := sm.responseCodeCounter.Load(code); exists {
		return counter.(*atomic.Uint64).Load()
	}
	return 0
}

// count path when the respond code is 2xx
func (sm *SnifferManager) OnNewRequest(path string, respondCode int) {
	// calc all request counter
	sm.allRequestCounter.Add(1)

	// calc response code counter
	newCounter := &atomic.Uint64{}
	counter, loaded := sm.responseCodeCounter.LoadOrStore(respondCode, &atomic.Uint64{})
	if loaded {
		counter.(*atomic.Uint64).Add(1)
	} else {
		newCounter.Add(1)
	}

	// calc path counter
	if respondCode < 200 || respondCode >= 300 { // 200-300
		return
	}

	if counter, exists := sm.pathRequestCounter.Load(path); exists {
		counter.(*atomic.Uint64).Add(1)
		return
	}

	newCounter = &atomic.Uint64{}
	counter, loaded = sm.pathRequestCounter.LoadOrStore(path, newCounter)
	if loaded {
		counter.(*atomic.Uint64).Add(1)
	} else {
		newCounter.Add(1)
	}
}

func (sm *SnifferManager) ProxyResponseWriter(w http.ResponseWriter, r *http.Request) ProxyResponseWriter {
	pw := ProxyResponseWriter{w, 200, sm.OnNewRequest, r}
	return pw
}

type ProxyResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	OnNewRequest func(path string, respondCode int)
	Request      *http.Request
}

func (pw *ProxyResponseWriter) WriteHeader(statusCode int) {
	pw.statusCode = statusCode
	pw.ResponseWriter.WriteHeader(statusCode)
}

func (pw *ProxyResponseWriter) Write(b []byte) (int, error) {
	lenofb := len(b)
	i, e := pw.ResponseWriter.Write(b)
	if i == lenofb && e == nil {
		pw.OnNewRequest(pw.Request.URL.Path, pw.statusCode)
	}
	return i, e
}

func (pw *ProxyResponseWriter) StatusCode() int {
	return pw.statusCode
}
