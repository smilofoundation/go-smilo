package log

import "github.com/ethereum/go-ethereum/log"

const (
	TxCreated          = "TX-CREATED"
	TxAccepted         = "TX-ACCEPTED"
	BecameMinter       = "BECAME-MINTER"
	BecameVerifier     = "BECAME-VERIFIER"
	BlockCreated       = "BLOCK-CREATED"
	BlockVotingStarted = "BLOCK-VOTING-STARTED"
)

var DoEmitCheckpoints = false

func EmitCheckpoint(checkpointName string, logValues ...interface{}) {
	args := []interface{}{"name", checkpointName}
	args = append(args, logValues...)
	if DoEmitCheckpoints {
		log.Info("SMILO-CHECKPOINT", args...)
	}
}
