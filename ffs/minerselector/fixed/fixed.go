package fixed

import (
	"fmt"

	"github.com/textileio/powergate/v2/ffs"
)

// MinerSelector is a MinerSelector implementation which
// always return a single miner address with an fixed epochPrice.
type MinerSelector struct {
	miners []Miner
}

// Miner contains miner information.
type Miner struct {
	Addr       string
	Country    string
	EpochPrice uint64
}

var _ ffs.MinerSelector = (*MinerSelector)(nil)

// New returns a new FixedMinerSelector that always return addr as the miner address
// and epochPrice.
func New(miners []Miner) *MinerSelector {
	fixedMiners := make([]Miner, len(miners))
	copy(fixedMiners, miners)
	return &MinerSelector{
		miners: fixedMiners,
	}
}

// GetMiners returns the single allowed miner in the selector.
func (fms *MinerSelector) GetMiners(n int, f ffs.MinerSelectorFilter) ([]ffs.MinerProposal, error) {
	res := make([]ffs.MinerProposal, 0, n)
	mres := make(map[string]struct{})
	for _, pm := range f.TrustedMiners {
		for _, m := range fms.miners {
			if m.Addr == pm {
				if f.MaxPrice > 0 && m.EpochPrice > f.MaxPrice {
					continue
				}
				mres[m.Addr] = struct{}{}
				res = append(res, ffs.MinerProposal{
					Addr:       m.Addr,
					EpochPrice: m.EpochPrice,
				})
				break
			}
		}
		if len(res) == n {
			return res, nil
		}
	}

	for _, m := range fms.miners {
		if _, ok := mres[m.Addr]; ok {
			continue
		}
		if f.MaxPrice > 0 && m.EpochPrice > f.MaxPrice {
			continue
		}
		skip := false
		for _, bAddr := range f.ExcludedMiners {
			if bAddr == m.Addr {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if len(f.CountryCodes) != 0 {
			skip := true
			for _, c := range f.CountryCodes {
				if c == m.Country {
					skip = false
					break
				}
			}
			if skip {
				continue
			}
		}
		res = append(res, ffs.MinerProposal{
			Addr:       m.Addr,
			EpochPrice: m.EpochPrice,
		})
		if len(res) == n {
			break
		}
	}
	if len(res) != n {
		return nil, fmt.Errorf("not enough fixed miners to provide, want %d, got %d", n, len(res))
	}
	return res, nil
}
