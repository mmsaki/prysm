package cache

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

// TODO: comment code and see if tests fail

func TestAdd(t *testing.T) {
	k, err := blst.RandKey()
	require.NoError(t, err)
	sig := k.Sign([]byte{'X'})

	t.Run("new ID", func(t *testing.T) {
		t.Run("first ID ever", func(t *testing.T) {
			c := NewAttestationCache()

			att := &ethpb.Attestation{
				Data:            &ethpb.AttestationData{Slot: 123, BeaconBlockRoot: make([]byte, 32), Source: &ethpb.Checkpoint{Root: make([]byte, 32)}, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}},
				AggregationBits: bitfield.NewBitlist(8),
				Signature:       sig.Marshal(),
			}
			id, err := attestation.NewId(att, attestation.Data)
			require.NoError(t, err)
			require.NoError(t, c.Add(att))

			require.Equal(t, 1, len(c.atts))
			group, ok := c.atts[id]
			require.Equal(t, true, ok)
			assert.Equal(t, primitives.Slot(123), group.slot)
			require.Equal(t, 1, len(group.atts))
			assert.Equal(t, group.atts[0], att)
		})
		t.Run("other ID exists", func(t *testing.T) {
			c := NewAttestationCache()
			existingAtt := &ethpb.Attestation{
				Data:            &ethpb.AttestationData{BeaconBlockRoot: make([]byte, 32), Source: &ethpb.Checkpoint{Root: make([]byte, 32)}, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}},
				AggregationBits: bitfield.NewBitlist(8),
				Signature:       sig.Marshal(),
			}
			existingId, err := attestation.NewId(existingAtt, attestation.Data)
			require.NoError(t, err)
			c.atts[existingId] = &attGroup{slot: existingAtt.Data.Slot, atts: []ethpb.Att{existingAtt}}

			att := &ethpb.Attestation{
				Data:            &ethpb.AttestationData{Slot: 123, BeaconBlockRoot: make([]byte, 32), Source: &ethpb.Checkpoint{Root: make([]byte, 32)}, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}},
				AggregationBits: bitfield.NewBitlist(8),
				Signature:       sig.Marshal(),
			}
			id, err := attestation.NewId(att, attestation.Data)
			require.NoError(t, err)
			require.NoError(t, c.Add(att))

			require.Equal(t, 2, len(c.atts))
			group, ok := c.atts[id]
			require.Equal(t, true, ok)
			assert.Equal(t, primitives.Slot(123), group.slot)
			require.Equal(t, 1, len(group.atts))
			assert.Equal(t, group.atts[0], att)
		})
	})
	t.Run("aggregated", func(t *testing.T) {
		c := NewAttestationCache()
		existingAtt := &ethpb.Attestation{
			Data:            &ethpb.AttestationData{Slot: 123, BeaconBlockRoot: make([]byte, 32), Source: &ethpb.Checkpoint{Root: make([]byte, 32)}, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}},
			AggregationBits: bitfield.NewBitlist(8),
			Signature:       sig.Marshal(),
		}
		id, err := attestation.NewId(existingAtt, attestation.Data)
		require.NoError(t, err)
		c.atts[id] = &attGroup{slot: existingAtt.Data.Slot, atts: []ethpb.Att{existingAtt}}

		att := &ethpb.Attestation{
			Data:            &ethpb.AttestationData{Slot: 123, BeaconBlockRoot: make([]byte, 32), Source: &ethpb.Checkpoint{Root: make([]byte, 32)}, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}},
			AggregationBits: bitfield.NewBitlist(8),
			Signature:       sig.Marshal(),
		}
		att.AggregationBits.SetBitAt(0, true)
		att.AggregationBits.SetBitAt(1, true)
		require.NoError(t, c.Add(att))

		require.Equal(t, 1, len(c.atts))
		group, ok := c.atts[id]
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), group.slot)
		require.Equal(t, 2, len(group.atts))
		assert.Equal(t, group.atts[0], existingAtt)
		assert.Equal(t, group.atts[1], att)
	})
	t.Run("unaggregated - existing bit", func(t *testing.T) {
		c := NewAttestationCache()
		existingAtt := &ethpb.Attestation{
			Data:            &ethpb.AttestationData{Slot: 123, BeaconBlockRoot: make([]byte, 32), Source: &ethpb.Checkpoint{Root: make([]byte, 32)}, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}},
			AggregationBits: bitfield.NewBitlist(8),
			Signature:       sig.Marshal(),
		}
		existingAtt.AggregationBits.SetBitAt(0, true)
		id, err := attestation.NewId(existingAtt, attestation.Data)
		require.NoError(t, err)
		c.atts[id] = &attGroup{slot: existingAtt.Data.Slot, atts: []ethpb.Att{existingAtt}}

		att := &ethpb.Attestation{
			Data:            &ethpb.AttestationData{Slot: 123, BeaconBlockRoot: make([]byte, 32), Source: &ethpb.Checkpoint{Root: make([]byte, 32)}, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}},
			AggregationBits: bitfield.NewBitlist(8),
			Signature:       sig.Marshal(),
		}
		att.AggregationBits.SetBitAt(0, true)
		require.NoError(t, c.Add(att))

		require.Equal(t, 1, len(c.atts))
		group, ok := c.atts[id]
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), group.slot)
		require.Equal(t, 1, len(group.atts))
		assert.DeepEqual(t, []int{0}, group.atts[0].GetAggregationBits().BitIndices())
	})
	t.Run("unaggregated - new bit", func(t *testing.T) {
		c := NewAttestationCache()
		existingAtt := &ethpb.Attestation{
			Data:            &ethpb.AttestationData{Slot: 123, BeaconBlockRoot: make([]byte, 32), Source: &ethpb.Checkpoint{Root: make([]byte, 32)}, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}},
			AggregationBits: bitfield.NewBitlist(8),
			Signature:       sig.Marshal(),
		}
		existingAtt.AggregationBits.SetBitAt(0, true)
		id, err := attestation.NewId(existingAtt, attestation.Data)
		require.NoError(t, err)
		c.atts[id] = &attGroup{slot: existingAtt.Data.Slot, atts: []ethpb.Att{existingAtt}}

		att := &ethpb.Attestation{
			Data:            &ethpb.AttestationData{Slot: 123, BeaconBlockRoot: make([]byte, 32), Source: &ethpb.Checkpoint{Root: make([]byte, 32)}, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}},
			AggregationBits: bitfield.NewBitlist(8),
			Signature:       sig.Marshal(),
		}
		att.AggregationBits.SetBitAt(1, true)
		require.NoError(t, c.Add(att))

		require.Equal(t, 1, len(c.atts))
		group, ok := c.atts[id]
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), group.slot)
		require.Equal(t, 1, len(group.atts))
		assert.DeepEqual(t, []int{0, 1}, group.atts[0].GetAggregationBits().BitIndices())
	})
}

