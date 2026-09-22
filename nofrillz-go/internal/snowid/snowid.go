package snowid

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"nofrillz/internal/config"
)

const (
	DefaultEpochMs int64 = 1770741350000

	DefaultTimeBits   = 41
	DefaultRegionBits = 3  // 8 regions
	DefaultNodeBits   = 10 // 1,024 nodes
	DefaultSeqBits    = 10

	BusyWaitSleep = time.Microsecond
)

type Layout struct {
	EpochMs    int64
	TimeBits   uint8
	RegionBits uint8
	NodeBits   uint8
	SeqBits    uint8
}

func DefaultLayout() Layout {
	return Layout{
		EpochMs:    DefaultEpochMs,
		TimeBits:   DefaultTimeBits,
		RegionBits: DefaultRegionBits,
		NodeBits:   DefaultNodeBits,
		SeqBits:    DefaultSeqBits,
	}
}

func (l Layout) Validate() error {
	sum := int(l.TimeBits) + int(l.RegionBits) + int(l.NodeBits) + int(l.SeqBits)
	if sum != 64 {
		return fmt.Errorf("invalid layout: bits sum to %d (expected 64)", sum)
	}
	if l.TimeBits < 30 || l.TimeBits > 50 {
		return fmt.Errorf("invalid TimeBits: %d", l.TimeBits)
	}
	if l.RegionBits == 0 || l.RegionBits > 10 {
		return fmt.Errorf("invalid RegionBits: %d", l.RegionBits)
	}
	if l.NodeBits == 0 || l.NodeBits > 20 {
		return fmt.Errorf("invalid NodeBits: %d", l.NodeBits)
	}
	if l.SeqBits == 0 || l.SeqBits > 20 {
		return fmt.Errorf("invalid SeqBits: %d", l.SeqBits)
	}
	return nil
}

type Generator struct {
	mu sync.Mutex

	layout Layout

	region uint64
	node   uint64

	lastMs uint64
	seq    uint64
}

// New creates a new Generator using cfg.Region/cfg.Node and the provided layout.
func New(config *config.IDGeneratorConfig, layout Layout) (*Generator, error) {
	if err := layout.Validate(); err != nil {
		return nil, err
	}

	maxRegion := uint64((uint64(1) << layout.RegionBits) - 1)
	maxNode := uint64((uint64(1) << layout.NodeBits) - 1)

	if config.Region > maxRegion {
		return nil, fmt.Errorf("%w: %d (max %d)", errors.New("region out of range"), config.Region, maxRegion)
	}
	if config.Node > maxNode {
		return nil, fmt.Errorf("%w: %d (max %d)", errors.New("node out of range"), config.Node, maxNode)
	}

	return &Generator{
		layout: layout,
		region: config.Region,
		node:   config.Node,
	}, nil
}

// NewDefault creates a Generator using DefaultLayout().
func NewDefault(config *config.IDGeneratorConfig) (*Generator, error) {
	return New(config, DefaultLayout())
}

// Next returns the next ID.
func (g *Generator) Next() (uint64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	nowMs := uint64(time.Now().UTC().UnixMilli())

	if int64(nowMs) < g.layout.EpochMs {
		return 0, errors.New("epoch is in the future relative to current clock")
	}

	curr := uint64(int64(nowMs) - g.layout.EpochMs)

	if curr < g.lastMs {
		for curr < g.lastMs {
			time.Sleep(BusyWaitSleep)
			nowMs = uint64(time.Now().UTC().UnixMilli())
			if int64(nowMs) < g.layout.EpochMs {
				return 0, errors.New("epoch is in the future relative to current clock")
			}
			curr = uint64(int64(nowMs) - g.layout.EpochMs)
		}
	}

	maxSeq := uint64((uint64(1) << g.layout.SeqBits) - 1)

	if curr == g.lastMs {
		g.seq = (g.seq + 1) & maxSeq
		if g.seq == 0 {
			for curr == g.lastMs {
				time.Sleep(BusyWaitSleep)
				nowMs = uint64(time.Now().UTC().UnixMilli())
				if int64(nowMs) < g.layout.EpochMs {
					return 0, errors.New("epoch is in the future relative to current clock")
				}
				curr = uint64(int64(nowMs) - g.layout.EpochMs)
			}
			g.lastMs = curr
			g.seq = 0
		}
	} else {
		g.lastMs = curr
		g.seq = 0
	}

	return pack(curr, g.region, g.node, g.seq, g.layout), nil
}

func (g *Generator) MustNext() uint64 {
	id, err := g.Next()
	if err != nil {
		panic(err)
	}
	return id
}

type Decoded struct {
	UnixMs  int64
	SinceEp uint64
	Region  uint64
	Node    uint64
	Seq     uint64
}

func Decode(id uint64, layout Layout) (Decoded, error) {
	if err := layout.Validate(); err != nil {
		return Decoded{}, err
	}

	seqMask := uint64((uint64(1) << layout.SeqBits) - 1)
	nodeMask := uint64((uint64(1) << layout.NodeBits) - 1)
	regionMask := uint64((uint64(1) << layout.RegionBits) - 1)

	seq := id & seqMask
	node := (id >> layout.SeqBits) & nodeMask
	region := (id >> (layout.SeqBits + layout.NodeBits)) & regionMask
	since := id >> (layout.SeqBits + layout.NodeBits + layout.RegionBits)

	unixMs := int64(since) + layout.EpochMs

	return Decoded{
		UnixMs:  unixMs,
		SinceEp: since,
		Region:  region,
		Node:    node,
		Seq:     seq,
	}, nil
}

func pack(msSinceEpoch, region, node, seq uint64, layout Layout) uint64 {
	timeShift := layout.RegionBits + layout.NodeBits + layout.SeqBits
	regionShift := layout.NodeBits + layout.SeqBits
	nodeShift := layout.SeqBits

	return (msSinceEpoch << timeShift) |
		(region << regionShift) |
		(node << nodeShift) |
		seq
}
