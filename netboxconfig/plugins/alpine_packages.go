package plugins

import (
	"encoding/json"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
	mapset "github.com/deckarep/golang-set/v2"
)

func init() {
	netboxconfig.RegisterSimpleConfigFunc("alpine_packages", generateAlpinePackages)
}

func generateAlpinePackages(ovl *netboxconfig.APKOVL, cfg json.RawMessage) error {
	groups, err := netboxconfig.CollectGroups(cfg)
	if err != nil {
		return err
	}

	packages := mapset.NewSet[string]()

	for _, g := range groups {
		var groupCfg []string
		if err := json.Unmarshal(g, &groupCfg); err != nil {
			return err
		}

		for _, pkg := range groupCfg {
			packages.Add(pkg)
		}
	}

	return ovl.AddStringListFile(mapset.Sorted(packages), "etc/apk/world", 0644)
}
