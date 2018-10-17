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

package state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/worldopennetwork/go-won/params"
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
	"time"

	check "gopkg.in/check.v1"

	"github.com/worldopennetwork/go-won/common"
	"github.com/worldopennetwork/go-won/core/types"
	"github.com/worldopennetwork/go-won/wondb"
)

// Tests that updating a state trie does not leak any database writes prior to
// actually committing the state.
func TestUpdateLeaks(t *testing.T) {
	// Create an empty state database
	db, _ := wondb.NewMemDatabase()
	state, _ := New(common.Hash{}, NewDatabase(db))

	// Update it with some accounts
	for i := byte(0); i < 255; i++ {
		addr := common.BytesToAddress([]byte{i})
		state.AddBalance(addr, big.NewInt(int64(11*i)))
		state.SetNonce(addr, uint64(42*i))
		if i%2 == 0 {
			state.SetState(addr, common.BytesToHash([]byte{i, i, i}), common.BytesToHash([]byte{i, i, i, i}))
		}
		if i%3 == 0 {
			state.SetCode(addr, []byte{i, i, i, i, i})
		}
		state.IntermediateRoot(false)
	}
	// Ensure that no data was leaked into the database
	for _, key := range db.Keys() {
		value, _ := db.Get(key)
		t.Errorf("State leaked into database: %x -> %x", key, value)
	}
}

// Tests that no intermediate state of an object is stored into the database,
// only the one right before the commit.
func TestIntermediateLeaks(t *testing.T) {
	// Create two state databases, one transitioning to the final state, the other final from the beginning
	transDb, _ := wondb.NewMemDatabase()
	finalDb, _ := wondb.NewMemDatabase()
	transState, _ := New(common.Hash{}, NewDatabase(transDb))
	finalState, _ := New(common.Hash{}, NewDatabase(finalDb))

	modify := func(state *StateDB, addr common.Address, i, tweak byte) {
		state.SetBalance(addr, big.NewInt(int64(11*i)+int64(tweak)))
		state.SetNonce(addr, uint64(42*i+tweak))
		if i%2 == 0 {
			state.SetState(addr, common.Hash{i, i, i, 0}, common.Hash{})
			state.SetState(addr, common.Hash{i, i, i, tweak}, common.Hash{i, i, i, i, tweak})
		}
		if i%3 == 0 {
			state.SetCode(addr, []byte{i, i, i, i, i, tweak})
		}
	}

	// Modify the transient state.
	for i := byte(0); i < 255; i++ {
		modify(transState, common.Address{byte(i)}, i, 0)
	}
	// Write modifications to trie.
	transState.IntermediateRoot(false)

	// Overwrite all the data with new values in the transient database.
	for i := byte(0); i < 255; i++ {
		modify(transState, common.Address{byte(i)}, i, 99)
		modify(finalState, common.Address{byte(i)}, i, 99)
	}

	// Commit and cross check the databases.
	if _, err := transState.Commit(false); err != nil {
		t.Fatalf("failed to commit transition state: %v", err)
	}
	if _, err := finalState.Commit(false); err != nil {
		t.Fatalf("failed to commit final state: %v", err)
	}
	for _, key := range finalDb.Keys() {
		if _, err := transDb.Get(key); err != nil {
			val, _ := finalDb.Get(key)
			t.Errorf("entry missing from the transition database: %x -> %x", key, val)
		}
	}
	for _, key := range transDb.Keys() {
		if _, err := finalDb.Get(key); err != nil {
			val, _ := transDb.Get(key)
			t.Errorf("extra entry in the transition database: %x -> %x", key, val)
		}
	}
}

