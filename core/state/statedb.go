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

// Package state provides a caching layer atop the WorldOpenNetwork state trie.
package state

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/worldopennetwork/go-won/common"
	"github.com/worldopennetwork/go-won/core/types"
	"github.com/worldopennetwork/go-won/core/vm"
	"github.com/worldopennetwork/go-won/crypto"
	"github.com/worldopennetwork/go-won/log"
	"github.com/worldopennetwork/go-won/rlp"
	"github.com/worldopennetwork/go-won/trie"
)

type revision struct {
	id           int
	journalIndex int
}

var (
	// emptyState is the known hash of an empty state trie entry.
	emptyState = crypto.Keccak256Hash(nil)

	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode = crypto.Keccak256Hash(nil)

	kycProviderStartHash   = int64(10000000000)
	kycVoterStartHash      = int64(20000000000)
	kycVoteResultStartHash = int64(21000000000)
	maxKycProviderCount    = int64(10000000000)

	dposProducerAllStartKey = int64(30000000000)

	kycProviderNumberKey       = common.BigToHash(common.Big1)
	kycProposalAddressKey      = common.BigToHash(common.Big2)
	kycProposalStartTimeKey    = common.BigToHash(common.Big3)
	kycProposalVoteTotalKey    = common.BigToHash(big.NewInt(4))
	kycProposalAlreadyVotedKey = common.BigToHash(big.NewInt(5))

	//to active and meet the minimium vote for starting
	dposTotalActivatedStakeKey            = common.BigToHash(big.NewInt(100))
	dposProducerCountKey                  = common.BigToHash(big.NewInt(101))
	dposThreshActivatedStakeTimeKey       = common.BigToHash(big.NewInt(102))
	dposTotalProducerVoteWeightKey        = common.BigToHash(big.NewInt(103))
	dposLastProducerScheduleUpdateTimeKey = common.BigToHash(big.NewInt(104))
	dposTopProducerElectedDoneKey         = common.BigToHash(big.NewInt(105))

	dposProducerURLKey        = int64(0x1)
	dposProducerURLKeyHigh    = int64(0x5)
	dposProducerTotalVotesKey = int64(0x2)
	dposProducerActiveKey     = int64(0x3)
	dposProducerLocationKey   = int64(0x4)

	dposVoterStakingKey        = int64(0x70)
	dposVoterLastVoteWeightKey = int64(0x71)

	dposVoterRefundAmountBeginKey     = int64(0x80)
	dposVoterRefundReqestTimeBeginKey = int64(0x81)

	dposVoterCountKey          = int64(0x90)
	dposVoterBpAddressBeginKey = int64(0x91)
)

// StateDBs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type StateDB struct {
	db   Database
	trie Trie

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects      map[common.Address]*stateObject
	stateObjectsDirty map[common.Address]struct{}

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// The refund counter, also used by state transitioning.
	refund uint64

	thash, bhash common.Hash
	txIndex      int
	logs         map[common.Hash][]*types.Log
	logSize      uint

	preimages map[common.Hash][]byte

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int

	lock sync.Mutex
}

// Create a new state from a given trie.
func New(root common.Hash, db Database) (*StateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	return &StateDB{
		db:                db,
		trie:              tr,
		stateObjects:      make(map[common.Address]*stateObject),
		stateObjectsDirty: make(map[common.Address]struct{}),
		logs:              make(map[common.Hash][]*types.Log),
		preimages:         make(map[common.Hash][]byte),
		journal:           newJournal(),
	}, nil
}

// setError remembers the first non-nil error it is called with.
func (self *StateDB) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *StateDB) Error() error {
	return self.dbErr
}

// Reset clears out all ephemeral state objects from the state db, but keeps
// the underlying state trie to avoid reloading data for the next operations.
func (self *StateDB) Reset(root common.Hash) error {
	tr, err := self.db.OpenTrie(root)
	if err != nil {
		return err
	}
	self.trie = tr
	self.stateObjects = make(map[common.Address]*stateObject)
	self.stateObjectsDirty = make(map[common.Address]struct{})
	self.thash = common.Hash{}
	self.bhash = common.Hash{}
	self.txIndex = 0
	self.logs = make(map[common.Hash][]*types.Log)
	self.logSize = 0
	self.preimages = make(map[common.Hash][]byte)
	self.clearJournalAndRefund()
	return nil
}

func (self *StateDB) AddLog(log *types.Log) {
	self.journal.append(addLogChange{txhash: self.thash})

	log.TxHash = self.thash
	log.BlockHash = self.bhash
	log.TxIndex = uint(self.txIndex)
	log.Index = self.logSize
	self.logs[self.thash] = append(self.logs[self.thash], log)
	self.logSize++
}

func (self *StateDB) GetLogs(hash common.Hash) []*types.Log {
	return self.logs[hash]
}

