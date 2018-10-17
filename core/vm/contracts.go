// Copyright 2014 The go-ethereum Authors
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
	"crypto/sha256"
	"errors"
	"math/big"

	"encoding/binary"
	"github.com/worldopennetwork/go-won/common"
	"github.com/worldopennetwork/go-won/common/math"
	"github.com/worldopennetwork/go-won/crypto"
	"github.com/worldopennetwork/go-won/crypto/bn256"
	"github.com/worldopennetwork/go-won/params"
	"golang.org/x/crypto/ripemd160"
)

// PrecompiledContract is the basic interface for native Go contracts. The implementation
// requires a deterministic gas count based on the input size of the Run method of the
// contract.
type PrecompiledContract interface {
	RequiredGas(input []byte) uint64  // RequiredPrice calculates the contract gas use
	Run(input []byte) ([]byte, error) // Run runs the precompiled contract
}

// PrecompiledContractsHomestead contains the default set of pre-compiled WorldOpenNetwork
// contracts used in the Frontier and Homestead releases.
var PrecompiledContractsHomestead = map[common.Address]PrecompiledContract{
	common.BytesToAddress([]byte{1}): &ecrecover{},
	common.BytesToAddress([]byte{2}): &sha256hash{},
	common.BytesToAddress([]byte{3}): &ripemd160hash{},
	common.BytesToAddress([]byte{4}): &dataCopy{},
}

// PrecompiledContractsByzantium contains the default set of pre-compiled WorldOpenNetwork
// contracts used in the Byzantium release.
var PrecompiledContractsByzantium = map[common.Address]PrecompiledContract{
	common.BytesToAddress([]byte{1}): &ecrecover{},
	common.BytesToAddress([]byte{2}): &sha256hash{},
	common.BytesToAddress([]byte{3}): &ripemd160hash{},
	common.BytesToAddress([]byte{4}): &dataCopy{},
	common.BytesToAddress([]byte{5}): &bigModExp{},
	common.BytesToAddress([]byte{6}): &bn256Add{},
	common.BytesToAddress([]byte{7}): &bn256ScalarMul{},
	common.BytesToAddress([]byte{8}): &bn256Pairing{},
}

var KycContractAddress = common.BytesToAddress([]byte{9})
var DposActivatedStakeThreshold = big.NewInt(0).Mul(big.NewInt(15000000), big.NewInt(params.WON))

const KycMethodSet = 1
const KycMethodProviderVoteProposal = 2
const KycMethodVote = 3
const DposMethodRegProds = 4
const DposMethodRmvProds = 5
const DposMethodAddStake = 6
const DposMethodSubStake = 7
const DposMethodProdsVote = 8
const DposMethodRefund = 9

// RunPrecompiledContract runs and evaluates the output of a precompiled contract.
func RunPrecompiledContract(p PrecompiledContract, input []byte, contract *Contract) (ret []byte, err error) {
	gas := p.RequiredGas(input)
	if contract.UseGas(gas) {
		return p.Run(input)
	}
	return nil, ErrOutOfGas
}

// ECRECOVER implemented as a native contract.
type ecrecover struct{}

func (c *ecrecover) RequiredGas(input []byte) uint64 {
	return params.EcrecoverGas
}

func (c *ecrecover) Run(input []byte) ([]byte, error) {
	const ecRecoverInputLength = 128

	input = common.RightPadBytes(input, ecRecoverInputLength)
	// "input" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(input[64:96])
	s := new(big.Int).SetBytes(input[96:128])
	v := input[63] - 27

	// tighter sig s values input homestead only apply to tx sigs
	if !allZero(input[32:63]) || !crypto.ValidateSignatureValues(v, r, s, false) {
		return nil, nil
	}
	// v needs to be at the end for libsecp256k1
	pubKey, err := crypto.Ecrecover(input[:32], append(input[64:128], v))
	// make sure the public key is a valid one
	if err != nil {
		return nil, nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(crypto.Keccak256(pubKey[1:])[12:], 32), nil
}

// SHA256 implemented as a native contract.
type sha256hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *sha256hash) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.Sha256PerWordGas + params.Sha256BaseGas
}
func (c *sha256hash) Run(input []byte) ([]byte, error) {
	h := sha256.Sum256(input)
	return h[:], nil
}

