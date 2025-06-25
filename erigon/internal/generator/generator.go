package generator

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	"math/rand"
)

const (
	RecordMean     = 160
	KeyLen         = 32
	BatchSoftBytes = 256 << 20
	ringCap        = 10_000_000
)

type Operation struct {
	Kind  byte // 0=put,1=update,2=delete
	Key   []byte
	Value []byte
}

type Generator struct {
	rng      *rand.Rand
	existing [][]byte
}

func New() *Generator {
	var seed int64
	_ = binary.Read(crypto_rand.Reader, binary.LittleEndian, &seed)
	if seed == 0 {
		seed = 1
	}
	return &Generator{rng: rand.New(rand.NewSource(seed))}
}

func (g *Generator) NextOperation() Operation {
	p := g.rng.Float64()
	if len(g.existing) == 0 || p < 0.15 {
		key := make([]byte, KeyLen)
		g.rng.Read(key)
		val := make([]byte, RecordMean)
		g.rng.Read(val)
		g.addKey(key)
		return Operation{Kind: 0, Key: key, Value: val}
	}
	if p < 0.95 {
		idx := g.rng.Intn(len(g.existing))
		key := g.existing[idx]
		val := make([]byte, RecordMean)
		g.rng.Read(val)
		return Operation{Kind: 1, Key: key, Value: val}
	}
	idx := g.rng.Intn(len(g.existing))
	key := g.existing[idx]
	g.existing[idx] = g.existing[len(g.existing)-1]
	g.existing = g.existing[:len(g.existing)-1]
	return Operation{Kind: 2, Key: key}
}

func (g *Generator) addKey(k []byte) {
	if len(g.existing) < ringCap {
		g.existing = append(g.existing, k)
		return
	}
	idx := g.rng.Intn(ringCap)
	g.existing[idx] = k
}
