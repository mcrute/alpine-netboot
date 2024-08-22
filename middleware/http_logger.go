package middleware

import (
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type stubWriter struct {
	http.ResponseWriter
	code     int
	bytesOut int
}

func (s *stubWriter) Write(data []byte) (int, error) {
	i, err := s.ResponseWriter.Write(data)
	s.bytesOut += i
	return i, err
}

func (s *stubWriter) WriteHeader(statusCode int) {
	s.code = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
}

func HttpLogger(log *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bi, _ := strconv.Atoi(r.Header.Get("Content-Length"))

		sw := &stubWriter{ResponseWriter: w, code: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(sw, r)
		end := time.Now()

		log.Info("",
			zap.String("protocol", "http"),
			zap.String("method", r.Method),
			zap.String("uri", r.URL.Path),
			zap.String("remote_ip", r.RemoteAddr),
			zap.String("host", r.Host),
			zap.String("user_agent", r.Header.Get("User-Agent")),
			zap.Duration("latency", end.Sub(start)),
			zap.String("latency_human", end.Sub(start).String()),
			zap.Int("status", sw.code),
			zap.Int("bytes_in", bi),
			zap.Int("bytes_out", sw.bytesOut),
		)
	})
}