func (self *StateDB) Logs() []*types.Log {
	var logs []*types.Log
	for _, lgs := range self.logs {
		logs = append(logs, lgs...)
	}
	return logs
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (self *StateDB) AddPreimage(hash common.Hash, preimage []byte) {
	if _, ok := self.preimages[hash]; !ok {
		self.journal.append(addPreimageChange{hash: hash})
		pi := make([]byte, len(preimage))
		copy(pi, preimage)
		self.preimages[hash] = pi
	}
}

// Preimages returns a list of SHA3 preimages that have been submitted.
func (self *StateDB) Preimages() map[common.Hash][]byte {
	return self.preimages
}

func (self *StateDB) AddRefund(gas uint64) {
	self.journal.append(refundChange{prev: self.refund})
	self.refund += gas
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for suicided accounts.
func (self *StateDB) Exist(addr common.Address) bool {
	return self.getStateObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (self *StateDB) Empty(addr common.Address) bool {
	so := self.getStateObject(addr)
	return so == nil || so.empty()
}

// Retrieve the balance from the given address or 0 if object not found
func (self *StateDB) GetBalance(addr common.Address) *big.Int {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return common.Big0
}

func (self *StateDB) GetNonce(addr common.Address) uint64 {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}
	return 0
}

func (self *StateDB) GetCode(addr common.Address) []byte {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		//	log.Debug("Success to GetCode state object", "addr", addr)
		return stateObject.Code(self.db)
	}

	log.Debug("Failed to GetCode state object", "addr", addr)

	return nil
}

func (self *StateDB) GetCodeSize(addr common.Address) int {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return 0
	}
	if stateObject.code != nil {
		return len(stateObject.code)
	}
	size, err := self.db.ContractCodeSize(stateObject.addrHash, common.BytesToHash(stateObject.CodeHash()))
	if err != nil {
		self.setError(err)
	}
	return size
}

func (self *StateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())
}

func (self *StateDB) GetState(addr common.Address, bhash common.Hash) common.Hash {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetState(self.db, bhash)
	}
	return common.Hash{}
}

// Database retrieves the low level database supporting the lower level trie ops.
func (self *StateDB) Database() Database {
	return self.db
}

// StorageTrie returns the storage trie of an account.
// The return value is a copy and is nil for non-existent accounts.
func (self *StateDB) StorageTrie(addr common.Address) Trie {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return nil
	}
	cpy := stateObject.deepCopy(self)
	return cpy.updateTrie(self.db)
}

func (self *StateDB) HasSuicided(addr common.Address) bool {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.suicided
	}
	return false
}

/*
 * SETTERS
 */

// AddBalance adds amount to the account associated with addr.
func (self *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

// SubBalance subtracts amount from the account associated with addr.
func (self *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (self *StateDB) SetBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(amount)
	}
}

func (self *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (self *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
}

func (self *StateDB) SetState(addr common.Address, key, value common.Hash) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(self.db, key, value)
	}
}

// Suicide marks the given account as suicided.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after Suicide.
func (self *StateDB) Suicide(addr common.Address) bool {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return false
	}
	self.journal.append(suicideChange{
		account:     &addr,
		prev:        stateObject.suicided,
		prevbalance: new(big.Int).Set(stateObject.Balance()),
	})
	stateObject.markSuicided()
	stateObject.data.Balance = new(big.Int)

	return true
}

//
// Setting, updating & deleting state object methods.
//

// updateStateObject writes the given object to the trie.
func (self *StateDB) updateStateObject(stateObject *stateObject) {
	addr := stateObject.Address()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	self.setError(self.trie.TryUpdate(addr[:], data))
}

// deleteStateObject removes the given object from the state trie.
func (self *StateDB) deleteStateObject(stateObject *stateObject) {
	stateObject.deleted = true
	addr := stateObject.Address()
	self.setError(self.trie.TryDelete(addr[:]))
}

