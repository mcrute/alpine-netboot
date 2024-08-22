package plugins

import (
	"encoding/json"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
)

func init() {
	netboxconfig.RegisterSimpleConfigFunc("alpine_init", generateAlpineInit)
}

func generateAlpineInit(ovl *netboxconfig.APKOVL, cfg json.RawMessage) error {
	groups, err := netboxconfig.CollectGroups(cfg)
	if err != nil {
		return err
	}

	config := map[string]map[string]bool{} // runlevel -> serviceName -> alreadySeen

	for _, g := range groups {
		var groupCfg map[string][]string
		if err := json.Unmarshal(g, &groupCfg); err != nil {
			return err
		}

		for runlevel, services := range groupCfg {
			// setup runlevel if it hasn't been seen before
			runlevelCfg, exists := config[runlevel]
			if !exists {
				runlevelCfg = map[string]bool{}
				config[runlevel] = runlevelCfg
			}

			for _, s := range services {
				// Add service to runlevel if it doesn't already exist
				if _, exists := config[runlevel][s]; !exists {
					config[runlevel][s] = true
					if err := ovl.AddRCLink(s, runlevel); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
