package uidgen

import (
	"math"
	"strconv"
	"sync"
	"time"

	"errors"
)

// UidGeneratorConfig is the configuration for UidGenerator.
type UidGeneratorConfig struct {
	EpochLen    uint8 // EpochLen is the bit length of the epoch field.
	SrvLen      uint8 // SrvLen is the bit length of the server ID field.
	CntLen      uint8 // CntLen is the bit length of the sequence counter field.
	IntervalLen uint8 // IntervalLen is the bit length of the time interval. Defaults to 61 - EpochLen.
	TruncStr    bool  // TruncStr allows you to truncate the last zeroes when converting to string.
	EpochStart  int64 // EpochStart is the point in time since which the UniqueID time is defined as elapsed.
	SrvID       int64 // SrvID is the current server ID. Must be less than 2^SrvLen.

	strLen     int8
	epochShift uint8
	epochMask  UniqueID
	epochIota  UniqueID
	maxSrv     UniqueID
	srvMask    UniqueID
	maxCnt     UniqueID
	cntMask    UniqueID
	interval   int64
	timeMask   int64
}

func (cfg *UidGeneratorConfig) update() error {
	idLen := cfg.EpochLen + cfg.SrvLen + cfg.CntLen
	if idLen > 64 {
		return ErrTooLongID
	}

	cfg.strLen = int8(math.Ceil(float64(idLen) / letterLen))
	cfg.epochShift = cfg.SrvLen + cfg.CntLen
	cfg.epochMask = (1<<cfg.EpochLen - 1) << cfg.epochShift
	cfg.epochIota = 1 << cfg.epochShift
	cfg.maxSrv = 1<<cfg.SrvLen - 1
	cfg.srvMask = cfg.maxSrv << cfg.CntLen
	cfg.maxCnt = 1<<cfg.CntLen - 1
	cfg.cntMask = cfg.maxCnt

	if cfg.IntervalLen == 0 {
		cfg.IntervalLen = 61 - cfg.EpochLen
	}

	cfg.interval = 1 << cfg.IntervalLen
	cfg.timeMask = cfg.interval - 1

	return nil
}

// SnowflakeConfig is a sample configuration that is most closely related to the original Twitter Snowflake.
// The field lengths and EpochStart are identical to those in Snowflake.
// The time interval is 2^20 ns or approximately 1 ms.
var SnowflakeConfig = UidGeneratorConfig{
	EpochLen:    41,
	SrvLen:      10,
	CntLen:      12,
	EpochStart:  1288834974, // 2010-11-04T01:42:54
	IntervalLen: 20,
}

const (
	letters    = "abcdefghijklmnopqrstuvwxyzABCDEF"
	letterLen  = 5
	letterMask = 1<<letterLen - 1
)

var decodeLetters [256]byte

// ErrInvalidStringUID is returned by UidFromString when given an invalid string
var ErrInvalidStringUID = errors.New("invalid encoded UniqueID")

// ErrTooBigServerID is returned by NewUidGenerator when given a server ID bigger than 2^srvLen-1
var ErrTooBigServerID = errors.New("server ID is too big")

// ErrTooLongID is returned by NewUidGenerator if length of IDs would exceed 64 bits
var ErrTooLongID = errors.New("configured ID length is too big")

func init() {
	for i := 0; i < len(letters); i++ {
		decodeLetters[i] = 0xFF
	}

	for i := 0; i < len(letters); i++ {
		decodeLetters[letters[i]] = byte(i)
	}
}

type UniqueID uint64

// UidGenerator is a distributed unique ID generator.
type UidGenerator struct {
	mu sync.Mutex

	cfg UidGeneratorConfig

	start time.Time
	epoch UniqueID
	srvID UniqueID
	cnt   UniqueID
}

// NewUidGenerator returns a new UidGenerator configured with the specified UidGeneratorConfig.
// In the event of a restart, you can specify the previous generated UniqueID,
// and the sequence will continue from there.
func NewUidGenerator(cfg UidGeneratorConfig, prevID UniqueID) (*UidGenerator, error) {
	err := cfg.update()
	if err != nil {
		return nil, err
	}

	if cfg.SrvID > int64(cfg.maxSrv) {
		return nil, ErrTooBigServerID
	}

	gen := &UidGenerator{
		cfg:   cfg,
		epoch: prevID & cfg.epochMask,
		srvID: UniqueID(cfg.SrvID << cfg.CntLen),
		cnt:   prevID & cfg.cntMask,
	}

	now := time.Now()
	gen.start = now.Add(time.Unix(cfg.EpochStart, 0).Sub(now))

	return gen, nil
}

