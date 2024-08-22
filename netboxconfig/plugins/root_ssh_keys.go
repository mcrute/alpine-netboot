package plugins

import (
	"encoding/json"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
)

func init() {
	netboxconfig.RegisterSimpleConfigFunc("root_ssh_keys", generateRootSSHKeys)
}

func generateRootSSHKeys(ovl *netboxconfig.APKOVL, cfg json.RawMessage) error {
	var keys []string
	if err := json.Unmarshal(cfg, &keys); err != nil {
		return err
	}
	return ovl.AddStringListFile(keys, "root/.ssh/authorized_keys", 0600)
}
