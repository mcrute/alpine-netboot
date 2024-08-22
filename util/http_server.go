package util

import (
	"context"
	"net/http"

	"code.crute.us/mcrute/netboot-server/middleware"
	"go.uber.org/zap"
)

type HttpServer struct {
	Addr    string
	Logger  *zap.Logger
	Handler http.Handler
	server  *http.Server
}

func (s *HttpServer) ListenAndServe() error {
	s.server = &http.Server{
		Addr:    s.Addr,
		Handler: middleware.HttpLogger(s.Logger, s.Handler),
	}

	if s.Logger != nil {
		s.Logger.Sugar().Infof("HTTP server listening on %s", s.Addr)
	}

	return s.server.ListenAndServe()
}

func (s *HttpServer) ListenAndServeAsync() {
	go func() {
		if err := s.ListenAndServe(); err != nil && s.Logger != nil {
			s.Logger.Error("Error stopping http server", zap.Error(err))
		}
		if s.Logger != nil {
			s.Logger.Info("Http server has shut down")
		}
	}()
}

func (s *HttpServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
