package plugins

import (
	"encoding/json"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
	mapset "github.com/deckarep/golang-set/v2"
)

func init() {
	netboxconfig.RegisterSimpleConfigFunc("alpine_repos", generateAlpineRepos)
}

func generateAlpineRepos(ovl *netboxconfig.APKOVL, cfg json.RawMessage) error {
	groups, err := netboxconfig.CollectGroups(cfg)
	if err != nil {
		return err
	}

	repos := []string{}
	seen := mapset.NewSet[string]()

	for _, g := range groups {
		var groupCfg []string
		if err := json.Unmarshal(g, &groupCfg); err != nil {
			return err
		}

		for _, repo := range groupCfg {
			if !seen.Contains(repo) {
				seen.Add(repo)
				repos = append(repos, repo)
			}
		}
	}

	return ovl.AddStringListFile(repos, "etc/apk/repositories", 0644)
}
