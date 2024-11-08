package sync

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
)

func BenchmarkCacheKeyImplementations(b *testing.B) {

	b.Run("Old Cache Key Implementation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b := append(bytesutil.Bytes32(uint64(i)), bytesutil.Bytes32(uint64(i+10))...)
			b = append(b, bytesutil.SafeCopyBytes([]byte("random"))...)
			_ = string(b)
		}
	})

	b.Run("New Cache Key Implementation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = seenAttCacheKey(primitives.Slot(i), primitives.CommitteeIndex(i+10), []byte("random"))
		}
	})
}
