package block

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"sync"

	"github.com/iost-official/Go-IOS-Protocol/core/state"
	"github.com/iost-official/Go-IOS-Protocol/core/tx"
	"github.com/iost-official/Go-IOS-Protocol/db"
	"github.com/iost-official/Go-IOS-Protocol/log"
)

var (
	blockLength = []byte("BlockLength") //blockLength -> length of ChainImpl

	blockNumberPrefix = []byte("n") //blockNumberPrefix + block number -> block hash
	blockPrefix       = []byte("H") //blockHashPrefix + block hash -> block data
)

type ChainImpl struct {
	db     db.Database
	length uint64
	tx     tx.TxPool
}

var BChain Chain
var once sync.Once

var LdbPath string

func Instance() (Chain, error) {
	var err error

	once.Do(func() {

		ldb, er := db.NewLDBDatabase(LdbPath+"blockDB", 0, 0)
		if er != nil {
			err = fmt.Errorf("failed to init db %v", err)
			return
		}
		//defer ldb.Close()

		var length uint64
		var lenByte = make([]byte, 128)

		if ok, _ := ldb.Has(blockLength); ok {
			lenByte, er := ldb.Get(blockLength)
			if er != nil {
				err = fmt.Errorf("failed to Get blockLength")
				return
			}

			length = binary.BigEndian.Uint64(lenByte)

		} else {
			fmt.Printf("blockLength not exist")
			length = 0
			binary.BigEndian.PutUint64(lenByte, length)

			er := ldb.Put(blockLength, lenByte)
			if er != nil {
				err = fmt.Errorf("failed to Put blockLength")
				return
			}
		}

		txDb := tx.TxDb
		if txDb == nil {
			panic(fmt.Errorf("TxDb shouldn't be nil"))
		}
		if er != nil {
			err = fmt.Errorf("failed to NewTxPoolDb: [%v]", err)
			return
		}

		BChain = &ChainImpl{db: ldb, length: length, tx: txDb}

		BChain.CheckLength()
	})

	return BChain, err
}

func (b *ChainImpl) Push(block *Block) error {

	hash := block.HeadHash()
	number := uint64(block.Head.Number)

	err := b.db.Put(append(blockNumberPrefix, strconv.FormatUint(number, 10)...), hash)
	if err != nil {
		return fmt.Errorf("failed to Put block hash err[%v]", err)
	}

	err = b.db.Put(append(blockPrefix, hash...), block.Encode())
	if err != nil {
		return fmt.Errorf("failed to Put block data")
	}

	//put all the tx of this block to txdb
	for _, ctx := range block.Content {
		if err := b.tx.Add(&ctx); err != nil {
			return fmt.Errorf("failed to add tx %v", err)
		}

	}

	err = b.lengthAdd(number)
	if err != nil {
		return fmt.Errorf("failed to lengthAdd %v", err)
	}

	state.StdPool.Put(state.Key("BlockNum"), state.MakeVInt(int(block.Head.Number)))
	state.StdPool.Put(state.Key("BlockHash"), state.MakeVByte(block.HeadHash()))
	state.StdPool.Flush()

	// add servi
	go tx.Data.AddServi(block.Content)

	return nil
}

func (b *ChainImpl) Length() uint64 {
	return b.length
}

func (b *ChainImpl) CheckLength() error {

	dbLen := b.Length()

	var i uint64
	for i = dbLen; i > 0; i-- {
		bb := b.GetBlockByNumber(i - 1)
		if bb != nil {
			log.Log.I("[block] set block length %v", i)
			b.setLength(i)
			break
		} else {
			log.Log.E("[block] Length error %v", i)
		}
	}

	return nil
}

func (b *ChainImpl) setLength(l uint64) error {

	var lenB = make([]byte, 128)
	binary.BigEndian.PutUint64(lenB, l)

	er := b.db.Put(blockLength, lenB)
	if er != nil {
		return fmt.Errorf("failed to Put blockLength err:%v", er)
	}

	b.length = l

	return nil
}

func (b *ChainImpl) HasTx(tx *tx.Tx) (bool, error) {
	return b.tx.Has(tx)
}

func (b *ChainImpl) GetTx(hash []byte) (*tx.Tx, error) {
	return b.tx.Get(hash)
}

func (b *ChainImpl) lengthAdd(blockNum uint64) error {

	log.Log.E("[block] lengthAdd length:%v block num:%v ", b.length, blockNum)

	b.length = blockNum + 1

	var tmpByte = make([]byte, 128)
	binary.BigEndian.PutUint64(tmpByte, b.length)

	err := b.db.Put(blockLength, tmpByte)
	if err != nil {
		b.length--
	}

	return nil
}

func (b *ChainImpl) getLengthBytes(length uint64) []byte {

	return []byte(strconv.FormatUint(length, 10))
}

func (b *ChainImpl) Top() *Block {

	var blk *Block
	if b.length == 0 {
		return b.GetBlockByNumber(b.length)
	} else {
		for i := b.length; i > 0; i-- {
			blk = b.GetBlockByNumber(i - 1)
			if blk != nil {
				break
			}
		}

		return blk
	}
}

func (b *ChainImpl) GetHashByNumber(number uint64) []byte {
	hash, err := b.db.Get(append(blockNumberPrefix, b.getLengthBytes(number)...))
	if err != nil {
		log.Log.E("Get block hash error: %v number: %v", err, number)
		return nil
	}
	return hash
}

func (b *ChainImpl) GetBlockByNumber(number uint64) *Block {

	hash, err := b.db.Get(append(blockNumberPrefix, b.getLengthBytes(number)...))
	if err != nil {
		log.Log.E("Get block hash error: %v number: %v", err, number)
		return nil
	}

	block, err := b.db.Get(append(blockPrefix, hash...))
	if err != nil {
		log.Log.E("Get block error: %v number: %v", err, number)
		return nil
	}
	if len(block) == 0 {
		log.Log.E("GetBlockByNumber Block empty! number: %v", number)
		return nil
	}
	rBlock := new(Block)
	if err := rBlock.Decode(block); err != nil {
		log.Log.E("Failed to GetBlockByNumber Decode err: %v", err)
		return nil
	}
	return rBlock
}

func (b *ChainImpl) GetBlockByHash(blockHash []byte) *Block {

	block, err := b.db.Get(append(blockPrefix, blockHash...))
	if err != nil {
		return nil
	}
	if len(block) == 0 {
		return nil
	}

	rBlock := new(Block)
	if err := rBlock.Decode(block); err != nil {
		return nil
	}
	return rBlock
}

func (b *ChainImpl) GetBlockByteByHash(blockHash []byte) ([]byte, error) {

	block, err := b.db.Get(append(blockPrefix, blockHash...))
	if err != nil {
		log.Log.E("Get block error: %v hash: %v", err, string(blockHash))
		return nil, err
	}
	if len(block) == 0 {
		log.Log.E("GetBlockByteByHash Block empty! : %v", string(blockHash))
		return nil, fmt.Errorf("block empty")
	}

	return block, nil
}

func (b *ChainImpl) Iterator() ChainIterator {
	return nil
}
