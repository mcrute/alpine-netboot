package plugins

import (
	"encoding/json"
	"strings"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
)

func init() {
	netboxconfig.RegisterSimpleConfigFunc("doas", generateDoas)
}

type doasConfig struct {
	Action   string   `json:"action"`
	Options  []string `json:"options"`
	Identity string   `json:"identity"`
	Target   *string  `json:"as"`
	Command  *string  `json:"command"`
	Args     []string `json:"args"`
}

func generateDoas(ovl *netboxconfig.APKOVL, cfg json.RawMessage) error {
	var config []doasConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return err
	}

	lines := make([]string, len(config))

	for i, c := range config {
		parts := []string{
			c.Action,
			strings.Join(c.Options, " "),
			c.Identity,
		}

		if c.Target != nil {
			parts = append(parts, "as", *c.Target)
		}

		if c.Command != nil {
			parts = append(parts, "cmd", *c.Command)
		}

		if c.Args != nil && len(c.Args) > 0 {
			parts = append(parts, "args")
			parts = append(parts, c.Args...)
		}

		lines[i] = strings.Join(parts, " ")
	}

	return ovl.AddStringListFile(lines, "etc/doas.d/local.conf", 0644)
}
