package app

import (
	"net/http"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
	"go.uber.org/zap"
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
		return
	}

	if !macExists {
		h.Logger.Info("No netbox config for mac", zap.String("mac", mac))

		if err := h.Coordinator.GenerateDefault(ctx, w); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.Logger.Error("Error generating default APKOVL", zap.String("mac", mac), zap.Error(err))
			return
		}

		return
	}

	if err := h.Coordinator.GenerateForMac(ctx, mac, w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.Logger.Error("Error generating APKOVL", zap.String("mac", mac), zap.Error(err))
		return
	}
}
