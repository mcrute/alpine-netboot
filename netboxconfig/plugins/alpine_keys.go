package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
)

func init() {
	netboxconfig.RegisterConfigFunc("alpine_keys", generateAlpineKeys)
}

func generateAlpineKeys(ctx context.Context, ovl *netboxconfig.APKOVL, cfg json.RawMessage, _ *netboxconfig.RawConfig) error {
	groups, err := netboxconfig.CollectGroups(cfg)
	if err != nil {
		return err
	}

	for _, g := range groups {
		var groupCfg map[string]string
		if err := json.Unmarshal(g, &groupCfg); err != nil {
			return err
		}

		for name, key := range groupCfg {
			keyPath := fmt.Sprintf("etc/apk/keys/%s", name)

			if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
				if err := ovl.AddHTTPFile(ctx, key, keyPath, 0644); err != nil {
					return err
				}
			} else {
				if err := ovl.AddStringFile(key, keyPath, 0644); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
