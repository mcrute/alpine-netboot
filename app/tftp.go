package app

import (
	"io"
	"io/fs"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	tftpServeSuccessMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "netboot_tftp_read_success",
		Help: "Successful TFTP read responses",
	}, []string{"filename"})
	tftpServeFailuresMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "netboot_tftp_read_failure",
		Help: "Failed TFTP read responses",
	}, []string{"filename"})
)

type TftpHandler struct {
	Root fs.FS
}

func (h *TftpHandler) HandleRead(filename string, rf io.ReaderFrom) (int64, error) {
	file, err := h.Root.Open(filename)
	if err != nil {
		tftpServeFailuresMetric.WithLabelValues(filename).Inc()
		return 0, err
	}
	defer file.Close()

	n, err := rf.ReadFrom(file)
	if err != nil {
		tftpServeFailuresMetric.WithLabelValues(filename).Inc()
		return 0, err
	}

	tftpServeSuccessMetric.WithLabelValues(filename).Inc()
	return n, nil
}