func TestGetAll(t *testing.T) {
	c := NewAttestationCache()
	c.atts[bytesutil.ToBytes32([]byte("id1"))] = &attGroup{atts: []ethpb.Att{&ethpb.Attestation{}, &ethpb.Attestation{}}}
	c.atts[bytesutil.ToBytes32([]byte("id2"))] = &attGroup{atts: []ethpb.Att{&ethpb.Attestation{}}}

	assert.Equal(t, 3, len(c.GetAll()))
}

func TestCount(t *testing.T) {
	c := NewAttestationCache()
	c.atts[bytesutil.ToBytes32([]byte("id1"))] = &attGroup{atts: []ethpb.Att{&ethpb.Attestation{}, &ethpb.Attestation{}}}
	c.atts[bytesutil.ToBytes32([]byte("id2"))] = &attGroup{atts: []ethpb.Att{&ethpb.Attestation{}}}

	assert.Equal(t, 3, c.Count())
}

func TestDeleteCovered(t *testing.T) {
	t.Run("no matching group", func(t *testing.T) {

	})
	t.Run("covered atts deleted", func(t *testing.T) {

	})
	t.Run("last att in group deleted", func(t *testing.T) {

	})
}

func TestPruneBefore(t *testing.T) {

}

func TestAggregateIsRedundant(t *testing.T) {
	t.Run("no matching group", func(t *testing.T) {

	})
	t.Run("redundant", func(t *testing.T) {

	})
	t.Run("not redundant", func(t *testing.T) {

	})
}

func TestGetBySlotAndCommitteeIndex(t *testing.T) {

}