// RIPMED160 implemented as a native contract.
type ripemd160hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *ripemd160hash) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.Ripemd160PerWordGas + params.Ripemd160BaseGas
}
func (c *ripemd160hash) Run(input []byte) ([]byte, error) {
	ripemd := ripemd160.New()
	ripemd.Write(input)
	return common.LeftPadBytes(ripemd.Sum(nil), 32), nil
}

// data copy implemented as a native contract.
type dataCopy struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *dataCopy) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.IdentityPerWordGas + params.IdentityBaseGas
}
func (c *dataCopy) Run(in []byte) ([]byte, error) {
	return in, nil
}

// bigModExp implements a native big integer exponential modular operation.
type bigModExp struct{}

var (
	big1      = big.NewInt(1)
	big4      = big.NewInt(4)
	big8      = big.NewInt(8)
	big16     = big.NewInt(16)
	big32     = big.NewInt(32)
	big64     = big.NewInt(64)
	big96     = big.NewInt(96)
	big480    = big.NewInt(480)
	big1024   = big.NewInt(1024)
	big3072   = big.NewInt(3072)
	big199680 = big.NewInt(199680)
)

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bigModExp) RequiredGas(input []byte) uint64 {
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32))
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32))
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32))
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Retrieve the head 32 bytes of exp for the adjusted exponent length
	var expHead *big.Int
	if big.NewInt(int64(len(input))).Cmp(baseLen) <= 0 {
		expHead = new(big.Int)
	} else {
		if expLen.Cmp(big32) > 0 {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), 32))
		} else {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), expLen.Uint64()))
		}
	}
	// Calculate the adjusted exponent length
	var msb int
	if bitlen := expHead.BitLen(); bitlen > 0 {
		msb = bitlen - 1
	}
	adjExpLen := new(big.Int)
	if expLen.Cmp(big32) > 0 {
		adjExpLen.Sub(expLen, big32)
		adjExpLen.Mul(big8, adjExpLen)
	}
	adjExpLen.Add(adjExpLen, big.NewInt(int64(msb)))

	// Calculate the gas cost of the operation
	gas := new(big.Int).Set(math.BigMax(modLen, baseLen))
	switch {
	case gas.Cmp(big64) <= 0:
		gas.Mul(gas, gas)
	case gas.Cmp(big1024) <= 0:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big4),
			new(big.Int).Sub(new(big.Int).Mul(big96, gas), big3072),
		)
	default:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big16),
			new(big.Int).Sub(new(big.Int).Mul(big480, gas), big199680),
		)
	}

	gas.Mul(gas, math.BigMax(adjExpLen, big1))
	gas.Div(gas, new(big.Int).SetUint64(params.ModExpQuadCoeffDiv))

	if gas.BitLen() > 64 {
		return math.MaxUint64
	}
	return gas.Uint64()
}

func (c *bigModExp) Run(input []byte) ([]byte, error) {
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32)).Uint64()
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32)).Uint64()
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32)).Uint64()
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Handle a special case when both the base and mod length is zero
	if baseLen == 0 && modLen == 0 {
		return []byte{}, nil
	}
	// Retrieve the operands and execute the exponentiation
	var (
		base = new(big.Int).SetBytes(getData(input, 0, baseLen))
		exp  = new(big.Int).SetBytes(getData(input, baseLen, expLen))
		mod  = new(big.Int).SetBytes(getData(input, baseLen+expLen, modLen))
	)
	if mod.BitLen() == 0 {
		// Modulo 0 is undefined, return zero
		return common.LeftPadBytes([]byte{}, int(modLen)), nil
	}
	return common.LeftPadBytes(base.Exp(base, exp, mod).Bytes(), int(modLen)), nil
}

// newCurvePoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newCurvePoint(blob []byte) (*bn256.G1, error) {
	p := new(bn256.G1)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

// newTwistPoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newTwistPoint(blob []byte) (*bn256.G2, error) {
	p := new(bn256.G2)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

// bn256Add implements a native elliptic curve point addition.
type bn256Add struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256Add) RequiredGas(input []byte) uint64 {
	return params.Bn256AddGas
}

