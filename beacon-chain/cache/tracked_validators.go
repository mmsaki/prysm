package cache

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

var validatorInactivityThreshold = 5 // 5 epochs

type TrackedValidator struct {
	Active       bool
	FeeRecipient primitives.ExecutionAddress
	Index        primitives.ValidatorIndex
	LastUpdated  time.Time
}

type TrackedValidatorsCache struct {
	sync.RWMutex
	trackedValidators               map[primitives.ValidatorIndex]TrackedValidator
	trackedValidatorsNum            uint64
	trackedValidatorsNumLastUpdated uint64
	trackedValidatorsMetric         prometheus.GaugeFunc
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
	t.updateTackedValidatorsNum()
}

func (t *TrackedValidatorsCache) updateTackedValidatorsNum() {
	epochTime := uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().SecondsPerSlot
	if t.trackedValidatorsNumLastUpdated != 0 && t.trackedValidatorsNumLastUpdated+epochTime < uint64(time.Now().Unix()) {
		// too early to update again
		return
	}
	num := 0
	for _, value := range t.trackedValidators {
		if value.LastUpdated.Unix()+int64(validatorInactivityThreshold)*int64(epochTime) < time.Now().Unix() {
			// validator expired
			// TODO: should we prune?
			continue
		}
		num++
	}
	t.trackedValidatorsNum = uint64(num)
	t.trackedValidatorsNumLastUpdated = uint64(time.Now().Unix())
}

func (t *TrackedValidatorsCache) GetTrackedValidatorsNum() uint64 {
	t.RLock()
	defer t.RUnlock()
	return t.trackedValidatorsNum
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

func (t *TrackedValidatorsCache) RegisterTrackedValidatorMetric() error {
	t.Lock()
	defer t.Unlock()
	t.trackedValidatorsMetric = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "tracked_validator_count",
			Help: "The total number of validators tracked by trackedValidatorsCache in the beacon node. This is updated at intervals via the push proposer settings API endpoint.",
		},
		func() float64 { return float64(t.GetTrackedValidatorsNum()) },
	)
	return prometheus.Register(t.trackedValidatorsMetric)
}

func (t *TrackedValidatorsCache) UnregisterTrackedValidatorMetric() {
	t.RLock()
	defer t.RUnlock()
	prometheus.Unregister(t.trackedValidatorsMetric)
}