// TestCopy tests that copying a statedb object indeed makes the original and
// the copy independent of each other. This test is a regression test against
// https://github.com/worldopennetwork/go-won/pull/15549.
func TestCopy(t *testing.T) {
	// Create a random state test to copy and modify "independently"
	db, _ := wondb.NewMemDatabase()
	orig, _ := New(common.Hash{}, NewDatabase(db))

	for i := byte(0); i < 255; i++ {
		obj := orig.GetOrNewStateObject(common.BytesToAddress([]byte{i}))
		obj.AddBalance(big.NewInt(int64(i)))
		orig.updateStateObject(obj)
	}
	orig.Finalise(false)

	// Copy the state, modify both in-memory
	copy := orig.Copy()

	for i := byte(0); i < 255; i++ {
		origObj := orig.GetOrNewStateObject(common.BytesToAddress([]byte{i}))
		copyObj := copy.GetOrNewStateObject(common.BytesToAddress([]byte{i}))

		origObj.AddBalance(big.NewInt(2 * int64(i)))
		copyObj.AddBalance(big.NewInt(3 * int64(i)))

		orig.updateStateObject(origObj)
		copy.updateStateObject(copyObj)
	}
	// Finalise the changes on both concurrently
	done := make(chan struct{})
	go func() {
		orig.Finalise(true)
		close(done)
	}()
	copy.Finalise(true)
	<-done

	// Verify that the two states have been updated independently
	for i := byte(0); i < 255; i++ {
		origObj := orig.GetOrNewStateObject(common.BytesToAddress([]byte{i}))
		copyObj := copy.GetOrNewStateObject(common.BytesToAddress([]byte{i}))

		if want := big.NewInt(3 * int64(i)); origObj.Balance().Cmp(want) != 0 {
			t.Errorf("orig obj %d: balance mismatch: have %v, want %v", i, origObj.Balance(), want)
		}
		if want := big.NewInt(4 * int64(i)); copyObj.Balance().Cmp(want) != 0 {
			t.Errorf("copy obj %d: balance mismatch: have %v, want %v", i, copyObj.Balance(), want)
		}
	}
}

func TestSnapshotRandom(t *testing.T) {
	config := &quick.Config{MaxCount: 1000}
	err := quick.Check((*snapshotTest).run, config)
	if cerr, ok := err.(*quick.CheckError); ok {
		test := cerr.In[0].(*snapshotTest)
		t.Errorf("%v:\n%s", test.err, test)
	} else if err != nil {
		t.Error(err)
	}
}

// A snapshotTest checks that reverting StateDB snapshots properly undoes all changes
// captured by the snapshot. Instances of this test with pseudorandom content are created
// by Generate.
//
// The test works as follows:
//
// A new state is created and all actions are applied to it. Several snapshots are taken
// in between actions. The test then reverts each snapshot. For each snapshot the actions
// leading up to it are replayed on a fresh, empty state. The behaviour of all public
// accessor methods on the reverted state must match the return value of the equivalent
// methods on the replayed state.
type snapshotTest struct {
	addrs     []common.Address // all account addresses
	actions   []testAction     // modifications to the state
	snapshots []int            // actions indexes at which snapshot is taken
	err       error            // failure details are reported through this field
}

type testAction struct {
	name   string
	fn     func(testAction, *StateDB)
	args   []int64
	noAddr bool
}

