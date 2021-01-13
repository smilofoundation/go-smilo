package crypto

import (
	"crypto/ecdsa"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"go-smilo/src/blockchain/smilobft/core/types"
)

var ErrUnauthorizedAddress = errors.New("unauthorized address")

// SignHeader signs the given header with the given private key.
func SignHeader(h *types.Header, priv *ecdsa.PrivateKey) error {
	hashData := crypto.Keccak256(types.SigHash(h).Bytes())
	signature, err := crypto.Sign(hashData, priv)
	if err != nil {
		return err
	}
	err = types.WriteSeal(h, signature)
	if err != nil {
		return err
	}
	return nil
}

func CheckValidatorSignature(previousHeader *types.Header, data []byte, sig []byte) (common.Address, error) {
	// 1. Get signature address
	signer, err := types.GetSignatureAddress(data, sig)
	if err != nil {
		log.Error("Failed to get signer address", "err", err)
		return common.Address{}, err
	}

	// 2. Check validator
	val := previousHeader.CommitteeMember(signer)
	if val == nil {
		return common.Address{}, ErrUnauthorizedAddress
	}

	return val.Address, nil
}