// Retrieve a state object given my the address. Returns nil if not found.
func (self *StateDB) getStateObject(addr common.Address) (stateObject *stateObject) {
	// Prefer 'live' objects.
	if obj := self.stateObjects[addr]; obj != nil {
		if obj.deleted {
			return nil
		}
		return obj
	}

	// Load the object from the database.
	enc, err := self.trie.TryGet(addr[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data Account
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newObject(self, addr, data)
	self.setStateObject(obj)
	return obj
}

func (self *StateDB) setStateObject(object *stateObject) {
	self.stateObjects[object.Address()] = object
}

// Retrieve a state object or create a new state object if nil.
func (self *StateDB) GetOrNewStateObject(addr common.Address) *stateObject {
	stateObject := self.getStateObject(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject, _ = self.createObject(addr)
	}
	return stateObject
}

// createObject creates a new state object. If there is an existing account with
// the given address, it is overwritten and returned as the second return value.
func (self *StateDB) createObject(addr common.Address) (newobj, prev *stateObject) {
	prev = self.getStateObject(addr)
	newobj = newObject(self, addr, Account{})
	newobj.setNonce(0) // sets the object to dirty
	if prev == nil {
		self.journal.append(createObjectChange{account: &addr})
	} else {
		self.journal.append(resetObjectChange{prev: prev})
	}
	self.setStateObject(newobj)
	return newobj, prev
}

// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
//   1. sends funds to sha(account ++ (nonce + 1))
//   2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
//
// Carrying over the balance ensures that Ether doesn't disappear.
func (self *StateDB) CreateAccount(addr common.Address) {
	new, prev := self.createObject(addr)
	if prev != nil {
		new.setBalance(prev.data.Balance)
	}
}

func (db *StateDB) ForEachStorage(addr common.Address, cb func(key, value common.Hash) bool) {
	so := db.getStateObject(addr)
	if so == nil {
		return
	}

	// When iterating over the storage check the cache first
	for h, value := range so.cachedStorage {
		cb(h, value)
	}

	it := trie.NewIterator(so.getTrie(db.db).NodeIterator(nil))
	for it.Next() {
		// ignore cached values
		key := common.BytesToHash(db.trie.GetKey(it.Key))
		if _, ok := so.cachedStorage[key]; !ok {
			cb(key, common.BytesToHash(it.Value))
		}
	}
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (self *StateDB) Copy() *StateDB {
	self.lock.Lock()
	defer self.lock.Unlock()

	// Copy all the basic fields, initialize the memory ones
	state := &StateDB{
		db:                self.db,
		trie:              self.db.CopyTrie(self.trie),
		stateObjects:      make(map[common.Address]*stateObject, len(self.journal.dirties)),
		stateObjectsDirty: make(map[common.Address]struct{}, len(self.journal.dirties)),
		refund:            self.refund,
		logs:              make(map[common.Hash][]*types.Log, len(self.logs)),
		logSize:           self.logSize,
		preimages:         make(map[common.Hash][]byte),
		journal:           newJournal(),
	}
	// Copy the dirty states, logs, and preimages
	for addr := range self.journal.dirties {
		// As documented [here](https://github.com/worldopennetwork/go-won/pull/16485#issuecomment-380438527),
		// and in the Finalise-method, there is a case where an object is in the journal but not
		// in the stateObjects: OOG after touch on ripeMD prior to Byzantium. Thus, we need to check for
		// nil
		if object, exist := self.stateObjects[addr]; exist {
			state.stateObjects[addr] = object.deepCopy(state)
			state.stateObjectsDirty[addr] = struct{}{}
		}
	}
	// Above, we don't copy the actual journal. This means that if the copy is copied, the
	// loop above will be a no-op, since the copy's journal is empty.
	// Thus, here we iterate over stateObjects, to enable copies of copies
	for addr := range self.stateObjectsDirty {
		if _, exist := state.stateObjects[addr]; !exist {
			state.stateObjects[addr] = self.stateObjects[addr].deepCopy(state)
			state.stateObjectsDirty[addr] = struct{}{}
		}
	}

	for hash, logs := range self.logs {
		state.logs[hash] = make([]*types.Log, len(logs))
		copy(state.logs[hash], logs)
	}
	for hash, preimage := range self.preimages {
		state.preimages[hash] = preimage
	}
	return state
}

// Snapshot returns an identifier for the current revision of the state.
func (self *StateDB) Snapshot() int {
	id := self.nextRevisionId
	self.nextRevisionId++
	self.validRevisions = append(self.validRevisions, revision{id, self.journal.length()})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (self *StateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := self.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	self.journal.revert(self, snapshot)
	self.validRevisions = self.validRevisions[:idx]
}

// GetRefund returns the current value of the refund counter.
func (self *StateDB) GetRefund() uint64 {
	return self.refund
}

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
	for addr := range s.journal.dirties {
		stateObject, exist := s.stateObjects[addr]
		if !exist {
			// ripeMD is 'touched' at block 1714175, in tx 0x1237f737031e40bcde4a8b7e717b2d15e3ecadfe49bb1bbc71ee9deb09c6fcf2
			// That tx goes out of gas, and although the notion of 'touched' does not exist there, the
			// touch-event will still be recorded in the journal. Since ripeMD is a special snowflake,
			// it will persist in the journal even though the journal is reverted. In this special circumstance,
			// it may exist in `s.journal.dirties` but not in `s.stateObjects`.
			// Thus, we can safely ignore it here
			continue
		}

		if stateObject.suicided || (deleteEmptyObjects && stateObject.empty()) {
			s.deleteStateObject(stateObject)
		} else {
			stateObject.updateRoot(s.db)
			s.updateStateObject(stateObject)
		}
		s.stateObjectsDirty[addr] = struct{}{}
	}
	// Invalidate journal because reverting across transactions is not allowed.
	s.clearJournalAndRefund()
}

func (self *StateDB) SetKycLevel(addr common.Address, level uint32) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetKycLevel(level)
	}
}

func (self *StateDB) GetKycLevel(addr common.Address) uint32 {

	if self.IsContractAddress(addr) {
		//should be human
		addr = self.GetContractCreator(addr)
	}

	//check is has valid provider
	if pd := self.GetKycProvider(addr); pd == (common.Address{}) || addr == (common.Address{}) {
		return 0
	}

	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetKycLevel()
	}
	return 0
}