// newTestAction creates a random action that changes state.
func newTestAction(addr common.Address, r *rand.Rand) testAction {
	actions := []testAction{
		{
			name: "SetBalance",
			fn: func(a testAction, s *StateDB) {
				s.SetBalance(addr, big.NewInt(a.args[0]))
			},
			args: make([]int64, 1),
		},
		{
			name: "AddBalance",
			fn: func(a testAction, s *StateDB) {
				s.AddBalance(addr, big.NewInt(a.args[0]))
			},
			args: make([]int64, 1),
		},
		{
			name: "SetNonce",
			fn: func(a testAction, s *StateDB) {
				s.SetNonce(addr, uint64(a.args[0]))
			},
			args: make([]int64, 1),
		},
		{
			name: "SetState",
			fn: func(a testAction, s *StateDB) {
				var key, val common.Hash
				binary.BigEndian.PutUint16(key[:], uint16(a.args[0]))
				binary.BigEndian.PutUint16(val[:], uint16(a.args[1]))
				s.SetState(addr, key, val)
			},
			args: make([]int64, 2),
		},
		{
			name: "SetCode",
			fn: func(a testAction, s *StateDB) {
				code := make([]byte, 16)
				binary.BigEndian.PutUint64(code, uint64(a.args[0]))
				binary.BigEndian.PutUint64(code[8:], uint64(a.args[1]))
				s.SetCode(addr, code)
			},
			args: make([]int64, 2),
		},
		{
			name: "CreateAccount",
			fn: func(a testAction, s *StateDB) {
				s.CreateAccount(addr)
			},
		},
		{
			name: "Suicide",
			fn: func(a testAction, s *StateDB) {
				s.Suicide(addr)
			},
		},
		{
			name: "AddRefund",
			fn: func(a testAction, s *StateDB) {
				s.AddRefund(uint64(a.args[0]))
			},
			args:   make([]int64, 1),
			noAddr: true,
		},
		{
			name: "AddLog",
			fn: func(a testAction, s *StateDB) {
				data := make([]byte, 2)
				binary.BigEndian.PutUint16(data, uint16(a.args[0]))
				s.AddLog(&types.Log{Address: addr, Data: data})
			},
			args: make([]int64, 1),
		},
	}
	action := actions[r.Intn(len(actions))]
	var nameargs []string
	if !action.noAddr {
		nameargs = append(nameargs, addr.Hex())
	}
	for _, i := range action.args {
		action.args[i] = rand.Int63n(100)
		nameargs = append(nameargs, fmt.Sprint(action.args[i]))
	}
	action.name += strings.Join(nameargs, ", ")
	return action
}

// Generate returns a new snapshot test of the given size. All randomness is
// derived from r.
func (*snapshotTest) Generate(r *rand.Rand, size int) reflect.Value {
	// Generate random actions.
	addrs := make([]common.Address, 50)
	for i := range addrs {
		addrs[i][0] = byte(i)
	}
	actions := make([]testAction, size)
	for i := range actions {
		addr := addrs[r.Intn(len(addrs))]
		actions[i] = newTestAction(addr, r)
	}
	// Generate snapshot indexes.
	nsnapshots := int(math.Sqrt(float64(size)))
	if size > 0 && nsnapshots == 0 {
		nsnapshots = 1
	}
	snapshots := make([]int, nsnapshots)
	snaplen := len(actions) / nsnapshots
	for i := range snapshots {
		// Try to place the snapshots some number of actions apart from each other.
		snapshots[i] = (i * snaplen) + r.Intn(snaplen)
	}
	return reflect.ValueOf(&snapshotTest{addrs, actions, snapshots, nil})
}

func (test *snapshotTest) String() string {
	out := new(bytes.Buffer)
	sindex := 0
	for i, action := range test.actions {
		if len(test.snapshots) > sindex && i == test.snapshots[sindex] {
			fmt.Fprintf(out, "---- snapshot %d ----\n", sindex)
			sindex++
		}
		fmt.Fprintf(out, "%4d: %s\n", i, action.name)
	}
	return out.String()
}

func (test *snapshotTest) run() bool {
	// Run all actions and create snapshots.
	var (
		db, _        = wondb.NewMemDatabase()
		state, _     = New(common.Hash{}, NewDatabase(db))
		snapshotRevs = make([]int, len(test.snapshots))
		sindex       = 0
	)
	for i, action := range test.actions {
		if len(test.snapshots) > sindex && i == test.snapshots[sindex] {
			snapshotRevs[sindex] = state.Snapshot()
			sindex++
		}
		action.fn(action, state)
	}
	// Revert all snapshots in reverse order. Each revert must yield a state
	// that is equivalent to fresh state with all actions up the snapshot applied.
	for sindex--; sindex >= 0; sindex-- {
		checkstate, _ := New(common.Hash{}, state.Database())
		for _, action := range test.actions[:test.snapshots[sindex]] {
			action.fn(action, checkstate)
		}
		state.RevertToSnapshot(snapshotRevs[sindex])
		if err := test.checkEqual(state, checkstate); err != nil {
			test.err = fmt.Errorf("state mismatch after revert to snapshot %d\n%v", sindex, err)
			return false
		}
	}
	return true
}

