package netboxconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"code.crute.us/mcrute/golib/clients/netbox/v4"
)

type (
	simpleHandlerFunc func(*APKOVL, json.RawMessage) error
	handlerFunc       func(context.Context, *APKOVL, json.RawMessage, *RawConfig) error

	configHandler interface {
		Generate(ctx context.Context, ovl *APKOVL, cfg json.RawMessage, rawCfg *RawConfig) error
	}
)

type configAdapter struct {
	simpleFunc  simpleHandlerFunc
	handlerFunc handlerFunc
}

func (a configAdapter) Generate(ctx context.Context, ovl *APKOVL, cfg json.RawMessage, rawCfg *RawConfig) error {
	if a.simpleFunc != nil {
		return a.simpleFunc(ovl, cfg)
	}
	if a.handlerFunc != nil {
		return a.handlerFunc(ctx, ovl, cfg, rawCfg)
	}
	// This should be impossible unless there's a bug
	return errors.New("Handler function not configured in adapter")
}

var configPlugins = map[string]configHandler{}

func RegisterConfigPlugin(name string, handler configHandler) {
	if _, exists := configPlugins[name]; exists {
		panic(fmt.Sprintf("Unable to add config plugin %s because it already exists", name))
	}
	configPlugins[name] = handler
}

func RegisterSimpleConfigFunc(name string, handler simpleHandlerFunc) {
	RegisterConfigPlugin(name, configAdapter{simpleFunc: handler})
}

func RegisterConfigFunc(name string, handler handlerFunc) {
	RegisterConfigPlugin(name, configAdapter{handlerFunc: handler})
}

type ConfigCoordinator struct {
	NetboxClient    *netbox.BasicNetboxClient
	DefaultConfigId int
}

func (c *ConfigCoordinator) MacExists(ctx context.Context, mac string) (bool, error) {
	count, err := netboxGetInterfaceCountForMac(ctx, c.NetboxClient, mac)
	return count == 1, err
}

func (c *ConfigCoordinator) GenerateDefault(ctx context.Context, out io.Writer) error {
	cfg, err := netboxGetConfigContext(ctx, c.NetboxClient, c.DefaultConfigId)
	if err != nil {
		return err
	}

	ovl := NewAPKOVLFromWriter(out)
	defer ovl.Close()

	for k, v := range cfg {
		plugin, pluginExists := configPlugins[k]
		if pluginExists {
			// Note that not all plugins can work here because there is no
			// RawConfig, only a default config. This will fail with an error if
			// anyone configures the default to have a plugin that requires a
			// RawConfig.
			if err := plugin.Generate(ctx, ovl, v, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

// TODO: Chainload into a fully working system (mount data drives, start jobs)
func (c *ConfigCoordinator) GenerateForMac(ctx context.Context, mac string, out io.Writer) error {
	cfg, err := netboxGetHost(ctx, c.NetboxClient, mac)
	if err != nil {
		return err
	}

	ovl := NewAPKOVLFromWriter(out)
	defer ovl.Close()

	hostnamePlugin := configPlugins["hostname"]
	if err := hostnamePlugin.Generate(ctx, ovl, nil, cfg); err != nil {
		return err
	}

	for k, v := range cfg.ConfigContext {
		// Ignores plugins that don't exist because config_context can be used
		// for many other things.
		plugin, pluginExists := configPlugins[k]
		if pluginExists {
			if err := plugin.Generate(ctx, ovl, v, cfg); err != nil {
				return err
			}
		}
	}

	return nil
}
