package plugins

import (
	"context"
	"encoding/json"
	"fmt"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
)

func init() {
	netboxconfig.RegisterConfigFunc("hostname", generateHostname)
}

func generateHostname(_ context.Context, ovl *netboxconfig.APKOVL, _ json.RawMessage, cfg *netboxconfig.RawConfig) error {
	if err := ovl.AddStringFile(cfg.Name, "etc/hostname", 0644); err != nil {
		return err
	}

	fqdn := fmt.Sprintf("%s.%s", cfg.Name, cfg.Site.CustomFields.BaseFqdn)

	hostEntries := []string{
		fmt.Sprintf("127.0.0.1       %s %s localhost localhost.localdomain", fqdn, cfg.Name),
		fmt.Sprintf("::1             %s %s localhost localhost.localdomain", fqdn, cfg.Name),
	}

	return ovl.AddStringListFile(hostEntries, "etc/hosts", 0644)
}