// checkEqual checks that methods of state and checkstate return the same values.
func (test *snapshotTest) checkEqual(state, checkstate *StateDB) error {
	for _, addr := range test.addrs {
		var err error
		checkeq := func(op string, a, b interface{}) bool {
			if err == nil && !reflect.DeepEqual(a, b) {
				err = fmt.Errorf("got %s(%s) == %v, want %v", op, addr.Hex(), a, b)
				return false
			}
			return true
		}
		// Check basic accessor methods.
		checkeq("Exist", state.Exist(addr), checkstate.Exist(addr))
		checkeq("HasSuicided", state.HasSuicided(addr), checkstate.HasSuicided(addr))
		checkeq("GetBalance", state.GetBalance(addr), checkstate.GetBalance(addr))
		checkeq("GetNonce", state.GetNonce(addr), checkstate.GetNonce(addr))
		checkeq("GetCode", state.GetCode(addr), checkstate.GetCode(addr))
		checkeq("GetCodeHash", state.GetCodeHash(addr), checkstate.GetCodeHash(addr))
		checkeq("GetCodeSize", state.GetCodeSize(addr), checkstate.GetCodeSize(addr))
		// Check storage.
		if obj := state.getStateObject(addr); obj != nil {
			state.ForEachStorage(addr, func(key, val common.Hash) bool {
				return checkeq("GetState("+key.Hex()+")", val, checkstate.GetState(addr, key))
			})
			checkstate.ForEachStorage(addr, func(key, checkval common.Hash) bool {
				return checkeq("GetState("+key.Hex()+")", state.GetState(addr, key), checkval)
			})
		}
		if err != nil {
			return err
		}
	}

	if state.GetRefund() != checkstate.GetRefund() {
		return fmt.Errorf("got GetRefund() == %d, want GetRefund() == %d",
			state.GetRefund(), checkstate.GetRefund())
	}
	if !reflect.DeepEqual(state.GetLogs(common.Hash{}), checkstate.GetLogs(common.Hash{})) {
		return fmt.Errorf("got GetLogs(common.Hash{}) == %v, want GetLogs(common.Hash{}) == %v",
			state.GetLogs(common.Hash{}), checkstate.GetLogs(common.Hash{}))
	}
	return nil
}

func (s *StateSuite) TestTouchDelete(c *check.C) {
	s.state.GetOrNewStateObject(common.Address{})
	root, _ := s.state.Commit(false)
	s.state.Reset(root)

	snapshot := s.state.Snapshot()
	s.state.AddBalance(common.Address{}, new(big.Int))

	if len(s.state.journal.dirties) != 1 {
		c.Fatal("expected one dirty state object")
	}
	s.state.RevertToSnapshot(snapshot)
	if len(s.state.journal.dirties) != 0 {
		c.Fatal("expected no dirty state object")
	}
}

// TestCopyOfCopy tests that modified objects are carried over to the copy, and the copy of the copy.
// See https://github.com/worldopennetwork/go-won/pull/15225#issuecomment-380191512
func TestCopyOfCopy(t *testing.T) {
	db, _ := wondb.NewMemDatabase()
	sdb, _ := New(common.Hash{}, NewDatabase(db))
	addr := common.HexToAddress("aaaa")
	sdb.SetBalance(addr, big.NewInt(42))

	if got := sdb.Copy().GetBalance(addr).Uint64(); got != 42 {
		t.Fatalf("1st copy fail, expected 42, got %v", got)
	}
	if got := sdb.Copy().Copy().GetBalance(addr).Uint64(); got != 42 {
		t.Fatalf("2nd copy fail, expected 42, got %v", got)
	}
}

func TestKycInfo(t *testing.T) {
	transDb, _ := wondb.NewMemDatabase()
	transState, _ := New(common.Hash{}, NewDatabase(transDb))

	addr := common.HexToAddress("1FF")
	transState.SetKycLevel(addr, 32)
	transState.SetKycZone(addr, 86)
	transState.SetKycProvider(addr, common.BytesToAddress([]byte{101}))
	transState.SetKycProviderCount(5)

	// Write modifications to trie.
	transState.IntermediateRoot(false)

	// Commit to databases.
	if _, err := transState.Commit(false); err != nil {
		t.Fatalf("failed to commit transition state: %v", err)
	}

	checkEq := func(op string, a, b interface{}) bool {
		if !reflect.DeepEqual(a, b) {
			t.Errorf("got %s(%s) == %v, want %v", op, addr.Hex(), a, b)
			return false
		}
		return true
	}
	checkEq("KycLevel", transState.GetKycLevel(addr), uint32(32))
	checkEq("KycZone", transState.GetKycZone(addr), uint32(86))
	checkEq("KycProvider", transState.GetKycProvider(addr), common.BytesToAddress([]byte{101}))
	checkEq("KycProviderCount", transState.GetKycProviderCount(), int64(5))
}

