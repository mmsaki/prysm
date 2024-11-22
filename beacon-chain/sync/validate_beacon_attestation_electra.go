package sync

import (
	"context"
	"errors"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// validateCommitteeIndexElectra implements the following checks from the spec:
//   - [REJECT] attestation.data.index == 0
func validateCommitteeIndexElectra(ctx context.Context, a ethpb.Att) (primitives.CommitteeIndex, pubsub.ValidationResult, error) {
	_, span := trace.StartSpan(ctx, "sync.validateCommitteeIndexElectra")
	defer span.End()

	if a.GetData().CommitteeIndex != 0 {
		return 0, pubsub.ValidationReject, errors.New("attestation data's committee index must be zero")
	}
	committeeIndex, err := a.GetCommitteeIndex()
	if err != nil {
		return 0, pubsub.ValidationReject, err
	}
	return committeeIndex, pubsub.ValidationAccept, nil
}