func (self *StateDB) SetKycZone(addr common.Address, zone uint32) {

	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetKycZone(zone)
	}
}

func (self *StateDB) GetKycZone(addr common.Address) uint32 {

	if self.IsContractAddress(addr) {
		//should be human
		addr = self.GetContractCreator(addr)
	}

	//check is has valid provider
	if pd := self.GetKycProvider(addr); pd == (common.Address{}) || addr == (common.Address{}) {
		return 0
	}

	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetKycZone()
	}

	return 0
}

func (self *StateDB) SetKycProvider(addr common.Address, provider common.Address) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetKycProvider(provider)
	}
}

func (self *StateDB) GetKycProvider(addr common.Address) common.Address {
	if self.IsContractAddress(addr) {
		addr = self.GetContractCreator(addr)
	}
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		pdr := stateObject.GetKycProvider()
		if self.KycProviderExists(pdr) {
			return pdr
		}
	}

	return common.Address{}
}

func (self *StateDB) KycProviderExists(addr common.Address) bool {

	kycNum := self.GetKycProviderCount()

	//anyone could be a provider is not provider at all.
	if kycNum == 0 {
		return true
	}

	//loop and look ,  kyc provider should be a very little number, so no worries.
	//we can add a cache here if kycNum becomes large.
	for i := int64(kycProviderStartHash); i < (kycProviderStartHash + kycNum); i++ {
		haveV := self.GetState(vm.KycContractAddress, common.BigToHash(big.NewInt(int64(i))))
		if common.BytesToAddress(haveV.Bytes()) == addr {
			return true
		}
	}

	return false
}

func (self *StateDB) GetKycProviderCount() int64 {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	haveV := stateObject.GetState(self.db, kycProviderNumberKey)
	return haveV.Big().Int64()
}

func (self *StateDB) SetKycProviderCount(c int64) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	stateObject.SetState(self.db, kycProviderNumberKey, common.BigToHash(big.NewInt(c)))
}

func (self *StateDB) AddKycProvider(addr common.Address) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	kycNum := self.GetKycProviderCount()

	//can not work more
	if kycNum >= maxKycProviderCount {
		return
	}

	stateObject.SetState(self.db, common.BigToHash(big.NewInt(int64(kycProviderStartHash+kycNum))), addr.Hash())
	self.SetKycProviderCount(kycNum + 1)
}

func (self *StateDB) RemoveKycProvider(addr common.Address) {
	kycNum := self.GetKycProviderCount()

	//anyone could be a provider is not provider at all.
	if kycNum == 0 {
		return
	}

	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	//loop and look ,  kyc provider should be a very little number, so no worries.
	//we can add a cache here if kycNum becomes large.
	found := false
	for i := int64(kycProviderStartHash); i < (kycProviderStartHash + kycNum); i++ {
		hashK := common.BigToHash(big.NewInt(int64(i)))

		if !found {
			haveV := self.GetState(vm.KycContractAddress, hashK)
			if common.BytesToAddress(haveV.Bytes()) == addr {
				//stateObject.SetState(self.db, common.BigToHash(big.NewInt(int64(kycNum-1))), common.Hash{})
				found = true
			}
		}

		if found && i < (kycProviderStartHash+kycNum-1) {
			haveNext := self.GetState(vm.KycContractAddress, common.BigToHash(big.NewInt(int64(i+1))))
			stateObject.SetState(self.db, hashK, haveNext)

		}
	}
	if found {
		self.SetKycProviderCount(kycNum - 1)
	}
}

func (self *StateDB) SetKycProviderProposol(addr common.Address, st *big.Int, pt *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	kycNum := self.GetKycProviderCount()

	self.SetState(vm.KycContractAddress, kycProposalAddressKey, addr.Hash())
	self.SetState(vm.KycContractAddress, kycProposalStartTimeKey, common.BigToHash(st))
	self.SetState(vm.KycContractAddress, kycProposalVoteTotalKey, common.BigToHash(big.NewInt(int64(kycNum))))
	self.SetState(vm.KycContractAddress, kycProposalAlreadyVotedKey, common.BigToHash(pt))

	// initial vote list
	for i := kycVoterStartHash; i < kycVoterStartHash+kycNum; i++ {
		stateObject.SetState(self.db, common.BigToHash(big.NewInt(int64(i))), common.BigToHash(common.Big0))
	}
	// initial vote result list
	for i := kycVoteResultStartHash; i < kycVoteResultStartHash+kycNum; i++ {
		stateObject.SetState(self.db, common.BigToHash(big.NewInt(int64(i))), common.BigToHash(common.Big0))
	}
}