func TestKycProposal(t *testing.T) {
	transDb, _ := wondb.NewMemDatabase()
	transState, _ := New(common.Hash{}, NewDatabase(transDb))
	addr1 := common.HexToAddress("1FF1")

	transState.AddKycProvider(addr1)
	transState.AddKycProvider(common.HexToAddress("2ff1"))
	addrList := transState.GetKycProviderList()
	t.Logf("After add got provider %v", addrList)
	transState.RemoveKycProvider(common.HexToAddress("2ff1"))
	addrList = transState.GetKycProviderList()
	t.Logf("After remove got provider %v", addrList)
	transState.RemoveKycProvider(common.HexToAddress("2ff4"))
	addrList = transState.GetKycProviderList()
	t.Logf("After remove got provider %v", addrList)

	addr2 := common.HexToAddress("1FF2")

	// vote for additional provider
	transState.SetKycProviderProposol(addr2, big.NewInt(time.Now().Unix()), big.NewInt(1))

	candidateAddr, startTime, votesTotal, proposalType, votesYes, votesNo := transState.GetKycProviderProposol()
	t.Logf("The type %d proposal info is: candidate address=%s, start time=%v, total votes=%d, yes=%d, no=%d.",
		proposalType, candidateAddr.String(), startTime, votesTotal, votesYes, votesNo)

	transState.SetVoteForKycProviderProposol(addr1, 0)
	_, _, _, _, votesYes, votesNo = transState.GetKycProviderProposol()
	t.Logf("The type %d proposal info is: candidate address=%s, start time=%v, total votes=%d, yes=%d, no=%d.",
		proposalType, candidateAddr.String(), startTime, votesTotal, votesYes, votesNo)

	if votesYes.Uint64() > votesTotal.Uint64()/2 {
		transState.AddKycProvider(candidateAddr)
	}

	addrList = transState.GetKycProviderList()
	t.Logf("After vote got provider %v", addrList)

	// vote for removal provider
	transState.SetKycProviderProposol(addr2, big.NewInt(time.Now().Unix()), big.NewInt(2))

	transState.SetVoteForKycProviderProposol(addr1, 0)
	transState.SetVoteForKycProviderProposol(addr2, 0)
	candidateAddr, startTime, votesTotal, proposalType, votesYes, votesNo = transState.GetKycProviderProposol()
	t.Logf("The type %d proposal info is: candidate address=%s, start time=%v, total votes=%d, yes=%d, no=%d.",
		proposalType, candidateAddr.String(), startTime, votesTotal, votesYes, votesNo)

	if votesYes.Uint64() > votesTotal.Uint64()/2 {
		transState.RemoveKycProvider(candidateAddr)
	}

	addrList = transState.GetKycProviderList()
	t.Logf("After vote got provider %v", addrList)

	// Write modifications to trie.
	transState.IntermediateRoot(false)
}

