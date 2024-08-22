package app

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"text/template"

	"go.uber.org/zap"
)

type IpxeDistroList []*Distribution

func (l IpxeDistroList) Len() int      { return len(l) }
func (l IpxeDistroList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l IpxeDistroList) Less(i, j int) bool {
	return l[i].Default
}

type IpxeRendererHandler struct {
	Logger       *zap.Logger
	VarsConfig   *VarsConfig
	NtpServer    string
	HttpServer   string
	CatalogWatch chan DistroList
	x86Distros   IpxeDistroList
	arm64Distros IpxeDistroList
	template     *template.Template
	sync.RWMutex
}

func (h *IpxeRendererHandler) ParseTemplate(content string) (err error) {
	h.Lock()
	defer h.Unlock()

	h.template, err = template.New("ipxe").Parse(content)
	return err
}

func (h *IpxeRendererHandler) WatchCatalogAsync(ctx context.Context, wg *sync.WaitGroup) {
	go func() {
		wg.Add(1)
		defer wg.Done()

		h.Logger.Info("Starting IPXE catalog watcher")

		for {
			select {
			case distros := <-h.CatalogWatch:
				h.Logger.Info("IPXE catalog watcher updated")
				h.updateDistros(distros)
			case <-ctx.Done():
				h.Logger.Info("Shutting down IPXE catalog watcher")
				return
			}
		}
	}()
}

func (h *IpxeRendererHandler) updateDistros(distros []*Distribution) {
	x86Distros, arm64Distros := IpxeDistroList{}, IpxeDistroList{}
	for _, d := range distros {
		if d.Architecture == "x86_64" {
			x86Distros = append(x86Distros, d)
		} else if d.Architecture == "aarch64" {
			arm64Distros = append(arm64Distros, d)
		}
	}
	sort.Stable(x86Distros)
	sort.Stable(arm64Distros)

	h.Lock()
	defer h.Unlock()
	h.x86Distros = x86Distros
	h.arm64Distros = arm64Distros
}

func (h *IpxeRendererHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.RLock()
	defer h.RUnlock()

	w.Header().Set("Content-Type", "text/plain")

	if err := h.template.Execute(w, map[string]any{
		"DefaultVars":  h.VarsConfig.DefaultVars,
		"ProductVars":  h.VarsConfig.ProductVars,
		"HttpServer":   h.HttpServer,
		"NTP":          h.NtpServer,
		"X86Distros":   h.x86Distros,
		"ARM64Distros": h.arm64Distros,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.Logger.Error("Error rendering IPXE template",
			zap.String("mac", r.PathValue("mac")),
			zap.Error(err),
		)
		return
	}
}