func (self *StateDB) GetKycProviderProposol() (common.Address, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int) {

	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	hvAddr := stateObject.GetState(self.db, kycProposalAddressKey)
	hvTime := stateObject.GetState(self.db, kycProposalStartTimeKey)
	hvVoteTotal := stateObject.GetState(self.db, kycProposalVoteTotalKey)
	hvType := stateObject.GetState(self.db, kycProposalAlreadyVotedKey)
	// get number of vote yes
	iVotedYes := int64(0)
	iVotedNo := int64(0)
	yesHash := common.BigToHash(common.Big1)
	noHash := common.BigToHash(common.Big2)
	for i := kycVoteResultStartHash; i < kycVoteResultStartHash+hvVoteTotal.Big().Int64(); i++ {
		hvVoted := stateObject.GetState(self.db, common.BigToHash(big.NewInt(i)))
		if hvVoted == yesHash {
			iVotedYes++
		} else if hvVoted == noHash {
			iVotedNo++
		}
	}

	return common.BytesToAddress(hvAddr.Bytes()), hvTime.Big(), hvVoteTotal.Big(), hvType.Big(),
		big.NewInt(iVotedYes), big.NewInt(iVotedNo)

}

func (self *StateDB) SetVoteForKycProviderProposol(addr common.Address, nay uint16) bool {

	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	hvVoteTotal := stateObject.GetState(self.db, kycProposalVoteTotalKey)

	for i := int64(0); i < hvVoteTotal.Big().Int64(); i++ {
		hvVoted := stateObject.GetState(self.db, common.BigToHash(big.NewInt(kycVoterStartHash+i)))
		if hvVoted != (common.Hash{}) {
			// check if the address has been voted
			if hvVoted == addr.Hash() {
				return false
			}
			continue
		} else {
			stateObject.SetState(self.db, common.BigToHash(big.NewInt(kycVoterStartHash+i)), addr.Hash())
			if nay == 0 { // vote yes
				stateObject.SetState(self.db, common.BigToHash(big.NewInt(kycVoteResultStartHash+i)), common.BigToHash(common.Big1))
			} else { // vote no
				stateObject.SetState(self.db, common.BigToHash(big.NewInt(kycVoteResultStartHash+i)), common.BigToHash(common.Big2))
			}
			return true
		}
	}

	return false
}

func (self *StateDB) GetKycProviderList() []common.Address {
	kycNum := self.GetKycProviderCount()

	addresses := make([]common.Address, 0)
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	//loop and look ,  kyc provider should be a very little number, so no worries.
	// we can add a cache here if kycNum becomes large.
	for i := kycProviderStartHash; i < (kycProviderStartHash + kycNum); i++ {
		haveV := stateObject.GetState(self.db, common.BigToHash(big.NewInt(int64(i))))
		addresses = append(addresses, common.BytesToAddress(haveV.Bytes()))

	}

	return addresses
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	s.Finalise(deleteEmptyObjects)
	return s.trie.Hash()
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (self *StateDB) Prepare(thash, bhash common.Hash, ti int) {
	self.thash = thash
	self.bhash = bhash
	self.txIndex = ti
}

// DeleteSuicides flags the suicided objects for deletion so that it
// won't be referenced again when called / queried up on.
//
// DeleteSuicides should not be used for consensus related updates
// under any circumstances.
func (s *StateDB) DeleteSuicides() {
	// Reset refund so that any used-gas calculations can use this method.
	s.clearJournalAndRefund()

	for addr := range s.stateObjectsDirty {
		stateObject := s.stateObjects[addr]

		// If the object has been removed by a suicide
		// flag the object as deleted.
		if stateObject.suicided {
			stateObject.deleted = true
		}
		delete(s.stateObjectsDirty, addr)
	}
}

func (s *StateDB) clearJournalAndRefund() {
	s.journal = newJournal()
	s.validRevisions = s.validRevisions[:0]
	s.refund = 0
}

// Commit writes the state to the underlying in-memory trie database.
func (s *StateDB) Commit(deleteEmptyObjects bool) (root common.Hash, err error) {
	defer s.clearJournalAndRefund()

	for addr := range s.journal.dirties {
		s.stateObjectsDirty[addr] = struct{}{}
	}
	// Commit objects to the trie.
	for addr, stateObject := range s.stateObjects {
		_, isDirty := s.stateObjectsDirty[addr]
		switch {
		case stateObject.suicided || (isDirty && deleteEmptyObjects && stateObject.empty()):
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			s.deleteStateObject(stateObject)
		case isDirty:
			// Write any contract code associated with the state object
			if stateObject.code != nil && stateObject.dirtyCode {
				s.db.TrieDB().Insert(common.BytesToHash(stateObject.CodeHash()), stateObject.code)
				stateObject.dirtyCode = false
			}
			// Write any storage changes in the state object to its storage trie.
			if err := stateObject.CommitTrie(s.db); err != nil {
				return common.Hash{}, err
			}
			// Update the object in the main account trie.
			s.updateStateObject(stateObject)
		}
		delete(s.stateObjectsDirty, addr)
	}
	// Write trie changes.
	root, err = s.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var account Account
		if err := rlp.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.Root != emptyState {
			s.db.TrieDB().Reference(account.Root, parent)
		}
		code := common.BytesToHash(account.CodeHash)
		if code != emptyCode {
			s.db.TrieDB().Reference(code, parent)
		}
		return nil
	})
	log.Debug("Trie cache stats after commit", "misses", trie.CacheMisses(), "unloads", trie.CacheUnloads())
	return root, err
}

