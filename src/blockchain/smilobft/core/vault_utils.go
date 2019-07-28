package core

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"

	"go-smilo/src/blockchain/smilobft/core/types"
	"go-smilo/src/blockchain/smilobft/ethdb"
)

var (
	vaultRootPrefix = []byte("P")
	//vaultblockReceiptsPrefix = []byte("Pr") // blockReceiptsPrefix + num (uint64 big endian) + hash -> block receipts
	//vaultReceiptPrefix       = []byte("Prs")
	vaultBloomPrefix = []byte("Pb")
)

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

func GetVaultStateRoot(db ethdb.Database, blockRoot common.Hash) common.Hash {
	root, _ := db.Get(append(vaultRootPrefix, blockRoot[:]...))
	return common.BytesToHash(root)
}

func WriteVaultStateRoot(db ethdb.Database, blockRoot, root common.Hash) error {
	return db.Put(append(vaultRootPrefix, blockRoot[:]...), root[:])
}

// WriteVaultBlockBloom creates a bloom filter for the given receipts and saves it to the database
// with the number given as identifier (i.e. block number).
func WriteVaultBlockBloom(db ethdb.Database, number uint64, receipts types.Receipts) error {
	rbloom := types.CreateBloom(receipts)
	return db.Put(append(vaultBloomPrefix, encodeBlockNumber(number)...), rbloom[:])
}

// GetVaultBlockBloom retrieves the vault bloom associated with the given number.
func GetVaultBlockBloom(db ethdb.Database, number uint64) (bloom types.Bloom) {
	data, _ := db.Get(append(vaultBloomPrefix, encodeBlockNumber(number)...))
	if len(data) > 0 {
		bloom = types.BytesToBloom(data)
	}
	return bloom
}
