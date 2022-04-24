// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"io"
	"math/big"
	"math/bits"
	"sync/atomic"
	"time"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon/common"
)

type DepositTx struct {
	time time.Time // Time first seen locally (spam avoidance)
	// caches
	hash atomic.Value //nolint:structcheck
	size atomic.Value //nolint:structcheck
	// SourceHash uniquely identifies the source of the deposit
	SourceHash common.Hash
	// From is exposed through the types.Signer, not through TxData
	From common.Address
	// nil means contract creation
	To *common.Address `rlp:"nil"`
	// Mint is minted on L2, locked on L1, nil if no minting.
	Mint *uint256.Int
	// Value is transferred from L2 balance, executed after Mint (if any)
	Value *uint256.Int
	// gas limit
	Gas uint64
	// wei per gas
	GasPrice *uint256.Int
	Data     []byte
}

var _ Transaction = (*DepositTx)(nil)

func (tx DepositTx) GetChainID() *uint256.Int {
	panic("deposits are not signed and do not have a chain-ID")
}

// DepositsNonce identifies a deposit, since go-ethereum abstracts all transaction types to a core.Message.
// Deposits do not set a nonce, deposits are included by the system and cannot be repeated or included elsewhere.
const DepositsNonce uint64 = 0xffff_ffff_ffff_fffd

func (tx DepositTx) GetNonce() uint64 {
	return DepositsNonce
}

func (tx DepositTx) GetTo() *common.Address {
	return tx.To
}

func (tx DepositTx) GetGas() uint64 {
	return tx.Gas
}

func (tx DepositTx) GetValue() *uint256.Int {
	return tx.Value
}

func (tx DepositTx) GetData() []byte {
	return tx.Data
}

func (tx DepositTx) GetSender() (common.Address, bool) {
	return tx.From, false
}

func (tx DepositTx) SetSender(addr common.Address) {
	tx.From = addr
}

func (tx DepositTx) RawSignatureValues() (*uint256.Int, *uint256.Int, *uint256.Int) {
	panic("deposit tx does not have a signature")
}

func (tx DepositTx) SigningHash(chainID *big.Int) common.Hash {
	return rlpHash([]interface{}{
		DepositsNonce,
		tx.GasPrice,
		tx.Gas,
		tx.To,
		tx.Value,
		tx.Data,
	})
}

// NOTE: Need to check this
func (tx *DepositTx) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := tx.EncodingSize()
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// NOTE: Need to check this
func (tx DepositTx) EncodingSize() int {
	payloadSize, _, _ := tx.payloadSize()
	return payloadSize
}

// NOTE: honestly, tired of saying it, but need to check this lol
func (tx DepositTx) payloadSize() (payloadSize int, nonceLen, gasLen int) {
	payloadSize++
	if tx.Nonce >= 128 {
		nonceLen = (bits.Len64(tx.Nonce) + 7) / 8
	}
	payloadSize += nonceLen
	payloadSize++
	var gasPriceLen int
	if tx.GasPrice.BitLen() >= 8 {
		gasPriceLen = (tx.GasPrice.BitLen() + 7) / 8
	}
	payloadSize += gasPriceLen
	payloadSize++
	if tx.Gas >= 128 {
		gasLen = (bits.Len64(tx.Gas) + 7) / 8
	}
	payloadSize += gasLen
	payloadSize++
	if tx.To != nil {
		payloadSize += 20
	}
	payloadSize++
	var valueLen int
	if tx.Value.BitLen() >= 8 {
		valueLen = (tx.Value.BitLen() + 7) / 8
	}
	payloadSize += valueLen
	// size of Data
	payloadSize++
	switch len(tx.Data) {
	case 0:
	case 1:
		if tx.Data[0] >= 128 {
			payloadSize++
		}
	default:
		if len(tx.Data) >= 56 {
			payloadSize += (bits.Len(uint(len(tx.Data))) + 7) / 8
		}
		payloadSize += len(tx.Data)
	}
	// size of V
	payloadSize++
	var vLen int
	if tx.V.BitLen() >= 8 {
		vLen = (tx.V.BitLen() + 7) / 8
	}
	payloadSize += vLen
	payloadSize++
	var rLen int
	if tx.R.BitLen() >= 8 {
		rLen = (tx.R.BitLen() + 7) / 8
	}
	payloadSize += rLen
	payloadSize++
	var sLen int
	if tx.S.BitLen() >= 8 {
		sLen = (tx.S.BitLen() + 7) / 8
	}
	payloadSize += sLen
	return payloadSize, nonceLen, gasLen
}

// EncodeString