func IsPrecompiledAddress(addr common.Address) bool {

	if addr == vm.KycContractAddress {
		return true
	}

	precompiles := vm.PrecompiledContractsByzantium
	if p := precompiles[addr]; p != nil {
		return true
	}

	return false
}

func (db *StateDB) TxKycValidate(addr common.Address, dst common.Address, amount *big.Int) bool {

	if amount.Cmp(common.Big0) == 0 {
		return true
	}

	if db.GetKycProviderCount() == 0 {
		return true
	}

	if (db.KycProviderExists(addr) || IsPrecompiledAddress(addr) || db.GetKycLevel(addr) > 0) &&
		(db.KycProviderExists(dst) || IsPrecompiledAddress(dst) || db.GetKycLevel(dst) > 0 || (dst == common.Address{})) {
		return true
	}

	return false
}

func (db *StateDB) IsContractAddress(address common.Address) bool {

	return db.GetCodeSize(address) > 0
}

//store for dpos
func (self *StateDB) GetDposTotalActivatedStake() *big.Int {

	haveV := self.GetState(vm.KycContractAddress, dposTotalActivatedStakeKey)

	return haveV.Big()
}

func (self *StateDB) SetDposTotalActivatedStake(val *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	stateObject.SetState(self.db, dposTotalActivatedStakeKey, common.BigToHash(val))
}

// = common.BigToHash( big.NewInt(103))

func (self *StateDB) GetDposThreshActivatedStakeTime() *big.Int {

	haveV := self.GetState(vm.KycContractAddress, dposThreshActivatedStakeTimeKey)
	return haveV.Big()
}

func (self *StateDB) SetDposThreshActivatedStakeTime(val *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	stateObject.SetState(self.db, dposThreshActivatedStakeTimeKey, common.BigToHash(val))
}

func (self *StateDB) GetDposTotalProducerWeight() *big.Int {

	haveV := self.GetState(vm.KycContractAddress, dposTotalProducerVoteWeightKey)
	return haveV.Big()
}

func (self *StateDB) SetDposTotalProducerWeight(val *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	stateObject.SetState(self.db, dposTotalProducerVoteWeightKey, common.BigToHash(val))
}

func (self *StateDB) GetDposProducerCount() *big.Int {
	haveV := self.GetState(vm.KycContractAddress, dposProducerCountKey)
	return haveV.Big()
}

func (self *StateDB) SetDposProducerCount(val *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	stateObject.setState(dposProducerCountKey, common.BigToHash(val))
}

func (self *StateDB) RegisterProducer(pb *common.Address, url string) {
	hk := common.AddressToHashWithPrefix(pb, dposProducerURLKey)
	vb := []byte(url)
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	oldhv := stateObject.GetState(self.db, hk)

	if len(vb) > common.HashLength {
		stateObject.SetState(self.db, hk, common.BytesToHash(vb[:common.HashLength]))
		hk2 := common.AddressToHashWithPrefix(pb, dposProducerURLKeyHigh)
		stateObject.SetState(self.db, hk2, common.BytesToHash(vb[common.HashLength:]))
	} else {
		stateObject.SetState(self.db, hk, common.BytesToHash(vb))
	}

	if oldhv == common.BytesToHash([]byte{0}) {
		//new one. add to the end of list
		pbCount := self.GetDposProducerCount()

		hk := common.BigToHash(big.NewInt(pbCount.Int64() + dposProducerAllStartKey))
		hv := pb.Hash()
		stateObject.setState(hk, hv)

		pbCount = big.NewInt(pbCount.Int64() + 1)
		self.SetDposProducerCount(pbCount)

		self.UpdateProducerActive(pb, true)

	}

}

func (self *StateDB) UpdateProducerTotalVotes(pb *common.Address, stake *big.Int) {
	hk := common.AddressToHashWithPrefix(pb, dposProducerTotalVotesKey)
	hv := common.BigToHash(stake)
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	stateObject.SetState(self.db, hk, hv)
}

func (self *StateDB) UpdateProducerActive(pb *common.Address, val bool) {
	hk := common.AddressToHashWithPrefix(pb, dposProducerActiveKey)
	bv := common.Big0
	if val {
		bv = common.Big1
	}
	hv := common.BigToHash(bv)
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	stateObject.SetState(self.db, hk, hv)
}

