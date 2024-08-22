package plugins

import (
	"encoding/json"
	"strconv"
	"strings"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
)

func init() {
	netboxconfig.RegisterSimpleConfigFunc("inittab", generateInittab)
}

type inittabEntry struct {
	Id        string `json:"id"`
	Runlevels []int  `json:"runlevels"`
	Action    string `json:"action"`
	Process   string `json:"process"`
}

func generateInittab(ovl *netboxconfig.APKOVL, cfg json.RawMessage) error {
	var config []inittabEntry
	if err := json.Unmarshal(cfg, &config); err != nil {
		return err
	}

	out := make([]string, len(config))
	for i, c := range config {
		// Convert runlevels to a list of strings
		runlevels := make([]string, len(c.Runlevels))
		for j, l := range c.Runlevels {
			runlevels[j] = strconv.Itoa(l)
		}

		out[i] = strings.Join([]string{
			c.Id,
			strings.Join(runlevels, ""),
			c.Action,
			c.Process,
		}, ":")
	}

	return ovl.AddStringListFile(out, "etc/inittab", 0644)
}
