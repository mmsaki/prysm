package interfaces

import (
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// ReadOnlyValidator defines a struct which only has read access to validator methods.
type ReadOnlyValidator interface {
	EffectiveBalance() uint64
	ActivationEligibilityEpoch() primitives.Epoch
	ActivationEpoch() primitives.Epoch
	WithdrawableEpoch() primitives.Epoch
	ExitEpoch() primitives.Epoch
	PublicKey() [fieldparams.BLSPubkeyLength]byte
	GetWithdrawalCredentials() []byte
	Copy() *ethpb.Validator
	Slashed() bool
	IsNil() bool
}

type WithWithdrawalCredentials interface {
	GetWithdrawalCredentials() []byte
}