func (self *StateDB) UpdateProducerLocation(pb *common.Address, val *big.Int) {
	hk := common.AddressToHashWithPrefix(pb, dposProducerLocationKey)
	hv := common.BigToHash(val)
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	stateObject.SetState(self.db, hk, hv)
}

func (self *StateDB) GetProducerInfo(pb *common.Address) *common.ProducerInfo {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	hk := common.AddressToHashWithPrefix(pb, dposProducerURLKey)
	hv := stateObject.GetState(self.db, hk)
	hv2 := stateObject.GetState(self.db, common.AddressToHashWithPrefix(pb, dposProducerURLKeyHigh))
	if hv != common.BytesToHash([]byte{0}) {
		ret := common.ProducerInfo{}
		cpaddr := common.BytesToAddress(pb.Bytes())
		ret.Owner = &cpaddr

		urlbytes := append(bytes.Trim(hv.Bytes(), "\x00"), bytes.Trim(hv2.Bytes(), "\x00")...)
		ret.Url = string(urlbytes)

		hk = common.AddressToHashWithPrefix(pb, dposProducerTotalVotesKey)
		hv = stateObject.GetState(self.db, hk)

		ret.TotalVotes = hv.Big()

		hk = common.AddressToHashWithPrefix(pb, dposProducerActiveKey)
		hv = stateObject.GetState(self.db, hk)

		ret.IsActive = false
		if hv != common.BytesToHash([]byte{0}) {
			ret.IsActive = true
		}

		hk = common.AddressToHashWithPrefix(pb, dposProducerLocationKey)
		hv = stateObject.GetState(self.db, hk)
		ret.Location = hv.Big()
		return &ret
	}
	return nil
}

type ProducerInfoSorter struct {
	infos []*common.ProducerInfo
}

func (s *ProducerInfoSorter) Len() int {
	return len(s.infos)
}

func (s *ProducerInfoSorter) Swap(i, j int) {
	s.infos[i], s.infos[j] = s.infos[j], s.infos[i]
}

func (s *ProducerInfoSorter) Less(i, j int) bool {
	return s.infos[i].TotalVotes.Cmp(s.infos[j].TotalVotes) > 0
}

func (self *StateDB) GetProducerTopList() []common.Address {
	addresses := make([]common.Address, 0)
	producerCount := self.GetDposProducerCount().Int64()

	//return empty if we haven't reach the DposActivatedStakeThreshold
	totalActivatedState := self.GetDposTotalActivatedStake()
	if totalActivatedState.Cmp(vm.DposActivatedStakeThreshold) < 0 {
		return addresses
	}

	isElectedDone := self.GetDposTopProducerElectedDone().Int64()

	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	if isElectedDone == 0 {

		oldproducerCount := producerCount
		//sort firstly
		pbls := self.GetProducerList(0, producerCount)

		infolist := make([]*common.ProducerInfo, 0)

		for _, pb := range pbls {
			pi := self.GetProducerInfo(&pb)
			if pi != nil && pi.IsActive {
				infolist = append(infolist, pi)
			} else {
				producerCount = producerCount - 1
			}
		}

		ssi := &ProducerInfoSorter{infos: infolist}

		sort.Sort(ssi)

		for k, pb := range ssi.infos {
			hk := common.BigToHash(big.NewInt(int64(k) + dposProducerAllStartKey))
			hv := pb.Owner.Hash()
			stateObject.setState(hk, hv)
		}

		//updated it
		if oldproducerCount != producerCount {
			self.SetDposProducerCount(big.NewInt(producerCount))

		}

		self.SetDposTopProducerElectedDone(big.NewInt(1))
	}

	for i := dposProducerAllStartKey; i < producerCount+dposProducerAllStartKey && i < 21+dposProducerAllStartKey; i++ {
		hk := common.BigToHash(big.NewInt(int64(i)))
		hv := stateObject.GetState(self.db, hk)
		if hv != common.BytesToHash([]byte{0}) {
			addresses = append(addresses, common.BytesToAddress(hv.Bytes()))
		}
	}

	return addresses

}

func (self *StateDB) GetProducerList(startPos int64, number int64) []common.Address {
	addresses := make([]common.Address, 0)

	if startPos < 0 || number <= 0 {
		return addresses
	}

	//if number > 30 {
	//	number = 30
	//}

	producerCount := self.GetDposProducerCount().Int64()
	gotlen := int64(0)

	for i := int64(startPos) + dposProducerAllStartKey; i < producerCount+dposProducerAllStartKey; i++ {
		hk := common.BigToHash(big.NewInt(int64(i)))
		hv := self.GetState(vm.KycContractAddress, hk)
		if hv != common.BytesToHash([]byte{0}) {
			pAddress := common.BytesToAddress(hv.Bytes())
			pi := self.GetProducerInfo(&pAddress)
			if pi != nil && pi.IsActive {
				addresses = append(addresses, pAddress)
				gotlen++
				if gotlen >= number {
					break
				}
			}
		}
	}

	return addresses
}

