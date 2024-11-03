package sync

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/math"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
)

// lightClientBootstrapRPCHandler handles the /eth2/beacon_chain/req/light_client_bootstrap/1/ RPC request.
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
		tracing.AnnotateError(span, err)
		return err
	}

	SetStreamWriteDeadline(stream, defaultWriteDuration)
	if err = WriteLightClientBootstrapChunk(stream, s.cfg.clock, s.cfg.p2p.Encoding(), bootstrap); err != nil {
		s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
		tracing.AnnotateError(span, err)
		return err
	}

	closeStream(stream, log)
	return nil
}

// lightClientBootstrapRPCHandler handles the /eth2/beacon_chain/req/light_client_updates_by_range/1/ RPC request.
func (s *Service) lightClientUpdatesByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.lightClientUpdatesByRangeRPCHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()

	// TODO: What should we log?
	log := log.WithField("handler", p2p.LightClientUpdatesByRangeName[1:]) // slice the leading slash off the name var

	SetRPCStreamDeadlines(stream)
	if err := s.rateLimiter.validateRequest(stream, 1); err != nil {
		return err
	}
	s.rateLimiter.add(stream, 1)

	rawMsg, ok := msg.(*types.LightClientUpdatesByRangeReq)
	if !ok {
		return fmt.Errorf("message is not type %T", &types.LightClientUpdatesByRangeReq{})
	}
	r := *rawMsg

	if r.Count > params.BeaconConfig().MaxRequestLightClientUpdates {
		r.Count = params.BeaconConfig().MaxRequestLightClientUpdates
	}
	endPeriod, ok := math.SafeAdd(r.StartPeriod, r.Count-1)
	if !ok {
		err := errors.Wrap(types.ErrInvalidRequest, "end period overflows")
		s.writeErrorResponseToStream(responseCodeInvalidRequest, err.Error(), stream)
		s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		tracing.AnnotateError(span, err)
		return err
	}

	updates, err := s.cfg.beaconDB.LightClientUpdates(ctx, r.StartPeriod, endPeriod)
	if err != nil {
		s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
		tracing.AnnotateError(span, err)
		return err
	}
	for _, u := range updates {
		SetStreamWriteDeadline(stream, defaultWriteDuration)
		if err = WriteLightClientUpdateChunk(stream, s.cfg.clock, s.cfg.p2p.Encoding(), u); err != nil {
			s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
			tracing.AnnotateError(span, err)
			return err
		}
		s.rateLimiter.add(stream, 1)
	}

	closeStream(stream, log)
	return nil
}

// lightClientBootstrapRPCHandler handles the /eth2/beacon_chain/req/light_client_finality_update/1/ RPC request.
func (s *Service) lightClientFinalityUpdateRPCHandler(ctx context.Context, _ interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.lightClientFinalityUpdateRPCHandler")
	defer span.End()
	_, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()

	// TODO: What should we log?
	log := log.WithField("handler", p2p.LightClientFinalityUpdateName[1:]) // slice the leading slash off the name var

	SetRPCStreamDeadlines(stream)
	if err := s.rateLimiter.validateRequest(stream, 1); err != nil {
		return err
	}
	s.rateLimiter.add(stream, 1)

	if s.lcStore.LastLCFinalityUpdate == nil {
		s.writeErrorResponseToStream(responseCodeResourceUnavailable, types.ErrResourceUnavailable.Error(), stream)
		return nil
	}

	SetStreamWriteDeadline(stream, defaultWriteDuration)
	if err := WriteLightClientFinalityUpdateChunk(stream, s.cfg.clock, s.cfg.p2p.Encoding(), s.lcStore.LastLCFinalityUpdate); err != nil {
		s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
		tracing.AnnotateError(span, err)
		return err
	}

	closeStream(stream, log)
	return nil
}

// lightClientOptimisticUpdateRPCHandler handles the /eth2/beacon_chain/req/light_client_optimistic_update/1/ RPC request.
func (s *Service) lightClientOptimisticUpdateRPCHandler(ctx context.Context, _ interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.lightClientOptimisticUpdateRPCHandler")
	defer span.End()
	_, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()

	// TODO: What should we log?
	log := log.WithField("handler", p2p.LightClientOptimisticUpdateName[1:]) // slice the leading slash off the name var

	SetRPCStreamDeadlines(stream)
	if err := s.rateLimiter.validateRequest(stream, 1); err != nil {
		return err
	}
	s.rateLimiter.add(stream, 1)

	if s.lcStore.LastLCOptimisticUpdate == nil {
		s.writeErrorResponseToStream(responseCodeResourceUnavailable, types.ErrResourceUnavailable.Error(), stream)
		return nil
	}

	SetStreamWriteDeadline(stream, defaultWriteDuration)
	if err := WriteLightClientOptimisticUpdateChunk(stream, s.cfg.clock, s.cfg.p2p.Encoding(), s.lcStore.LastLCOptimisticUpdate); err != nil {
		s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
		tracing.AnnotateError(span, err)
		return err
	}

	closeStream(stream, log)
	return nil
}
