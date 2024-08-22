package plugins

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
)

func init() {
	netboxconfig.RegisterSimpleConfigFunc("load_kernel_modules", generateLoadKernelModules)
}

func generateLoadKernelModules(ovl *netboxconfig.APKOVL, cfg json.RawMessage) error {
	var config map[string][]string
	if err := json.Unmarshal(cfg, &config); err != nil {
		return err
	}

	for k, v := range config {
		filename := filepath.Join("etc/modules-load.d", fmt.Sprintf("%s.conf", k))
		if err := ovl.AddStringListFile(v, filename, 0644); err != nil {
			return err
		}
	}

	return nil
}
