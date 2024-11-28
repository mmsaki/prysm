package beacon

import (
	"context"
	"net/http"

	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func (s *Server) validateBroadcast(ctx context.Context, r *http.Request, blk *eth.GenericSignedBeaconBlock) error {
	switch r.URL.Query().Get(broadcastValidationQueryParam) {
	case broadcastValidationConsensus:
		if err := s.validateConsensus(ctx, blk); err != nil {
			return errors.Wrap(err, "consensus validation failed")
		}
	case broadcastValidationConsensusAndEquivocation:
		b, err := blocks.NewSignedBeaconBlock(blk.Block)
		if err != nil {
			return errors.Wrapf(err, "could not create signed beacon block")
		}
		if err = s.validateConsensus(r.Context(), blk); err != nil {
			return errors.Wrap(err, "consensus validation failed")
		}
		if err = s.validateEquivocation(b.Block()); err != nil {
			return errors.Wrap(err, "equivocation validation failed")
		}
	default:
		return nil
	}
	return nil
}

func (s *Server) validateBlobSidecars(blk interfaces.SignedBeaconBlock, blobs [][]byte, proofs [][]byte) error {
	if blk.Version() < version.Deneb {
		return nil
	}
	kzgs, err := blk.Block().Body().BlobKzgCommitments()
	if err != nil {
		return errors.Wrap(err, "could not get blob kzg commitments")
	}
	if len(blobs) != len(proofs) || len(blobs) != len(kzgs) {
		return errors.New("number of blobs, proofs, and commitments do not match")
	}
	for i, blob := range blobs {
		if err := kzg4844.VerifyBlobProof(kzg4844.Blob(blob), kzg4844.Commitment(kzgs[i]), kzg4844.Proof(proofs[i])); err != nil {
			return errors.Wrap(err, "could not verify blob proof")
		}
	}
	return nil
}

func (s *Server) validateConsensus(ctx context.Context, b *eth.GenericSignedBeaconBlock) error {
	blk, err := blocks.NewSignedBeaconBlock(b.Block)
	if err != nil {
		return errors.Wrapf(err, "could not create signed beacon block")
	}

	parentBlockRoot := blk.Block().ParentRoot()
	parentBlock, err := s.Blocker.Block(ctx, parentBlockRoot[:])
	if err != nil {
		return errors.Wrap(err, "could not get parent block")
	}

	if err := blocks.BeaconBlockIsNil(blk); err != nil {
		return errors.Wrap(err, "could not validate block")
	}

	parentStateRoot := parentBlock.Block().StateRoot()
	parentState, err := s.Stater.State(ctx, parentStateRoot[:])
	if err != nil {
		return errors.Wrap(err, "could not get parent state")
	}
	_, err = transition.ExecuteStateTransition(ctx, parentState, blk)
	if err != nil {
		return errors.Wrap(err, "could not execute state transition")
	}

	var blobs [][]byte
	var proofs [][]byte
	switch {
	case blk.Version() == version.Electra:
		blobs = b.GetElectra().Blobs
		proofs = b.GetElectra().KzgProofs
	case blk.Version() == version.Deneb:
		blobs = b.GetDeneb().Blobs
		proofs = b.GetDeneb().KzgProofs
	default:
		return nil
	}

	if err := s.validateBlobSidecars(blk, blobs, proofs); err != nil {
		return err
	}

	return nil
}

func (s *Server) validateEquivocation(blk interfaces.ReadOnlyBeaconBlock) error {
	if s.ForkchoiceFetcher.HighestReceivedBlockSlot() == blk.Slot() {
		return errors.Wrapf(errEquivocatedBlock, "block for slot %d already exists in fork choice", blk.Slot())
	}
	return nil
}
