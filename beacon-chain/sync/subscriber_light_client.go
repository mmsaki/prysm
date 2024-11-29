package sync

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"google.golang.org/protobuf/proto"
)

func (s *Service) lightClientFinalityUpdateSubscriber(_ context.Context, msg proto.Message) error {
	update, ok := msg.(interfaces.LightClientFinalityUpdate)
	if !ok {
		return fmt.Errorf("message type %T is not a light client finality update", msg)
	}

	log.Info("LC: storing new finality update in p2p subscriber")
	s.lcStore.LastLCFinalityUpdate = update

	s.cfg.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.LightClientFinalityUpdate,
		Data: update,
	})

	return nil
}

func (s *Service) lightClientOptimisticUpdateSubscriber(_ context.Context, msg proto.Message) error {
	update, ok := msg.(interfaces.LightClientOptimisticUpdate)
	if !ok {
		return fmt.Errorf("message type %T is not a light client optimistic update", msg)
	}

	log.Info("LC: storing new optimistic update in p2p subscriber")
	s.lcStore.LastLCOptimisticUpdate = update

	s.cfg.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.LightClientOptimisticUpdate,
		Data: update,
	})

	return nil
}
