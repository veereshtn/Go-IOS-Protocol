package pow

import (
	"github.com/ethereum/go-ethereum/core"
	"fmt"
)

type CachedBlockChain struct {
	core.BlockChain
	cachedBlock []*core.Block
}

func NewCBC(chain core.BlockChain) CachedBlockChain {
	return CachedBlockChain{
		BlockChain:  chain,
		cachedBlock: make([]*core.Block, 0),
	}
}

func (c *CachedBlockChain) Get(layer int) (*core.Block, error) {
	if layer < 0 || layer >= c.BlockChain.Length()+len(c.cachedBlock) {
		return nil, fmt.Errorf("overflow")
	}
	if layer < c.BlockChain.Length() {
		return c.BlockChain.Get(layer)
	}
	return c.cachedBlock[layer-c.BlockChain.Length()], nil
}
func (c *CachedBlockChain) Push(block *core.Block) error {
	c.cachedBlock = append(c.cachedBlock, block)
	return nil
}
func (c *CachedBlockChain) Length() int {
	return c.BlockChain.Length() + len(c.cachedBlock)
}
