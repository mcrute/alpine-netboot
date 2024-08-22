package netboxconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"code.crute.us/mcrute/golib/clients/netbox/v4"
)

const hostQuery = `query {
  interface_list(filters: {mac_address: "%s"}) {
    device {
      name
      config_context
      custom_fields
      interfaces {
        name
        ip_addresses {
          address
        }
      }
      site {
        name
        custom_fields
      }
    }
  }
}`

type rawConfigEnvelope struct {
	Data struct {
		InterfaceList []struct {
			Device *RawConfig `json:"device"`
		} `json:"interface_list"`
	} `json:"data"`
}

type RawConfig struct {
	Name          string                     `json:"name"`
	ConfigContext map[string]json.RawMessage `json:"config_context"`
	CustomFields  struct {
		RootVaultPath string `json:"root_vault_path"`
	} `json:"custom_fields"`
	Interfaces []struct {
		Name        string `json:"name"`
		IPAddresses []struct {
			Address string `json:"address"`
		} `json:"ip_addresses"`
	} `json:"interfaces"`
	Site struct {
		Name         string `json:"name"`
		CustomFields struct {
			BaseFqdn string `json:"site_base_fqdn"`
		} `json:"custom_fields"`
	} `json:"site"`
}

func netboxGetHost(ctx context.Context, client *netbox.BasicNetboxClient, mac string) (*RawConfig, error) {
	_, err := net.ParseMAC(mac)
	if err != nil {
		return nil, fmt.Errorf("Invalid mac format: %w", err)
	}

	var m rawConfigEnvelope
	if err := client.Do(ctx, &netbox.NetboxGraphQLRequest{
		Query: fmt.Sprintf(hostQuery, mac),
	}, &m); err != nil {
		return nil, err
	}

	if len(m.Data.InterfaceList) != 1 {
		return nil, fmt.Errorf("No devices found for mac %s", mac)
	}

	return m.Data.InterfaceList[0].Device, nil
}

func netboxGetInterfaceCountForMac(ctx context.Context, client *netbox.BasicNetboxClient, mac string) (int, error) {
	q := netbox.NewNetboxGetRequest("/api/dcim/interfaces/")
	q.Add("mac_address", mac)

	out := &struct {
		Count int `json:"count"`
	}{}
	if err := client.Do(ctx, q, out); err != nil {
		return 0, err
	}
	return out.Count, nil
}

func netboxGetConfigContext(ctx context.Context, client *netbox.BasicNetboxClient, id int) (map[string]json.RawMessage, error) {
	q := netbox.NewNetboxGetRequest(fmt.Sprintf("/api/extras/config-contexts/%d", id))

	out := &struct {
		Data map[string]json.RawMessage `json:"data"`
	}{}
	if err := client.Do(ctx, q, out); err != nil {
		return nil, err
	}
	return out.Data, nil
}
