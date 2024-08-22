package middleware

import (
	"io"

	"go.uber.org/zap"
)

func TftpLogger(log *zap.Logger, next func(string, io.ReaderFrom) (int64, error)) func(string, io.ReaderFrom) error {
	return func(filename string, rf io.ReaderFrom) error {
		n, err := next(filename, rf)
		if err != nil {
			log.Error("",
				zap.String("protocol", "tftp"),
				zap.String("uri", filename),
				zap.Error(err),
			)
		} else {
			log.Info("",
				zap.String("protocol", "tftp"),
				zap.String("uri", filename),
				zap.Int64("bytes_out", n),
			)
		}
		return err
	}
}
