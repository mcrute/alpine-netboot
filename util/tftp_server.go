package util

import (
	"context"
	"io"

	"code.crute.us/mcrute/netboot-server/middleware"
	"github.com/pin/tftp"
	"go.uber.org/zap"
)

type (
	tftpReadHandlerFunc  func(string, io.ReaderFrom) error
	tftpWriteHandlerFunc func(string, io.WriterTo) error
)

type TftpReadHandler interface {
	HandleRead(filename string, rf io.ReaderFrom) (int64, error)
}

type TftpWriteHandler interface {
	HandleWrite(filename string, rf io.ReaderFrom) (int64, error)
}

type TftpServer struct {
	Addr         string
	Logger       *zap.Logger
	ReadHandler  TftpReadHandler
	WriteHandler TftpWriteHandler
	server       *tftp.Server
}

func (s *TftpServer) ListenAndServe() error {
	var readHandler tftpReadHandlerFunc
	if s.ReadHandler != nil {
		readHandler = middleware.TftpLogger(s.Logger, s.ReadHandler.HandleRead)
	}

	var writeHandler tftpWriteHandlerFunc
	if s.WriteHandler != nil {
		readHandler = middleware.TftpLogger(s.Logger, s.WriteHandler.HandleWrite)
	}

	s.server = tftp.NewServer(readHandler, writeHandler)

	if s.Logger != nil {
		s.Logger.Sugar().Infof("TFTP server listening on %s", s.Addr)
	}

	return s.server.ListenAndServe(s.Addr)
}

func (s *TftpServer) ListenAndServeAsync() {
	go func() {
		if err := s.ListenAndServe(); err != nil && s.Logger != nil {
			s.Logger.Error("Error stopping tftp server", zap.Error(err))
		}
		if s.Logger != nil {
			s.Logger.Info("Tftp server has shut down")
		}
	}()
}

func (s *TftpServer) Shutdown(ctx context.Context) error {
	s.server.Shutdown()
	return nil
}