// NextID generates a next UniqueID.
func (gen *UidGenerator) NextID() UniqueID {
	since := time.Since(gen.start).Nanoseconds() >> gen.cfg.IntervalLen
	epoch := UniqueID(since) << gen.cfg.epochShift & gen.cfg.epochMask

	gen.mu.Lock()

	if epoch <= gen.epoch {
		gen.cnt++
		if gen.cnt > gen.cfg.maxCnt {
			nsec := gen.cfg.interval - int64(time.Now().Nanosecond())&gen.cfg.timeMask
			time.Sleep(time.Duration(nsec) * time.Nanosecond)

			gen.epoch = gen.epoch + gen.cfg.epochIota
			gen.cnt = 0
		}
	} else {
		gen.epoch = epoch
		gen.cnt = 0
	}

	id := gen.epoch + gen.srvID + gen.cnt

	// We don't use "defer" here to improve performance.
	gen.mu.Unlock()

	return id
}

// FromBase32 returns a UniqueID parsed from the string.
func (gen *UidGenerator) FromBase32(str string) (UniqueID, error) {
	var id UniqueID

	var i int8

	for ; i < int8(len(str)); i++ {
		ch := decodeLetters[str[i]]
		if ch == 0xFF {
			return 0, ErrInvalidStringUID
		}

		id <<= letterLen
		id = id + UniqueID(ch)
	}

	for ; i < gen.cfg.strLen; i++ {
		id <<= letterLen
	}

	return id, nil
}

// ToBase32 stringifies the UniqueID.
func (gen *UidGenerator) ToBase32(id UniqueID) string {
	i := gen.cfg.strLen - 1

	if gen.cfg.TruncStr {
		for id&letterMask == 0 {
			id >>= letterLen
			i--
		}
	}

	b := make([]byte, i+1)

	for ; i >= 0; i-- {
		idx := id & letterMask
		b[i] = letters[idx]
		id >>= letterLen
	}

	return string(b)
}

// FromUnix returns a new UniqueID with the specified time in seconds since Unix epoch.
func (gen *UidGenerator) FromUnix(epoch int64) UniqueID {
	return UniqueID(epoch-gen.cfg.EpochStart)*1e9>>gen.cfg.IntervalLen<<gen.cfg.epochShift + gen.srvID
}

// Unix returns the time in seconds since Unix epoch.
func (gen *UidGenerator) Unix(id UniqueID) int64 {
	nsec := int64(id >> gen.cfg.epochShift << gen.cfg.IntervalLen)
	unix := nsec / 1e9

	if nsec%1e9 >= 5e8 {
		unix++
	}

	return unix + gen.cfg.EpochStart
}

// FromUnixNano returns a new UniqueID with the specified time in nanoseconds since Unix epoch.
func (gen *UidGenerator) FromUnixNano(epoch int64) UniqueID {
	return UniqueID(epoch-gen.cfg.EpochStart*1e9)>>gen.cfg.IntervalLen<<gen.cfg.epochShift + gen.srvID
}

// UnixNano returns the time in nanoseconds since Unix epoch.
func (gen *UidGenerator) UnixNano(id UniqueID) int64 {
	return int64(id>>gen.cfg.epochShift<<gen.cfg.IntervalLen) + gen.cfg.EpochStart*1e9
}

// ServerID returns the server ID.
func (gen *UidGenerator) ServerID(id UniqueID) int64 {
	return int64(id&gen.cfg.srvMask) >> gen.cfg.CntLen
}

// Count returns the sequence counter.
func (gen *UidGenerator) Count(id UniqueID) int64 {
	return int64(id & gen.cfg.cntMask)
}

func (id UniqueID) Int64() int64 {
	return int64(id)
}

func (id UniqueID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

func FromString(idStr string) (UniqueID, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	return UniqueID(id), err
}
