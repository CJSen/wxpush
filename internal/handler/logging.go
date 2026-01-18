package handler

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var requestLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// WrapWithLogging 统一记录入站请求日志，包含时间、耗时、状态码等关键信息。
func WrapWithLogging(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		requestLogger.Printf(
			"request route=%s method=%s path=%s status=%d bytes=%d duration=%s remote=%s ua=%q",
			route,
			r.Method,
			r.URL.Path,
			status,
			recorder.bytes,
			time.Since(start),
			clientIP(r),
			r.UserAgent(),
		)
	})
}

func clientIP(r *http.Request) string {
	if ip := firstIP(r.Header.Get("X-Forwarded-For")); ip != "" {
		return ip
	}
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func firstIP(list string) string {
	if list == "" {
		return ""
	}
	parts := strings.Split(list, ",")
	return strings.TrimSpace(parts[0])
}
