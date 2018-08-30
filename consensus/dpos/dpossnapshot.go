// Copyright 2018 The go-won Authors
// This file is part of the go-ethereum library.
//
// The go-won library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-won library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-won library. If not, see <http://www.gnu.org/licenses/>.

package dpos

import (
	"encoding/json"

	"bytes"
	"github.com/worldopennet/go-won/common"
	"github.com/worldopennet/go-won/wondb"
	"github.com/worldopennet/go-won/params"
	lru "github.com/hashicorp/golang-lru"
)

// Vote represents a single vote that an authorized signer made to modify the
// list of authorizations.
//type Vote struct {
//	Signer    common.Address `json:"signer"`    // Authorized signer that cast this vote
//	Block     uint64         `json:"block"`     // Block number the vote was cast in (expire old votes)
//	Address   common.Address `json:"address"`   // Account being voted on to change its authorization
//	Authorize bool           `json:"authorize"` // Whwon to authorize or deauthorize the voted account
//}
//
//// Tally is a simple vote tally to keep the current score of votes. Votes that
//// go against the proposal aren't counted since it's equivalent to not voting.
//type Tally struct {
//	Authorize bool `json:"authorize"` // Whwon the vote is about authorizing or kicking someone
//	Votes     int  `json:"votes"`     // Number of votes until now wanting to pass the proposal
//}

// Snapshot is the state of the authorization voting at a given point in time.
type DposSnapshot struct {
	config   *params.DposConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache      // Cache of recent block signatures to speed up ecrecover

	Number  uint64                      `json:"number"`  // Block number where the snapshot was created
	Hash    common.Hash                 `json:"hash"`    // Block hash where the snapshot was created
	Signers map[common.Address]struct{} `json:"signers"` // Set of authorized signers at this moment
	Recents map[uint64]common.Address   `json:"recents"` // Set of recent signers for spam protections

}

// newSnapshot creates a new snapshot with the specified startup parameters. This
// method does not initialize the set of recent signers, so only ever use if for
// the genesis block.
func newSnapshot(config *params.DposConfig, sigcache *lru.ARCCache, number uint64, hash common.Hash, signers []common.Address) *DposSnapshot {
	snap := &DposSnapshot{
		config:   config,
		sigcache: sigcache,
		Number:   number,
		Hash:     hash,
		Signers:  make(map[common.Address]struct{}),
		Recents:  make(map[uint64]common.Address),
		//Tally:    make(map[common.Address]Tally),
	}
	for _, signer := range signers {
		snap.Signers[signer] = struct{}{}
	}


	return snap
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(config *params.DposConfig, sigcache *lru.ARCCache, db wondb.Database, hash common.Hash) (*DposSnapshot, error) {
	blob, err := db.Get(append([]byte("dpos-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(DposSnapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.config = config
	snap.sigcache = sigcache
	return snap, nil
}

// store inserts the snapshot into the database.
func (s *DposSnapshot) store(db wondb.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("dpos-"), s.Hash[:]...), blob)
}

// copy creates a deep copy of the snapshot, though not the individual votes.
func (s *DposSnapshot) copy() *DposSnapshot {
	cpy := &DposSnapshot{
		config:   s.config,
		sigcache: s.sigcache,
		Number:   s.Number,
		Hash:     s.Hash,
		Signers:  make(map[common.Address]struct{}),
		Recents:  make(map[uint64]common.Address),
		//Votes:    make([]*Vote, len(s.Votes)),
		//Tally:    make(map[common.Address]Tally),
	}
	for signer := range s.Signers {
		cpy.Signers[signer] = struct{}{}
	}
	for block, signer := range s.Recents {
		cpy.Recents[block] = signer
	}

	//for address, tally := range s.Tally {
	//	cpy.Tally[address] = tally
	//}
	//copy(cpy.Votes, s.Votes)

	return cpy
}

// signers retrieves the list of authorized signers in ascending order.
func (s *DposSnapshot) signers() []common.Address {
	signers := make([]common.Address, 0, len(s.Signers))
	for signer := range s.Signers {
		signers = append(signers, signer)
	}

	for i := 0; i < len(signers); i++ {
		for j := i + 1; j < len(signers); j++ {
			if bytes.Compare(signers[i][:], signers[j][:]) > 0 {
				signers[i], signers[j] = signers[j], signers[i]
			}
		}
	}

	return signers
}

//// inturn returns if a signer at a given block height is in-turn or not.
func (s *DposSnapshot) inturn(number uint64, signer common.Address) bool {
	signers, offset := s.signers(), 0
	for offset < len(signers) && signers[offset] != signer {
		offset++
	}
	return (number % uint64(len(signers))) == uint64(offset)
}
