package core

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/ethdb"
)

var (
	privateRootPrefix = []byte("P")
	//vaultblockReceiptsPrefix = []byte("Pr") // blockReceiptsPrefix + num (uint64 big endian) + hash -> block receipts
	//vaultReceiptPrefix       = []byte("Prs")
	privateBloomPrefix = []byte("Pb")
)

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

func GetPrivateStateRoot(db ethdb.Database, blockRoot common.Hash) common.Hash {
	root, _ := db.Get(append(privateRootPrefix, blockRoot[:]...))
	return common.BytesToHash(root)
}

func WritePrivateStateRoot(db ethdb.Database, blockRoot, root common.Hash) error {
	return db.Put(append(privateRootPrefix, blockRoot[:]...), root[:])
}


// GetPrivateBlockBloom retrieves the vault bloom associated with the given number.
func GetPrivateBlockBloom(db ethdb.Database, number uint64) (bloom types.Bloom) {
	data, _ := db.Get(append(privateBloomPrefix, encodeBlockNumber(number)...))
	if len(data) > 0 {
		bloom = types.BytesToBloom(data)
	}
	return bloom
}
