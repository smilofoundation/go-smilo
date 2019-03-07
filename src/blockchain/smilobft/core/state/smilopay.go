// Copyright 2019 The go-smilo Authors
// This file is part of the go-smilo library.
//
// The go-smilo library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-smilo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-smilo library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"math/big"
)

func CalculateSmiloPay(prevBlock, newBlock, prevSmiloPay, balance *big.Int) *big.Int {
	//if balance.Cmp(big.NewInt(1e+16)) < 0 {
	//	log.Debug("CalculateSmiloPay, Balance:", balance.Int64(), ", Min required: ", big.NewInt(1e+16).Int64(), ", returned: ",common.Big0)
	//	return common.Big0
	//}

	//if block did not change, return prevSmiloPay
	if prevBlock.Cmp(newBlock) >= 0 {
		return prevSmiloPay
	}

	maxSmiloPay, balanceSmilo := MaxSmiloPay(balance)

	blockGap := new(big.Int).Sub(newBlock, prevBlock)

	// (0,000001 + (√balance / 750000)) * Correction factor
	// smiloSpeed := (0.000001 + (sqrt / 750000)) * 8 * 1000000000000000000 (To avoid overflow, its coded with big.Float)
	sqrt := new(big.Float).Sqrt(balanceSmilo)
	sqrtDiv := new(big.Float).Quo(sqrt, big.NewFloat(750000))
	sqrtAdd := new(big.Float).Add(sqrtDiv, big.NewFloat(0.000001))
	smiloSpeedMul := new(big.Float).Mul(sqrtAdd, big.NewFloat(1))
	smiloSpeedBig := new(big.Float).Mul(smiloSpeedMul, big.NewFloat(1000000000000000000))

	blockGapFloat := new(big.Float).SetInt(blockGap)
	smiloPayResult := new(big.Float).Mul(blockGapFloat, smiloSpeedBig)

	prevSmiloPayFloat := new(big.Float).SetInt(prevSmiloPay)
	smiloPayFloat := new(big.Float).Add(prevSmiloPayFloat, smiloPayResult)

	if smiloPayFloat.Cmp(new(big.Float).SetInt(maxSmiloPay)) > 0 || prevSmiloPayFloat.Cmp(smiloPayFloat) > 0 {
		return maxSmiloPay
	}

	return floatToBigInt(smiloPayFloat, big.NewInt(1))
}

func MaxSmiloPay(balance *big.Int) (maxSmiloPayReturn *big.Int, balanceSmilo *big.Float) {
	balanceDecimals := new(big.Int).Div(balance, big.NewInt(1e18))
	balanceSmilo = new(big.Float).SetInt(balanceDecimals)

	//(0,001 + (√balance / 50000)) * Correction factor
	// maxSmiloPay := (0.001 + (f / 50000)) * 5000 * 1000000000000000 (To avoid overflow, its coded with big.Float)
	f := new(big.Float).Sqrt(balanceSmilo)
	fDiv := new(big.Float).Quo(f, big.NewFloat(50000))
	fAdd := new(big.Float).Add(fDiv, big.NewFloat(0.001))
	maxSmiloPayMul := new(big.Float).Mul(fAdd, big.NewFloat(1))

	maxSmiloPayInt := floatToBigInt(maxSmiloPayMul, big.NewInt(1000000000000000000))

	return maxSmiloPayInt, balanceSmilo
}

func floatToBigInt(bigval *big.Float, precision *big.Int) *big.Int {
	coin := new(big.Float)
	coin.SetInt(precision)

	bigval.SetPrec(64)
	bigval.Mul(bigval, coin)

	result := new(big.Int)
	bigval.Int(result) // store converted number in result

	return result
}
