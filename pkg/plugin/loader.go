package plugin

import (
	"fmt"
	"os/exec"

	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	"github.com/hashicorp/go-plugin"
)

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ROADY_PLUGIN",
	MagicCookieValue: "roady",
}

var PluginMap = map[string]plugin.Plugin{
	"syncer": &domainPlugin.SyncerPlugin{},
}

type Loader struct {
	plugins map[string]*plugin.Client
}

func NewLoader() *Loader {
	return &Loader{
		plugins: make(map[string]*plugin.Client),
	}
}

func (l *Loader) Load(path string) (domainPlugin.Syncer, error) {
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         PluginMap,
		Cmd:             exec.Command(path),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
		},
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to create plugin client: %w", err)
	}

	raw, err := rpcClient.Dispense("syncer")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense plugin: %w", err)
	}

	l.plugins[path] = client
	return raw.(domainPlugin.Syncer), nil
}

func (l *Loader) Cleanup() {
	for _, client := range l.plugins {
		client.Kill()
	}
}
