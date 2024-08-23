package app

import (
	"net/http"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	macLookupErrorMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "netboot_apkovl_lookup_failures",
		Help: "Failures when looking up a MAC address",
	})
	defaultGenerateErrorMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "netboot_apkovl_default_gen_error",
		Help: "Failures when generating a default apkovl",
	})
	generateErrorMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "netboot_apkovl_gen_error",
		Help: "Failures when generating a MAC-specific apkovl",
	})
	apkovlServeDefaultMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "netboot_apkovl_serve_default",
		Help: "Default apkovl files served",
	})
	apkovlGenerateSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "netboot_apkovl_success",
		Help: "Successfully generated apkovl files",
	})
)

type ApkOvlHandler struct {
	Logger      *zap.Logger
	Coordinator *netboxconfig.ConfigCoordinator
}

func (h *ApkOvlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/gzip")

	mac := r.PathValue("mac")
	ctx := r.Context()

	macExists, err := h.Coordinator.MacExists(ctx, mac)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.Logger.Error("Error looking up MAC address", zap.String("mac", mac), zap.Error(err))
		macLookupErrorMetric.Inc()
		return
	}

	if !macExists {
		h.Logger.Info("No netbox config for mac", zap.String("mac", mac))

		if err := h.Coordinator.GenerateDefault(ctx, w); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.Logger.Error("Error generating default APKOVL", zap.String("mac", mac), zap.Error(err))
			defaultGenerateErrorMetric.Inc()
			return
		}

		apkovlServeDefaultMetric.Inc()
		return
	}

	if err := h.Coordinator.GenerateForMac(ctx, mac, w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.Logger.Error("Error generating APKOVL", zap.String("mac", mac), zap.Error(err))
		generateErrorMetric.Inc()
		return
	}

	apkovlGenerateSuccess.Inc()
}
