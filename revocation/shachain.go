package revocation

import "github.com/btcsuite/btcd/wire"

// chainFragment...
type chainFragment struct {
	index uint64
	hash  wire.ShaHash
}

// HyperShaChain...
type HyperShaChain struct {
	lastChainIndex uint64

	chainFragments []chainFragment
}

// NewHyperShaChain...
func NewHyperShaChain(seed wire.ShaHash) *HyperShaChain {
	// TODO(roasbeef): from/to or static size?
	return nil
}

// NextHash...
func (s *HyperShaChain) NextHash() {
}

// GetHash...
func (s *HyperShaChain) GetHash(index uint64) {
}
