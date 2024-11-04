package lightclient

import (
	"reflect"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func HasRelevantSyncCommittee(update interfaces.LightClientUpdate) (bool, error) {
	if update.Version() >= version.Electra {
		branch, err := update.NextSyncCommitteeBranchElectra()
		if err != nil {
			return false, err
		}
		return !reflect.DeepEqual(branch, interfaces.LightClientSyncCommitteeBranchElectra{}), nil
	}
	branch, err := update.NextSyncCommitteeBranch()
	if err != nil {
		return false, err
	}
	return !reflect.DeepEqual(branch, interfaces.LightClientSyncCommitteeBranch{}), nil
}

func HasFinality(update interfaces.LightClientUpdate) (bool, error) {
	if update.Version() >= version.Electra {
		b, err := update.FinalityBranchElectra()
		if err != nil {
			return false, err
		}
		return !reflect.DeepEqual(b, interfaces.LightClientFinalityBranchElectra{}), nil
	}

	b, err := update.FinalityBranch()
	if err != nil {
		return false, err
	}
	return !reflect.DeepEqual(b, interfaces.LightClientFinalityBranch{}), nil
}

func IsBetterUpdate(newUpdate, oldUpdate interfaces.LightClientUpdate) (bool, error) {
	maxActiveParticipants := newUpdate.SyncAggregate().SyncCommitteeBits.Len()
	newNumActiveParticipants := newUpdate.SyncAggregate().SyncCommitteeBits.Count()
	oldNumActiveParticipants := oldUpdate.SyncAggregate().SyncCommitteeBits.Count()
	newHasSupermajority := newNumActiveParticipants*3 >= maxActiveParticipants*2
	oldHasSupermajority := oldNumActiveParticipants*3 >= maxActiveParticipants*2

	if newHasSupermajority != oldHasSupermajority {
		return newHasSupermajority, nil
	}
	if !newHasSupermajority && newNumActiveParticipants != oldNumActiveParticipants {
		return newNumActiveParticipants > oldNumActiveParticipants, nil
	}

	newUpdateAttestedHeaderBeacon := newUpdate.AttestedHeader().Beacon()
	oldUpdateAttestedHeaderBeacon := oldUpdate.AttestedHeader().Beacon()

	// Compare presence of relevant sync committee
	newHasRelevantSyncCommittee, err := HasRelevantSyncCommittee(newUpdate)
	if err != nil {
		return false, err
	}
	newHasRelevantSyncCommittee = newHasRelevantSyncCommittee &&
		(slots.SyncCommitteePeriod(slots.ToEpoch(newUpdateAttestedHeaderBeacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(newUpdate.SignatureSlot())))
	oldHasRelevantSyncCommittee, err := HasRelevantSyncCommittee(oldUpdate)
	if err != nil {
		return false, err
	}
	oldHasRelevantSyncCommittee = oldHasRelevantSyncCommittee &&
		(slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdateAttestedHeaderBeacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdate.SignatureSlot())))

	if newHasRelevantSyncCommittee != oldHasRelevantSyncCommittee {
		return newHasRelevantSyncCommittee, nil
	}

	// Compare indication of any finality
	newHasFinality, err := HasFinality(newUpdate)
	if err != nil {
		return false, err
	}
	oldHasFinality, err := HasFinality(oldUpdate)
	if err != nil {
		return false, err
	}
	if newHasFinality != oldHasFinality {
		return newHasFinality, nil
	}

	newUpdateFinalizedHeaderBeacon := newUpdate.FinalizedHeader().Beacon()
	oldUpdateFinalizedHeaderBeacon := oldUpdate.FinalizedHeader().Beacon()

	// Compare sync committee finality
	if newHasFinality {
		newHasSyncCommitteeFinality :=
			slots.SyncCommitteePeriod(slots.ToEpoch(newUpdateFinalizedHeaderBeacon.Slot)) ==
				slots.SyncCommitteePeriod(slots.ToEpoch(newUpdateAttestedHeaderBeacon.Slot))
		oldHasSyncCommitteeFinality :=
			slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdateFinalizedHeaderBeacon.Slot)) ==
				slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdateAttestedHeaderBeacon.Slot))

		if newHasSyncCommitteeFinality != oldHasSyncCommitteeFinality {
			return newHasSyncCommitteeFinality, nil
		}
	}

	// Tiebreaker 1: Sync committee participation beyond supermajority
	if newNumActiveParticipants != oldNumActiveParticipants {
		return newNumActiveParticipants > oldNumActiveParticipants, nil
	}

	// Tiebreaker 2: Prefer older data (fewer changes to best)
	if newUpdateAttestedHeaderBeacon.Slot != oldUpdateAttestedHeaderBeacon.Slot {
		return newUpdateAttestedHeaderBeacon.Slot < oldUpdateAttestedHeaderBeacon.Slot, nil
	}

	return newUpdate.SignatureSlot() < oldUpdate.SignatureSlot(), nil
}