func (c *bn256Add) Run(input []byte) ([]byte, error) {
	x, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	y, err := newCurvePoint(getData(input, 64, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.Add(x, y)
	return res.Marshal(), nil
}

// bn256ScalarMul implements a native elliptic curve scalar multiplication.
type bn256ScalarMul struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256ScalarMul) RequiredGas(input []byte) uint64 {
	return params.Bn256ScalarMulGas
}

func (c *bn256ScalarMul) Run(input []byte) ([]byte, error) {
	p, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.ScalarMult(p, new(big.Int).SetBytes(getData(input, 64, 32)))
	return res.Marshal(), nil
}

var (
	// true32Byte is returned if the bn256 pairing check succeeds.
	true32Byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	// false32Byte is returned if the bn256 pairing check fails.
	false32Byte = make([]byte, 32)

	// errBadPairingInput is returned if the bn256 pairing input is invalid.
	errBadPairingInput = errors.New("bad elliptic curve pairing size")
)

// bn256Pairing implements a pairing pre-compile for the bn256 curve
type bn256Pairing struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256Pairing) RequiredGas(input []byte) uint64 {
	return params.Bn256PairingBaseGas + uint64(len(input)/192)*params.Bn256PairingPerPointGas
}

func (c *bn256Pairing) Run(input []byte) ([]byte, error) {
	// Handle some corner cases cheaply
	if len(input)%192 > 0 {
		return nil, errBadPairingInput
	}
	// Convert the input into a set of coordinates
	var (
		cs []*bn256.G1
		ts []*bn256.G2
	)
	for i := 0; i < len(input); i += 192 {
		c, err := newCurvePoint(input[i : i+64])
		if err != nil {
			return nil, err
		}
		t, err := newTwistPoint(input[i+64 : i+192])
		if err != nil {
			return nil, err
		}
		cs = append(cs, c)
		ts = append(ts, t)
	}
	// Execute the pairing checks and return the results
	if bn256.PairingCheck(cs, ts) {
		return true32Byte, nil
	}
	return false32Byte, nil
}

func setContractKycInfoAtCreate(evm *EVM, caller common.Address, address common.Address) {

	humanCaller := caller
	for evm.StateDB.IsContractAddress(humanCaller) {
		humanCaller = evm.StateDB.GetKycProvider(humanCaller)
	}
	evm.StateDB.SetKycProvider(address, humanCaller)
	evm.StateDB.SetKycZone(address, evm.StateDB.GetKycZone(caller))
	evm.StateDB.SetKycLevel(address, evm.StateDB.GetKycLevel(caller))
}

func kycSetForAddress(evm *EVM, contract *Contract, address common.Address, level uint32, zone uint32) ([]byte, error) {

	evm.StateDB.SetKycProvider(address, contract.caller.Address())
	evm.StateDB.SetKycZone(address, zone)
	evm.StateDB.SetKycLevel(address, level)
	return nil, nil
}

func kycSetDefaultInfoForProvider(evm *EVM, addr common.Address) {
	evm.StateDB.SetKycProvider(addr, addr)
	evm.StateDB.SetKycZone(addr, 99999999)
	evm.StateDB.SetKycLevel(addr, 99999999)
}