func TestDposInfoRw(t *testing.T) {
	db, _ := wondb.NewMemDatabase()
	state, _ := New(common.Hash{}, NewDatabase(db))

	state.SetDposTotalActivatedStake(big.NewInt(100))
	state.SetDposThreshActivatedStakeTime(big.NewInt(1536654737))
	state.SetDposTotalProducerWeight(big.NewInt(30))
	state.SetDposProducerCount(big.NewInt(50))
	state.SetVoterStaking(&addr, big.NewInt(10000))
	state.SetDposVoterLastVoteWeight(&addr, big.NewInt(90))
	state.SetDposLastProducerScheduleUpdateTime(big.NewInt(1536654868))
	state.SetDposTopProducerElectedDone(big.NewInt(1))

	// Write modifications to trie.
	state.IntermediateRoot(false)

	checkEq := func(op string, a, b interface{}) bool {
		if !reflect.DeepEqual(a, b) {
			t.Errorf("got %s(%s) == %v, want %v", op, addr.Hex(), a, b)
			return false
		}
		return true
	}
	checkEq("TotalActivatedStake", state.GetDposTotalActivatedStake(), big.NewInt(100))
	checkEq("ThreshActivatedStakeTime", state.GetDposThreshActivatedStakeTime(), big.NewInt(1536654737))
	checkEq("TotalProducerWeight", state.GetDposTotalProducerWeight(), big.NewInt(30))
	checkEq("ProducerCount", state.GetDposProducerCount(), big.NewInt(50))
	checkEq("VoterStaking", state.GetVoterStaking(&addr), big.NewInt(10000))
	checkEq("VoterLastVoteWeight", state.GetDposVoterLastVoteWeight(&addr), big.NewInt(90))
	checkEq("LastProducerScheduleUpdateTime", state.GetDposLastProducerScheduleUpdateTime(), big.NewInt(1536654868))
	checkEq("TopProducerElectedDone", state.GetDposTopProducerElectedDone(), big.NewInt(1))
}

func TestDposProducerInfo(t *testing.T) {
	db, _ := wondb.NewMemDatabase()
	state, _ := New(common.Hash{}, NewDatabase(db))

	t.Logf("The current producer number = %d", state.GetDposProducerCount())
	state.RegisterProducer(&addr, "https://127.0.0.1:808")
	state.RegisterProducer(&addr, "https://node111.worldopennetwork.net:2808")
	t.Logf("The current producer number = %d", state.GetDposProducerCount())

	t.Logf("The producer info is: %v", state.GetProducerInfo(&addr))

	state.UpdateProducerTotalVotes(&addr, big.NewInt(10))
	state.UpdateProducerActive(&addr, false)
	state.UpdateProducerLocation(&addr, big.NewInt(658685))

	t.Logf("The producer info is: %v", state.GetProducerInfo(&addr))
}

func TestDposProducerList(t *testing.T) {
	db, _ := wondb.NewMemDatabase()
	state, _ := New(common.Hash{}, NewDatabase(db))

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 60; i++ {
		addr := common.BigToAddress(big.NewInt(rand.Int63n(999999)))
		state.RegisterProducer(&addr, "https://node.woncoin.net:"+strconv.Itoa(rand.Intn(65535)))
	}
	count := state.GetDposProducerCount()
	t.Logf("The current producer number = %d", count)

	prList := state.GetProducerList(0, count.Int64())
	var votesList []int
	for i := 0; i < len(prList); i++ {
		votes := rand.Intn(1000)
		state.UpdateProducerTotalVotes(&prList[i], big.NewInt(int64(votes)))
		votesList = append(votesList, votes)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(votesList)))
	t.Logf("Votes rank is : %v", votesList[:21])

	state.SetDposTotalActivatedStake(big.NewInt(0).Mul(big.NewInt(25000000), big.NewInt(params.WON)))
	topList := state.GetProducerTopList()
	for _, val := range topList {
		t.Logf("The producer info is: %v", state.GetProducerInfo(&val))
	}

}

func TestRefundRequestInfo(t *testing.T) {
	db, _ := wondb.NewMemDatabase()
	state, _ := New(common.Hash{}, NewDatabase(db))

	state.SetRefundRequestInfo(&addr, big.NewInt(399), big.NewInt(time.Now().Unix()))

	checkEq := func(op string, a, b interface{}) bool {
		if !reflect.DeepEqual(a, b) {
			t.Errorf("got %s(%s) == %v, want %v", op, addr.Hex(), a, b)
			return false
		}
		return true
	}

	stake, reqTime := state.GetRefundRequestInfo(&addr)
	checkEq("RefundStake", stake, big.NewInt(399))
	checkEq("RefundTime", reqTime, big.NewInt(time.Now().Unix()))
}
