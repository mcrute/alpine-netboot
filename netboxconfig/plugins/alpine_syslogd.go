package plugins

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
	"github.com/dustin/go-humanize"
)

func init() {
	netboxconfig.RegisterSimpleConfigFunc("alpine_syslogd", generateAlpineSyslogd)
}

type syslogConfig struct {
	KeepNRotated          *int `json:"keep_n_rotated"`
	MaxSize               *int `json:"max_size"`
	StripClientTimestamps bool `json:"strip_client_timestamps"`
}

func generateAlpineSyslogd(ovl *netboxconfig.APKOVL, cfg json.RawMessage) error {
	var config syslogConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return err
	}

	args := []string{}
	out := []string{}

	if config.StripClientTimestamps {
		args = append(args, "-t")
	}

	if config.KeepNRotated != nil {
		out = append(out, fmt.Sprintf("# Keep max %d rotated logs", *config.KeepNRotated))
		args = append(args, "-b", strconv.Itoa(*config.KeepNRotated))
	}

	if config.MaxSize != nil {
		out = append(out, fmt.Sprintf("# Rotate logs at size %s", humanize.Bytes(uint64(*config.MaxSize)*1000)))
		args = append(args, "-s", strconv.Itoa(*config.MaxSize))
	}

	out = append(out, fmt.Sprintf("SYSLOGD_OPTS=\"%s\"", strings.Join(args, " ")))

	return ovl.AddStringListFile(out, "etc/conf.d/syslog", 0644)
}
