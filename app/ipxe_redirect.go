package app

import (
	"fmt"
	"net/http"
)

type IpxeRedirectHandler struct {
	HttpServer string
}

func (h *IpxeRedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "#!ipxe")
	fmt.Fprintln(w, "set http_server", h.HttpServer)
	fmt.Fprintln(w, "chain --replace ${http_server}/${net0/mac}/boot.ipxe")
}
