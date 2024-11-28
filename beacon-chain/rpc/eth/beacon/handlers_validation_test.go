package beacon

import (
	"context"
	"testing"

	chainMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v5/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestValidateConsensus(t *testing.T) {
	ctx := context.Background()

	parentState, privs := util.DeterministicGenesisState(t, params.MinimalSpecConfig().MinGenesisActiveValidatorCount)
	parentBlock, err := util.GenerateFullBlock(parentState, privs, util.DefaultBlockGenConfig(), parentState.Slot())
	require.NoError(t, err)
	parentSbb, err := blocks.NewSignedBeaconBlock(parentBlock)
	require.NoError(t, err)
	st, err := transition.ExecuteStateTransition(ctx, parentState, parentSbb)
	require.NoError(t, err)
	block, err := util.GenerateFullBlock(st, privs, util.DefaultBlockGenConfig(), st.Slot())
	require.NoError(t, err)
	parentRoot, err := parentSbb.Block().HashTreeRoot()
	require.NoError(t, err)
	server := &Server{
		Blocker: &testutil.MockBlocker{RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{parentRoot: parentSbb}},
		Stater:  &testutil.MockStater{StatesByRoot: map[[32]byte]state.BeaconState{bytesutil.ToBytes32(parentBlock.Block.StateRoot): parentState}},
	}

	require.NoError(t, server.validateConsensus(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Phase0{
			Phase0: block,
		},
	}))
}

func TestValidateEquivocation(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(10))
		blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
		require.NoError(t, err)
		roblock, err := blocks.NewROBlockWithRoot(blk, bytesutil.ToBytes32([]byte("root")))
		require.NoError(t, err)
		fc := doublylinkedtree.New()
		require.NoError(t, fc.InsertNode(context.Background(), st, roblock))
		server := &Server{
			ForkchoiceFetcher: &chainMock.ChainService{ForkChoiceStore: fc},
		}
		blk.SetSlot(st.Slot() + 1)

		require.NoError(t, server.validateEquivocation(blk.Block()))
	})
	t.Run("block already exists", func(t *testing.T) {
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(10))
		blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
		require.NoError(t, err)
		blk.SetSlot(st.Slot())
		roblock, err := blocks.NewROBlockWithRoot(blk, bytesutil.ToBytes32([]byte("root")))
		require.NoError(t, err)

		fc := doublylinkedtree.New()
		require.NoError(t, fc.InsertNode(context.Background(), st, roblock))
		server := &Server{
			ForkchoiceFetcher: &chainMock.ChainService{ForkChoiceStore: fc},
		}
		err = server.validateEquivocation(blk.Block())
		assert.ErrorContains(t, "already exists", err)
		require.ErrorIs(t, err, errEquivocatedBlock)
	})
}
