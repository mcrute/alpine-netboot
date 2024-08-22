package app

import (
	"fmt"
	"net/http"
)

type WebIndexHandler struct{}

func (h *WebIndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, "<pre>")
	fmt.Fprintln(w, `<a href="/boot.ipxe">/boot.ipxe</a>`)
	fmt.Fprintln(w, `<a href="/distros/">/distros/</a>`)
	fmt.Fprintln(w, `<a href="/tftpboot/">/tftpboot/</a>`)
	fmt.Fprintln(w, "</pre>")
}
