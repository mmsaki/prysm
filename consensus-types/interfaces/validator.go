package interfaces

import (
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
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
	Proto() *ethpb.Validator
	Slashed() bool
	IsNil() bool
}

// ReadOnlyValidators defines a struct which only has read access to validators methods.
type ReadOnlyValidators interface {
	Validators() []*ethpb.Validator
	ValidatorsReadOnly() []ReadOnlyValidator
	ValidatorAtIndex(idx primitives.ValidatorIndex) (*ethpb.Validator, error)
	ValidatorAtIndexReadOnly(idx primitives.ValidatorIndex) (ReadOnlyValidator, error)
	ValidatorIndexByPubkey(key [fieldparams.BLSPubkeyLength]byte) (primitives.ValidatorIndex, bool)
	PublicKeys() ([][fieldparams.BLSPubkeyLength]byte, error)
	PubkeyAtIndex(idx primitives.ValidatorIndex) [fieldparams.BLSPubkeyLength]byte
	AggregateKeyFromIndices(idxs []uint64) (bls.PublicKey, error)
	NumValidators() int
	ReadFromEveryValidator(f func(idx int, val ReadOnlyValidator) error) error
}

type WithWithdrawalCredentials interface {
	GetWithdrawalCredentials() []byte
}