func kycStartProviderProposal(evm *EVM, contract *Contract, addr common.Address, pt uint64) ([]byte, error) {

	if evm.StateDB.IsContractAddress(addr) {
		return nil, ErrOutOfGas
	}

	curCount := evm.StateDB.GetKycProviderCount()

	if pt != 1 && pt != 2 {
		return nil, ErrOutOfGas
	}

	if curCount == 0 && pt == 2 {
		return nil, ErrOutOfGas
	}

	//must be a provider to do the proposal
	if !evm.StateDB.KycProviderExists(contract.caller.Address()) {
		return nil, ErrOutOfGas
	}

	if pt == 1 && curCount > 0 && evm.StateDB.KycProviderExists(addr) {
		return nil, ErrOutOfGas
	}

	if pt == 2 && curCount <= 0 && !evm.StateDB.KycProviderExists(addr) {
		return nil, ErrOutOfGas
	}

	if curCount < 2 {
		if pt == 1 {
			evm.StateDB.AddKycProvider(addr)
			kycSetDefaultInfoForProvider(evm, addr)

		} else if pt == 2 {
			evm.StateDB.RemoveKycProvider(addr)
		}
		return nil, nil
	}

	hvAddr, hvTime, hvVoteTotal, _, iVoted, _ := evm.StateDB.GetKycProviderProposol()

	//check if the last one is expired or finished .
	if hvAddr != common.BytesToAddress([]byte{0}) && hvTime.Uint64()+86400 > evm.Time.Uint64() && iVoted.Uint64() <= hvVoteTotal.Uint64()/2 {
		//still in voting, not expired
		return nil, ErrOutOfGas
	}

	ptv := big.NewInt(0)
	ptv.SetUint64(pt)
	evm.StateDB.SetKycProviderProposol(addr, evm.Time, ptv)
	evm.StateDB.SetVoteForKycProviderProposol(contract.caller.Address(), 0)
	return nil, nil

}

func kycVoteForProvider(evm *EVM, contract *Contract, nay uint16) ([]byte, error) {

	hvAddr, hvTime, hvVoteTotal, pt, iVoted, _ := evm.StateDB.GetKycProviderProposol()
	//check if the last one is expired or finished .
	if hvAddr != common.BytesToAddress([]byte{0}) && hvTime.Uint64()+86400 > evm.Time.Uint64() && iVoted.Uint64() <= hvVoteTotal.Uint64()/2 {
		//still in voting, not expired
		voteOk := evm.StateDB.SetVoteForKycProviderProposol(contract.caller.Address(), nay)
		if !voteOk {
			return nil, ErrOutOfGas
		}

		_, _, _, _, iVoted, _ := evm.StateDB.GetKycProviderProposol()

		if iVoted.Uint64() > hvVoteTotal.Uint64()/2 {
			if pt.Int64() == 1 && !evm.StateDB.KycProviderExists(hvAddr) {
				evm.StateDB.AddKycProvider(hvAddr)
				kycSetDefaultInfoForProvider(evm, hvAddr)
			} else if pt.Int64() == 2 && evm.StateDB.KycProviderExists(hvAddr) {
				evm.StateDB.RemoveKycProvider(hvAddr)
			}

			evm.StateDB.SetKycProviderProposol(common.BytesToAddress([]byte{0}), common.Big0, common.Big0)
		}

		return nil, nil
	}

	return nil, ErrOutOfGas
}

func dposRegisterProducer(evm *EVM, contract *Contract, from common.Address, url string) ([]byte, error) {

	evm.StateDB.RegisterProducer(&from, url)
	evm.StateDB.SetDposTopProducerElectedDone(common.Big0)

	return nil, nil
}

func dposUnregisterUnproducer(evm *EVM, contract *Contract, from common.Address) ([]byte, error) {
	pi := evm.StateDB.GetProducerInfo(&from)
	if pi != nil && pi.IsActive {
		evm.StateDB.UpdateProducerActive(&from, false)
		evm.StateDB.SetDposTopProducerElectedDone(common.Big0)
	}
	return nil, nil
}

func calcVoteWeight(value *big.Int, ct *big.Int) *big.Int {

	block_timestamp_epoch := int64(1534154327)

	/// TODO subtract 2080 brings the large numbers closer to this decade
	//double weight = int64_t( (now() - (block_timestamp::block_timestamp_epoch / 1000)) / (seconds_per_day * 7) )  / double( 52 );
	//return double(staked) * std::pow( 2, weigh	t );
	weight := (ct.Int64() - block_timestamp_epoch) / (24 * 3600) / 52
	ret := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(weight), common.Big0)
	ret = big.NewInt(0).Mul(value, ret)
	return ret
}

