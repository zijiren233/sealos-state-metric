// Package all imports all collector packages to register their factories
package all

import (
	// Import all collectors to trigger their init() functions
	_ "github.com/zijiren233/sealos-state-metric/pkg/collector/cloudbalance"
	_ "github.com/zijiren233/sealos-state-metric/pkg/collector/domain"
	_ "github.com/zijiren233/sealos-state-metric/pkg/collector/imagepull"
	_ "github.com/zijiren233/sealos-state-metric/pkg/collector/lvm"
	_ "github.com/zijiren233/sealos-state-metric/pkg/collector/node"
	_ "github.com/zijiren233/sealos-state-metric/pkg/collector/zombie"
)
