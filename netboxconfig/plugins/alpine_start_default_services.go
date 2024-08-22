package plugins

import (
	"encoding/json"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
)

func init() {
	netboxconfig.RegisterSimpleConfigFunc("alpine_start_default_services", generateStartDefaultServices)
}

func generateStartDefaultServices(ovl *netboxconfig.APKOVL, cfg json.RawMessage) error {
	var enabled bool
	if err := json.Unmarshal(cfg, &enabled); err != nil {
		return err
	}

	if enabled {
		return ovl.AddEmptyFile("etc/.default_boot_services", 0644)
	}

	return nil
}
