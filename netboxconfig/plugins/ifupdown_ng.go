package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/netip"

	"code.crute.us/mcrute/netboot-server/netboxconfig"
)

func init() {
	netboxconfig.RegisterConfigFunc("ifupdown_ng", generateIfUpdown)
}

type ifupdownNgConfig struct {
	ConfigureLoopback  bool `json:"configure_loopback"`
	GenerateInterfaces bool `json:"generate_interfaces"`
	ForcePrimaryDhcp   bool `json:"force_primary_dhcp"`
}

func generateIfUpdown(ctx context.Context, ovl *netboxconfig.APKOVL, cfg json.RawMessage, hostCfg *netboxconfig.RawConfig) error {
	var config ifupdownNgConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return err
	}

	contents := &bytes.Buffer{}

	if config.ConfigureLoopback {
		contents.WriteString("auto lo\n")
		contents.WriteString("iface lo\n")
		contents.WriteString("    use loopback\n")
		contents.WriteString("\n")
	}

	if config.GenerateInterfaces {
		for i, iface := range hostCfg.Interfaces {
			contents.WriteString(fmt.Sprintf("auto %s\n", iface.Name))
			contents.WriteString(fmt.Sprintf("iface %s\n", iface.Name))
			contents.WriteString("    use ipv6-ra\n")

			if i == 0 && config.ForcePrimaryDhcp {
				contents.WriteString("    use dhcp\n")
				contents.WriteString(fmt.Sprintf("    hostname %s\n", hostCfg.Name))
			} else {
				contents.WriteString(fmt.Sprintf("    hostname %s\n", hostCfg.Name))
				haveGateway := false
				for _, address := range iface.IPAddresses {
					if !haveGateway {
						prefix, err := netip.ParsePrefix(address.Address)
						if err != nil {
							return err
						}
						// TODO: Assumes gateway is the top of the network + 1, is this always right?
						gateway := prefix.Masked().Addr().Next()
						contents.WriteString(fmt.Sprintf("    gateway %s\n", gateway.String()))
						haveGateway = true
					}
					contents.WriteString(fmt.Sprintf("    address %s\n", address.Address))
				}
			}
		}
	}

	return ovl.AddStringFile(contents.String(), "etc/network/interfaces", 0644)
}
