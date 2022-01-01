package pythontype

import (
	"encoding/binary"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// rehash combines several hashes into one
func rehash(x ...FlatID) FlatID {
	var h uint64
	b := make([]byte, 8)
	for _, xi := range x {
		binary.LittleEndian.PutUint64(b, uint64(xi))
		h = spooky.Hash64Seed(b, h)
	}
	return FlatID(h)
}

// rehashValues combines a hash with the hashes of zero or more values
func rehashValues(ctx kitectx.CallContext, x FlatID, vs ...Value) FlatID {
	var h uint64
	b := make([]byte, 8)
	for _, v := range vs {
		binary.LittleEndian.PutUint64(b, uint64(hash(ctx, v)))
		h = spooky.Hash64Seed(b, h)
	}
	binary.LittleEndian.PutUint64(b, h)
	return FlatID(spooky.Hash64Seed(b, uint64(x)))
}

// rehashBytes combines a hash with the hash of a byte slice
func rehashBytes(x FlatID, b []byte) FlatID {
	return FlatID(spooky.Hash64Seed(b, uint64(x)))
}

// These constants ensure that the hash of each value is repeatable but unique.
// The numbers are randomly generated.
const (
	saltDict                = 6852785620859
	saltList                = 2608058625550
	saltSet                 = 2569784136639
	saltTuple               = 9785314953969
	saltProperty            = 4651918196213
	saltPropertyUpdater     = 1531549486451
	saltUnion               = 1085740485675
	saltNone                = 6758959635298
	saltBool                = 1935468612388
	saltTrue                = 1123535697898
	saltFalse               = 7451123546465
	saltInt                 = 4663644334535
	saltLong                = 1089457408548
	saltFloat               = 9843092804544
	saltComplex             = 6573865046781
	saltStr                 = 6087650786584
	saltFunc                = 6075460587450
	saltType                = 9005419490459
	saltModule              = 3569715369783
	saltBoundMethod         = 6950650687583
	saltExternal            = 5768797545612
	saltExternalInstance    = 7018347391875
	saltExternalReturnValue = 8916790813476
	saltCounter             = 5577006791947
	saltOrderedDict         = 1543039099823
	saltDefaultDict         = 6640668014774
	saltDeque               = 2244708090865
	saltQueue               = 7414159922357
	saltLifoQueue           = 3305628230121
	saltPriorityQueue       = 8475284246537
	saltNamedTupleType      = 4151935814835
	saltNamedTupleInstance  = 3363776116195
	saltManager             = 4554545203516
	saltQuerySet            = 6848946516548
	saltOptions             = 6515656151561
	saltSuper               = 8621382289131
	saltKwargDict           = 9406748134985
	saltGenerator           = 7454165149497
)
