package cache

import (
	"sync"
	"time"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

type TrackedValidator struct {
	Active       bool
	FeeRecipient primitives.ExecutionAddress
	Index        primitives.ValidatorIndex
	LastUpdated  time.Time
}

type TrackedValidatorsCache struct {
	sync.RWMutex
	trackedValidators map[primitives.ValidatorIndex]TrackedValidator
}

func NewTrackedValidatorsCache() *TrackedValidatorsCache {
	return &TrackedValidatorsCache{
		trackedValidators: make(map[primitives.ValidatorIndex]TrackedValidator),
	}
}

func (t *TrackedValidatorsCache) Validator(index primitives.ValidatorIndex) (TrackedValidator, bool) {
	t.RLock()
	defer t.RUnlock()
	val, ok := t.trackedValidators[index]
	return val, ok
}

func (t *TrackedValidatorsCache) Set(val TrackedValidator) {
	t.Lock()
	defer t.Unlock()
	t.trackedValidators[val.Index] = val
}

func (t *TrackedValidatorsCache) Prune() {
	t.Lock()
	defer t.Unlock()
	t.trackedValidators = make(map[primitives.ValidatorIndex]TrackedValidator)
}

func (t *TrackedValidatorsCache) Validating() bool {
	t.RLock()
	defer t.RUnlock()
	return len(t.trackedValidators) > 0
}

func (t *TrackedValidatorsCache) Size() int {
	t.RLock()
	defer t.RUnlock()
	return len(t.trackedValidators)
}
