package app

import (
	"io"
	"io/fs"
)

type TftpHandler struct {
	Root fs.FS
}

func (h *TftpHandler) HandleRead(filename string, rf io.ReaderFrom) (int64, error) {
	file, err := h.Root.Open(filename)
	if err != nil {
		return 0, err
	}
	n, err := rf.ReadFrom(file)
	if err != nil {
		return 0, err
	}
	return n, nil

}
