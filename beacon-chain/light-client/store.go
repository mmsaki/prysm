package light_client

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
)

type Store struct {
	// lcFinalityUpdateLock   sync.Mutex
	LastLCFinalityUpdate interfaces.LightClientFinalityUpdate
	// lcOptimisticUpdateLock sync.Mutex
	LastLCOptimisticUpdate interfaces.LightClientOptimisticUpdate
}
