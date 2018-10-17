// Copyright 2015 The go-ethereum Authors
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

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main WorldOpenNetwork network.
var MainnetBootnodes = []string{
	// WorldOpenNetwork Foundation Go Bootnodes

	//"enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303", // IE
	//"enode://3f1d12044546b76342d59d4a05532c14b85aa669704bfe1f864fe079415aa2c02d743e03218e57a33fb94523adb54032871a6c51b2cc5514cb7c7e35b3ed0a99@13.93.211.84:30303",  // US-WEST
	//"enode://78de8a0916848093c73790ead81d1928bec737d565119932b98c6b100d944b7a95e94f847f689fc723399d2e31129d182f7ef3863f2b4c820abbf3ab2722344d@191.235.84.50:30303", // BR
	//"enode://158f8aab45f6d19c6cbf4a089c2670541a8da11978a2f90dbf6a502a4a3bab80d288afdbeb7ec0ef6d92de563767f3b1ea9e8e334ca711e9f8e2df5a0385e8e6@13.75.154.138:30303", // AU
	//"enode://1118980bf48b0a3640bdba04e0fe78b1add18e1cd99bf22d53daac1fd9972ad650df52176e7c7d89d1114cfef2bc23a2959aa54998a46afcf7d91809f0855082@52.74.57.123:30303",  // SG

	// WorldOpenNetwork Foundation C++ Bootnodes
	//"enode://979b7fa28feeb35a4741660a16076f1943202cb72b6af70d327f053e248bab9ba81760f39d0701ef1d8f89cc1fbd2cacba0710a12cd5314d5e0c9021aa3637f9@5.1.83.226:30303", // DE

	"enode://ad336b0e0878c66a368beea8361ca74b5382548c10d1e205e278f72d7cdd5005963f8eb9d7ef234f7bd636516523e6031ab478fa9287926720a84f7b3f13f6c3@104.196.238.56:30331",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// won alpha test network.
var TestnetBootnodes = []string{
	"enode://0f711f95ff508e9c8f869cb08c815309dd4f3e71897fefb4986d53185bbdb1b912d4dae44860c2205c791fa5a31c8f9e7c41f73e17a420f1088ee9335aa6fdcc@216.66.17.57:30305", // alpha
	"enode://303ca44bbc92c6cbd370822b8050c8c6cf656b9dc211c0594189fbb9975125732e06dc66b2a6da131ea8e6f1bc5e7a17eef22d899f6586d81d18b74a868c1d9d@192.168.1.41:30304", // Tsingtao intranet
}

// BetanetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Betanet test network.
var BetanetBootnodes = []string{
	"enode://3434564bbc92c6cbd376788b8050c8c6cf656b9dc211c0594189fbb9975125732e06dc66b2a6da131ea8e6f1bc5e7a17eef22d899f6586d81d18b74a868c1dea@192.168.1.41:30304", // Tsingtao
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{}
