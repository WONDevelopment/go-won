// Copyright 2016 The go-ethereum Authors
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

package vm

import (
	"math/big"

	"github.com/worldopennet/go-won/common"
	"github.com/worldopennet/go-won/core/types"
)

// StateDB is an EVM database for full state querying.
type StateDB interface {
	CreateAccount(common.Address)

	SubBalance(common.Address, *big.Int)
	AddBalance(common.Address, *big.Int)
	GetBalance(common.Address) *big.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	AddRefund(uint64)
	GetRefund() uint64

	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	// Exist reports whwon the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(common.Address) bool
	// Empty returns whwon the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(common.Address) bool

	RevertToSnapshot(int)
	Snapshot() int

	AddLog(*types.Log)
	AddPreimage(common.Hash, []byte)

	ForEachStorage(common.Address, func(common.Hash, common.Hash) bool)

	SetKycLevel(addr common.Address, level uint32)
	GetKycLevel(addr common.Address) uint32
	SetKycZone(addr common.Address, zone uint32)
	GetKycZone(addr common.Address) uint32
	SetKycProvider(addr common.Address, provider common.Address)
	GetKycProvider(addr common.Address) common.Address

	KycProviderExists(addr common.Address) bool
	GetKycProviderCount() int64
	AddKycProvider(addr common.Address)
	RemoveKycProvider(addr common.Address)
	SetKycProviderProposol(addr common.Address, st *big.Int, pt *big.Int)
	SetVoteForKycProviderProposol(addr common.Address) bool
	GetKycProviderProposol() (common.Address, *big.Int, *big.Int, *big.Int, *big.Int)
	GetKycProviderList() []common.Address
	TxKycValidate(addr common.Address, dst common.Address, amount *big.Int) bool
	IsContractAddress(address common.Address) bool
	RegisterProducer(pb *common.Address, url string)


	UpdateProducerTotalVotes(pb *common.Address, stake *big.Int)
	UpdateProducerActive(pb *common.Address, val bool)
	UpdateProducerLocation(pb *common.Address, val *big.Int)
	GetProducerInfo(pb *common.Address) *common.ProducerInfo
	GetProducerTopList() []common.Address
	GetProducerList(startPos int64, number int64) []common.Address
	SetVoterStaking(myAddr *common.Address, stake *big.Int)
	GetVoterStaking(myAddr *common.Address) (stake *big.Int)
	SetVoterProducers(myAddr *common.Address, pbs []common.Address)
	GetVoterProducers(myAddr *common.Address) (pbs []common.Address)
	SetRefundRequestInfo(myAddr *common.Address, stake *big.Int, requestTime *big.Int)
	GetRefundRequestInfo(myAddr *common.Address) (stake *big.Int, requestTime *big.Int)
	SetDposVoterLastVoteWeight(myAddr *common.Address, weight *big.Int)
	GetDposVoterLastVoteWeight(myAddr *common.Address) (weight *big.Int)
	GetDposLastProducerScheduleUpdateTime() *big.Int
	SetDposLastProducerScheduleUpdateTime(val *big.Int)
	GetDposTopProducerElectedDone() *big.Int
	SetDposTopProducerElectedDone(val *big.Int)
	GetDposTotalActivatedStake() (*big.Int)
	SetDposTotalActivatedStake(val *big.Int)
	GetDposThreshActivatedStakeTime()(*big.Int)
	SetDposThreshActivatedStakeTime(val *big.Int)
}

// CallContext provides a basic interface for the EVM calling conventions. The EVM EVM
// depends on this context being implemented for doing subcalls and initialising new EVM contracts.
type CallContext interface {
	// Call another contract
	Call(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// Take another's contract code and execute within our own context
	CallCode(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// Same as CallCode except sender and value is propagated from parent to child scope
	DelegateCall(env *EVM, me ContractRef, addr common.Address, data []byte, gas *big.Int) ([]byte, error)
	// Create a new contract
	Create(env *EVM, me ContractRef, data []byte, gas, value *big.Int) ([]byte, common.Address, error)
}