func (self *StateDB) SetVoterStaking(myAddr *common.Address, stake *big.Int) {
	hk := common.AddressToHashWithPrefix(myAddr, dposVoterStakingKey)
	hv := common.BigToHash(stake)
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	stateObject.SetState(self.db, hk, hv)
}

func (self *StateDB) GetVoterStaking(myAddr *common.Address) (stake *big.Int) {
	hk := common.AddressToHashWithPrefix(myAddr, dposVoterStakingKey)
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	hv := stateObject.GetState(self.db, hk)
	return hv.Big()
}

func (self *StateDB) SetVoterProducers(myAddr *common.Address, pbs []common.Address) {
	vcount := len(pbs)

	if vcount > 30 {
		return
	}

	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)

	hk := common.AddressToHashWithPrefix(myAddr, dposVoterCountKey)
	hv := common.BigToHash(big.NewInt(int64(vcount)))
	stateObject.SetState(self.db, hk, hv)

	for i := 0; i < vcount; i++ {
		hk = common.AddressToHashWithPrefix(myAddr, dposVoterBpAddressBeginKey+int64(i))
		hv = pbs[i].Hash()
		stateObject.SetState(self.db, hk, hv)
	}
}

func (self *StateDB) GetVoterProducers(myAddr *common.Address) (pbs []common.Address) {

	addresses := make([]common.Address, 0)

	hk := common.AddressToHashWithPrefix(myAddr, dposVoterCountKey)
	hv := self.GetState(vm.KycContractAddress, hk)
	vcount := hv.Big()

	for i := int64(0); i < vcount.Int64(); i++ {
		hk := common.AddressToHashWithPrefix(myAddr, dposVoterBpAddressBeginKey+int64(i))
		hv := self.GetState(vm.KycContractAddress, hk)
		addresses = append(addresses, common.BytesToAddress(hv.Bytes()))
	}

	return addresses
}

func (self *StateDB) SetRefundRequestInfo(myAddr *common.Address, stake *big.Int, requestTime *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	hk := common.AddressToHashWithPrefix(myAddr, dposVoterRefundAmountBeginKey)
	hv := common.BigToHash(stake)
	stateObject.SetState(self.db, hk, hv)

	hk = common.AddressToHashWithPrefix(myAddr, dposVoterRefundReqestTimeBeginKey)
	hv = common.BigToHash(requestTime)
	stateObject.SetState(self.db, hk, hv)
}

func (self *StateDB) GetRefundRequestInfo(myAddr *common.Address) (stake *big.Int, requestTime *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	hk := common.AddressToHashWithPrefix(myAddr, dposVoterRefundAmountBeginKey)
	hv := stateObject.GetState(self.db, hk)
	stake = hv.Big()

	hk = common.AddressToHashWithPrefix(myAddr, dposVoterRefundReqestTimeBeginKey)
	hv = stateObject.GetState(self.db, hk)
	requestTime = hv.Big()

	return stake, requestTime
}

func (self *StateDB) SetDposVoterLastVoteWeight(myAddr *common.Address, weight *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	hk := common.AddressToHashWithPrefix(myAddr, dposVoterLastVoteWeightKey)
	hv := common.BigToHash(weight)
	stateObject.SetState(self.db, hk, hv)
}

func (self *StateDB) GetDposVoterLastVoteWeight(myAddr *common.Address) (weight *big.Int) {
	hk := common.AddressToHashWithPrefix(myAddr, dposVoterLastVoteWeightKey)
	hv := self.GetState(vm.KycContractAddress, hk)
	return hv.Big()
}

func (self *StateDB) GetDposLastProducerScheduleUpdateTime() *big.Int {
	hv := self.GetState(vm.KycContractAddress, dposLastProducerScheduleUpdateTimeKey)
	return hv.Big()
}

func (self *StateDB) SetDposLastProducerScheduleUpdateTime(val *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	hk := dposLastProducerScheduleUpdateTimeKey
	hv := common.BigToHash(val)
	stateObject.SetState(self.db, hk, hv)
}

func (self *StateDB) GetDposTopProducerElectedDone() *big.Int {
	hv := self.GetState(vm.KycContractAddress, dposTopProducerElectedDoneKey)
	return hv.Big()
}

func (self *StateDB) SetDposTopProducerElectedDone(val *big.Int) {
	stateObject := self.GetOrNewStateObject(vm.KycContractAddress)
	hk := dposTopProducerElectedDoneKey
	hv := common.BigToHash(val)
	stateObject.SetState(self.db, hk, hv)
}

func (self *StateDB) GetContractCreator(addr common.Address) common.Address {
	if self.IsContractAddress(addr) {
		stateObject := self.getStateObject(addr)
		if stateObject != nil {
			return stateObject.GetKycProvider()
		}
	}

	return addr
}
