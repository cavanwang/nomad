// +build !nonvidia

package catalog

import (
	"github.com/hashicorp/nomad/devices/gpu/nvidia"
	"github.com/hashicorp/nomad/devices/gpu/vfio"
)

// This file is where all builtin plugins should be registered in the catalog.
// Plugins with build restrictions should be placed in the appropriate
// register_XXX.go file.
func init() {
	Register(nvidia.PluginID, nvidia.PluginConfig)
	Register(vfio.PluginID, vfio.PluginConfig)
}
