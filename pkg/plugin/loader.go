package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	goplugin "github.com/hashicorp/go-plugin"
)

var HandshakeConfig = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ROADY_PLUGIN",
	MagicCookieValue: "roady",
}

var PluginMap = map[string]goplugin.Plugin{
	"syncer": &domainPlugin.SyncerPlugin{},
}

type Loader struct {
	plugins map[string]*goplugin.Client
}

func NewLoader() *Loader {
	return &Loader{
		plugins: make(map[string]*goplugin.Client),
	}
}

func (l *Loader) Load(path string) (domainPlugin.Syncer, error) {
	// Validate plugin path before execution
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin not found: %s", absPath)
		}
		return nil, fmt.Errorf("cannot access plugin: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("plugin path is a directory: %s", absPath)
	}

	// Check executable permission on Unix systems
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			return nil, fmt.Errorf("plugin is not executable: %s", absPath)
		}
	}

	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         PluginMap,
		Cmd:             exec.Command(path),
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolNetRPC,
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