// EncodeStringSizePrefix

// MarshalBinary

// encodePayload

// EncodeRLP

// DeocdeRLP

func (tx *DepositTx) FakeSign(address common.Address) (Transaction, error) {
	cpy := tx.copy()
	cpy.SetSender(address)
	return cpy, nil
}

func (tx *DepositTx) WithSignature(signer Signer, sig []byte) (Transaction, error) {
	return tx.copy(), nil
}

func (tx DepositTx) Time() time.Time {
	return tx.time
}

func (tx DepositTx) Type() byte { return DepositTxType }

func (tx *DepositTx) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return *hash.(*common.Hash)
	}
	hash := rlpHash([]interface{}{
		tx.SourceHash,
		tx.From,
		tx.To,
		tx.Mint,
		tx.Value,
		tx.Gas,
		tx.GasPrice,
		tx.Data,
	})
	tx.hash.Store(&hash)
	return hash
}

// TODO: marshalling stuff
func (tx DepositTx) MarshalBinary(w io.Writer) error {
	return nil
}

// not sure ab this one lol
func (tx DepositTx) Protected() bool {
	return true
}

func (tx DepositTx) IsContractDeploy() bool {
	return false
}

func (tx DepositTx) IsStarkNet() bool {
	return false
}

func (tx DepositTx) GetPrice() *uint256.Int  { return tx.GasPrice }
func (tx DepositTx) GetTip() *uint256.Int    { return tx.GasPrice }
func (tx DepositTx) GetFeeCap() *uint256.Int { return tx.GasPrice }

// Is this needed at all?
func (tx DepositTx) GetEffectiveGasTip(baseFee *uint256.Int) *uint256.Int {
	if baseFee == nil {
		return tx.GetTip()
	}
	gasFeeCap := tx.GetFeeCap()
	// return 0 because effectiveFee cant be < 0
	if gasFeeCap.Lt(baseFee) {
		return uint256.NewInt(0)
	}
	effectiveFee := new(uint256.Int).Sub(gasFeeCap, baseFee)
	if tx.GetTip().Lt(effectiveFee) {
		return tx.GetTip()
	} else {
		return effectiveFee
	}
}

func (tx DepositTx) Cost() *uint256.Int {
	total := new(uint256.Int).SetUint64(tx.Gas)
	total.Mul(total, tx.GasPrice)
	total.Add(total, tx.Value)
	return total
}

func (tx DepositTx) GetAccessList() AccessList {
	return nil
}

// NewDepositTransaction creates a deposit transaction
func NewDepositTransaction(to common.Address, mint *uint256.Int, amount *uint256.Int, gasLimit uint64, gasPrice *uint256.Int, data []byte) *DepositTx {
	return &DepositTx{
		// NOTE: Does the SourceHash get added some time after this function is called?
		// NOTE: from comes from TransactionMisc.from which is of type atomic.Value
		// nil means contract creation
		To: &to,
		// Mint is minted on L2, locked on L1, nil if no minting.
		Mint: mint,
		// Value is transferred from L2 balance, executed after Mint (if any)
		Value: amount,
		// gas limit
		Gas: gasLimit,
		// wei per gas
		GasPrice: gasPrice,
		Data:     data,
	}
}

// func (tx *DepositTx) txType() byte           { return DepositTxType }

// func (tx *DepositTx) setSignatureValues(chainID, v, r, s *big.Int) {
// 	panic("deposit tx does not have a signature")
// }

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx DepositTx) copy() *DepositTx {
	cpy := &DepositTx{
		SourceHash: tx.SourceHash,
		From:       tx.From,
		To:         tx.To,
		Mint:       nil,
		Value:      new(uint256.Int),
		Gas:        tx.Gas,
		Data:       common.CopyBytes(tx.Data),
	}
	if tx.Mint != nil {
		cpy.Mint = new(uint256.Int).Set(tx.Mint)
	}
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	return cpy
}

// AsMessage returns the transaction as a core.Message.
func (tx DepositTx) AsMessage(s Signer, _ *big.Int) (Message, error) {
	msg := Message{
		nonce:      DepositsNonce,
		gasLimit:   tx.Gas,
		gasPrice:   *tx.GasPrice,
		tip:        *tx.GasPrice,
		feeCap:     *tx.GasPrice,
		from:       tx.From,
		to:         tx.To,
		amount:     *tx.Value,
		data:       tx.Data,
		accessList: nil,
		checkNonce: true,
	}

	return msg, nil
}

func (tx *DepositTx) Sender(signer Signer) (common.Address, error) {
	return tx.From, nil
}