func doChangeProducerVoteingWeight(evm *EVM, from common.Address, newValue *big.Int, ct *big.Int) {
	vw := calcVoteWeight(newValue, ct)
	lastVw := evm.StateDB.GetDposVoterLastVoteWeight(&from)
	pbs := evm.StateDB.GetVoterProducers(&from)

	for _, pb := range pbs {
		pi := evm.StateDB.GetProducerInfo(&pb)
		pi.TotalVotes = big.NewInt(0).Sub(pi.TotalVotes, lastVw)
		pi.TotalVotes = big.NewInt(0).Add(pi.TotalVotes, vw)
		evm.StateDB.UpdateProducerTotalVotes(&pb, pi.TotalVotes)
	}

	evm.StateDB.SetDposVoterLastVoteWeight(&from, vw)
}

func dposIncStake(evm *EVM, contract *Contract, from common.Address, value *big.Int) ([]byte, error) {

	if value.Cmp(common.Big0) <= 0 {
		return nil, ErrOutOfGas
	}

	lastVw := evm.StateDB.GetDposVoterLastVoteWeight(&from)

	oldValue := evm.StateDB.GetVoterStaking(&from)
	newValue := big.NewInt(0).Add(oldValue, value)

	//check refunding stake, cancel some of stake if we inc
	stake, rt := evm.StateDB.GetRefundRequestInfo(&from)

	if stake.Cmp(value) >= 0 {

		//all is from refunding stake
		newRefunding := big.NewInt(0).Sub(stake, value)
		evm.StateDB.SetRefundRequestInfo(&from, newRefunding, rt)

	} else {
		needValue := big.NewInt(0).Sub(value, stake)

		// Fail if we're trying to transfer more than the available balance
		if !evm.CanTransfer(evm.StateDB, from, needValue) {
			return nil, ErrOutOfGas
		}

		if !evm.StateDB.TxKycValidate(from, KycContractAddress, needValue) {

			return nil, ErrOutOfGas
		}

		if needValue.Sign() < 0 {
			return nil, ErrOutOfGas
		}

		evm.StateDB.SetRefundRequestInfo(&from, common.Big0, common.Big0)
		evm.StateDB.AddBalance(KycContractAddress, needValue)
		evm.StateDB.SubBalance(from, needValue)

	}

	evm.StateDB.SetVoterStaking(&from, newValue)
	doChangeProducerVoteingWeight(evm, from, newValue, evm.Time)

	/**
	 * The first time someone votes we calculate and set last_vote_weight, since they cannot unstake until
	 * after total_activated_stake hits threshold, we can use last_vote_weight to determine that this is
	 * their first vote and should consider their stake activated.
	 */
	if lastVw.Cmp(common.Big0) <= 0 {
		totalActivatedState := evm.StateDB.GetDposTotalActivatedStake()
		totalActivatedState = big.NewInt(0).Add(totalActivatedState, value)
		evm.StateDB.SetDposTotalActivatedStake(totalActivatedState)
	}
	//evm.StateDB.get
	evm.StateDB.SetDposTopProducerElectedDone(common.Big0)

	return nil, nil
}

func dposDecStake(evm *EVM, contract *Contract, from common.Address, value *big.Int) ([]byte, error) {

	if value.Cmp(common.Big0) <= 0 {
		return nil, ErrOutOfGas
	}

	//don't allow dec stake if not activated
	//
	totalActivatedState := evm.StateDB.GetDposTotalActivatedStake()
	if totalActivatedState.Cmp(DposActivatedStakeThreshold) < 0 {
		return nil, ErrOutOfGas
	}

	oldValue := evm.StateDB.GetVoterStaking(&from)
	//import check .
	if oldValue.Cmp(value) < 0 {
		return nil, ErrOutOfGas
	}

	newValue := big.NewInt(0).Sub(oldValue, value)
	evm.StateDB.SetVoterStaking(&from, newValue)

	doChangeProducerVoteingWeight(evm, from, newValue, evm.Time)

	stake, _ := evm.StateDB.GetRefundRequestInfo(&from)
	stake = big.NewInt(0).Add(stake, value)
	evm.StateDB.SetRefundRequestInfo(&from, stake, evm.Time)

	evm.StateDB.SetDposTopProducerElectedDone(common.Big0)

	return nil, nil
}

