package uidgen

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUidGenerator(t *testing.T) {
	req := require.New(t)

	cfg := SnowflakeConfig
	gen, err := NewUidGenerator(cfg, 0)
	req.Nil(err)

	var prevID UniqueID

	for i := 0; i < (1<<cfg.CntLen)*3; i++ {
		id := gen.NextID()
		req.True(id > prevID)

		prevID = id
	}

	cfg.SrvID = 1
	gen, err = NewUidGenerator(cfg, prevID)
	req.Nil(err)

	id := gen.NextID()
	req.True(id > prevID)
	req.Equal(int64(1), gen.ServerID(id))
}

func TestUidEpoch(t *testing.T) {
	req := require.New(t)
	cfg := SnowflakeConfig
	gen, _ := NewUidGenerator(cfg, 0)

	now := time.Now().UnixNano()
	id1 := gen.FromUnixNano(now)
	req.Equal(now/1e7, gen.UnixNano(id1)/1e7)

	id1 = gen.NextID()
	id2 := gen.FromUnixNano(gen.UnixNano(id1))
	req.Equal(id1, id2)

	now = time.Now().Unix()
	id2 = gen.FromUnix(now)
	req.Equal(now, gen.Unix(id2))

	id2 = gen.FromUnix(gen.Unix(id1))
	req.Equal(gen.Unix(id1), gen.Unix(id2))
}

func TestServerID(t *testing.T) {
	req := require.New(t)

	cfg := SnowflakeConfig
	cfg.SrvLen = 1
	cfg.SrvID = 2

	gen, err := NewUidGenerator(cfg, 0)
	req.NotNil(err)

	cfg.SrvID = 1
	gen, err = NewUidGenerator(cfg, 0)
	req.Nil(err)

	id := gen.NextID()
	req.Equal(int64(1), gen.ServerID(id))
}

func TestUidToBase32(t *testing.T) {
	req := require.New(t)

	cfg := SnowflakeConfig
	gen, err := NewUidGenerator(cfg, 0)
	req.Nil(err)

	for i := 0; i < 100; i++ {
		id := gen.NextID()

		str := gen.ToBase32(id)
		decID, err := gen.FromBase32(str)
		req.Nil(err)
		req.Equal(id, decID)
	}

	cfg.EpochLen = 33
	cfg.TruncStr = true
	gen, err = NewUidGenerator(cfg, 0)
	req.Nil(err)

	id := gen.NextID()
	str := gen.ToBase32(id)
	decID, err := gen.FromBase32(str)
	req.Nil(err)
	req.Equal(id, decID)
}

func BenchmarkUnix(b *testing.B) {
	cfg := SnowflakeConfig
	cfg.SrvLen = 0
	cfg.CntLen = 22

	gen, _ := NewUidGenerator(cfg, 0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		id := gen.NextID()
		u := gen.Unix(id)
		id = gen.FromUnix(u)
	}
}

func BenchmarkUidGenerator(b *testing.B) {
	cfg := SnowflakeConfig
	cfg.SrvLen = 0
	cfg.CntLen = 22

	gen, _ := NewUidGenerator(cfg, 0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gen.NextID()
	}
}
