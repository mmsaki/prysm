package sync

import (
	"context"
	"fmt"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
)

// blobSidecarByRootRPCHandler handles the /eth2/beacon_chain/req/blob_sidecars_by_root/1/ RPC request.
// spec: https://github.com/ethereum/consensus-specs/blob/a7e45db9ac2b60a33e144444969ad3ac0aae3d4c/specs/deneb/p2p-interface.md#blobsidecarsbyroot-v1
func (s *Service) lightClientBootstrapRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.lightClientBootstrapRPCHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()

	// TODO: What should we log?
	log := log.WithField("handler", p2p.LightClientBootstrapName[1:]) // slice the leading slash off the name var

	SetRPCStreamDeadlines(stream)
	if err := s.rateLimiter.validateRequest(stream, 1); err != nil {
		return err
	}
	s.rateLimiter.add(stream, 1)

	rawMsg, ok := msg.(*types.LightClientBootstrapReq)
	if !ok {
		return fmt.Errorf("message is not type %T", &types.LightClientBootstrapReq{})
	}
	blkRoot := *rawMsg

	bootstrap, err := s.cfg.beaconDB.LightClientBootstrap(ctx, blkRoot)
	if err != nil {
		s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
		return err
	}
	if err = WriteLightClientBootstrapChunk(stream, s.cfg.clock, s.cfg.p2p.Encoding(), bootstrap); err != nil {
		return err
	}

	closeStream(stream, log)
	return nil
}

func (s *Service) lightClientUpdatesByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	return nil
}

func (s *Service) lightClientFinalityUpdateRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	return nil
}

func (s *Service) lightClientOptimisticUpdateRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	return nil
}