func dposVoteForProducer(evm *EVM, contract *Contract, from common.Address, tos []common.Address) ([]byte, error) {

	evm.StateDB.SetDposTopProducerElectedDone(common.Big0)

	//cancel the old voting for old producers
	doChangeProducerVoteingWeight(evm, from, common.Big0, evm.Time)

	validPbs := make([]common.Address, 0)

	for _, pb := range tos {
		pi := evm.StateDB.GetProducerInfo(&pb)
		if pi != nil && pi.IsActive {
			validPbs = append(validPbs, pb)
		}
	}

	evm.StateDB.SetVoterProducers(&from, validPbs)

	newValue := evm.StateDB.GetVoterStaking(&from)

	doChangeProducerVoteingWeight(evm, from, newValue, evm.Time)

	return nil, nil
}

func dposRefund(evm *EVM, contract *Contract, from common.Address) ([]byte, error) {

	stake, st := evm.StateDB.GetRefundRequestInfo(&from)

	if stake != common.Big0 && evm.Time.Uint64() > st.Uint64()+86400*3 {

		// Fail if we're trying to transfer more than the available balance
		if !evm.CanTransfer(evm.StateDB, KycContractAddress, stake) {
			return nil, ErrOutOfGas
		}

		if !evm.StateDB.TxKycValidate(KycContractAddress, from, stake) {

			return nil, ErrOutOfGas
		}

		evm.StateDB.SetRefundRequestInfo(&from, common.Big0, common.Big0)
		evm.StateDB.AddBalance(from, stake)
		evm.StateDB.SubBalance(KycContractAddress, stake)
		return nil, nil
	}
	return nil, ErrOutOfGas
}

func kycExecute(evm *EVM, contract *Contract, input []byte) ([]byte, error) {

	if input == nil || len(input) < 4 {
		//for transfer value only
		return nil, nil
	}

	if contract.UseGas(3000) {

		if evm.StateDB.IsContractAddress(contract.caller.Address()) {
			return nil, ErrOutOfGas
		}

		funcid := binary.BigEndian.Uint32(input[0:4])
		if funcid == KycMethodSet {
			if !evm.StateDB.KycProviderExists(contract.caller.Address()) {
				return nil, ErrOutOfGas
			}
			address := common.BytesToAddress(input[4:24])
			level := binary.BigEndian.Uint32(input[24:28])
			zone := binary.BigEndian.Uint32(input[28:32])
			return kycSetForAddress(evm, contract, address, level, zone)
		} else if funcid == KycMethodProviderVoteProposal {
			if !evm.StateDB.KycProviderExists(contract.caller.Address()) {
				return nil, ErrOutOfGas
			}
			address := common.BytesToAddress(input[4:24])
			pt := binary.BigEndian.Uint64(input[24:])
			return kycStartProviderProposal(evm, contract, address, pt)
		} else if funcid == KycMethodVote {
			if !evm.StateDB.KycProviderExists(contract.caller.Address()) {
				return nil, ErrOutOfGas
			}
			nay := binary.BigEndian.Uint16(input[4:])
			return kycVoteForProvider(evm, contract, nay)
		} else if funcid == DposMethodRegProds {
			url := string(input[4:])
			return dposRegisterProducer(evm, contract, contract.caller.Address(), url)
		} else if funcid == DposMethodRmvProds {
			return dposUnregisterUnproducer(evm, contract, contract.caller.Address())
		} else if funcid == DposMethodAddStake {
			value := common.BytesToHash(input[4:]).Big()
			return dposIncStake(evm, contract, contract.caller.Address(), value)
		} else if funcid == DposMethodSubStake {
			value := common.BytesToHash(input[4:]).Big()
			return dposDecStake(evm, contract, contract.caller.Address(), value)
		} else if funcid == DposMethodProdsVote {
			numaddr := (len(input) - 4) / 20
			if numaddr < 0 {
				return nil, nil
			}
			tos := make([]common.Address, 0)
			for i := 0; i < numaddr; i++ {
				addr := common.BytesToAddress(input[4+i*20 : 4+i*20+20])
				tos = append(tos, addr)
			}
			return dposVoteForProducer(evm, contract, contract.caller.Address(), tos)
		} else if funcid == DposMethodRefund {
			return dposRefund(evm, contract, contract.caller.Address())
		}

	}
	return nil, ErrOutOfGas
}
