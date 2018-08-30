var tokenAbi = [{
    "constant": true,
    "inputs": [],
    "name": "name",
    "outputs": [{"name": "", "type": "bytes32"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
}, {
    "constant": false,
    "inputs": [],
    "name": "stop",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "guy", "type": "address"}, {"name": "wad", "type": "uint256"}],
    "name": "approve",
    "outputs": [{"name": "", "type": "bool"}],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "owner_", "type": "address"}],
    "name": "setOwner",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": true,
    "inputs": [],
    "name": "totalSupply",
    "outputs": [{"name": "", "type": "uint256"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "src", "type": "address"}, {"name": "dst", "type": "address"}, {
        "name": "wad",
        "type": "uint256"
    }],
    "name": "transferFrom",
    "outputs": [{"name": "", "type": "bool"}],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": true,
    "inputs": [],
    "name": "decimals",
    "outputs": [{"name": "", "type": "uint256"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "dst", "type": "address"}, {"name": "wad", "type": "uint128"}],
    "name": "push",
    "outputs": [{"name": "", "type": "bool"}],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "name_", "type": "bytes32"}],
    "name": "setName",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "wad", "type": "uint128"}],
    "name": "mint",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": true,
    "inputs": [{"name": "src", "type": "address"}],
    "name": "balanceOf",
    "outputs": [{"name": "", "type": "uint256"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
}, {
    "constant": true,
    "inputs": [],
    "name": "stopped",
    "outputs": [{"name": "", "type": "bool"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "authority_", "type": "address"}],
    "name": "setAuthority",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "src", "type": "address"}, {"name": "wad", "type": "uint128"}],
    "name": "pull",
    "outputs": [{"name": "", "type": "bool"}],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": true,
    "inputs": [],
    "name": "owner",
    "outputs": [{"name": "", "type": "address"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "wad", "type": "uint128"}],
    "name": "burn",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": true,
    "inputs": [],
    "name": "symbol",
    "outputs": [{"name": "", "type": "bytes32"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
}, {
    "constant": false,
    "inputs": [{"name": "dst", "type": "address"}, {"name": "wad", "type": "uint256"}],
    "name": "transfer",
    "outputs": [{"name": "", "type": "bool"}],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": false,
    "inputs": [],
    "name": "start",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
}, {
    "constant": true,
    "inputs": [],
    "name": "authority",
    "outputs": [{"name": "", "type": "address"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
}, {
    "constant": true,
    "inputs": [{"name": "src", "type": "address"}, {"name": "guy", "type": "address"}],
    "name": "allowance",
    "outputs": [{"name": "", "type": "uint256"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
}, {
    "inputs": [{"name": "symbol_", "type": "bytes32"}],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "constructor"
}, {
    "anonymous": true,
    "inputs": [{"indexed": true, "name": "sig", "type": "bytes4"}, {
        "indexed": true,
        "name": "guy",
        "type": "address"
    }, {"indexed": true, "name": "foo", "type": "bytes32"}, {
        "indexed": true,
        "name": "bar",
        "type": "bytes32"
    }, {"indexed": false, "name": "wad", "type": "uint256"}, {"indexed": false, "name": "fax", "type": "bytes"}],
    "name": "LogNote",
    "type": "event"
}, {
    "anonymous": false,
    "inputs": [{"indexed": true, "name": "authority", "type": "address"}],
    "name": "LogSetAuthority",
    "type": "event"
}, {
    "anonymous": false,
    "inputs": [{"indexed": true, "name": "owner", "type": "address"}],
    "name": "LogSetOwner",
    "type": "event"
}, {
    "anonymous": false,
    "inputs": [{"indexed": true, "name": "from", "type": "address"}, {
        "indexed": true,
        "name": "to",
        "type": "address"
    }, {"indexed": false, "name": "value", "type": "uint256"}],
    "name": "Transfer",
    "type": "event"
}, {
    "anonymous": false,
    "inputs": [{"indexed": true, "name": "owner", "type": "address"}, {
        "indexed": true,
        "name": "spender",
        "type": "address"
    }, {"indexed": false, "name": "value", "type": "uint256"}],
    "name": "Approval",
    "type": "event"
}];

var tokenByteCode = "0x606060405260126006556000600790600019169055341561001f57600080fd5b604051602080611cde83398101806040528101908080519060200190929190505050600080600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550806000819055505033600460006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055503373ffffffffffffffffffffffffffffffffffffffff167fce241d7ca1f669fee44b6fc00b8eba2df3bb514eed0f6f668f8f89096e81ed9460405160405180910390a2806005816000191690555050611bb08061012e6000396000f30060606040526004361061011d576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806306fdde031461012257806307da68f514610153578063095ea7b31461016857806313af4035146101cb57806318160ddd1461020c57806323b872dd14610235578063313ce567146102b85780633452f51d146102e15780635ac801fe1461035657806369d3e20e1461038557806370a08231146103c257806375f12b21146104175780637a9e5e4b146104445780638402181f146104855780638da5cb5b146104fa57806390bc16931461054f57806395d89b411461058c578063a9059cbb146105bd578063be9a655514610620578063bf7e214f14610635578063dd62ed3e1461068a575b600080fd5b341561012d57600080fd5b6101356106ff565b60405180826000191660001916815260200191505060405180910390f35b341561015e57600080fd5b610166610705565b005b341561017357600080fd5b6101b1600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610805565b604051808215151515815260200191505060405180910390f35b34156101d657600080fd5b61020a600480360381019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506108df565b005b341561021757600080fd5b61021f6109be565b6040518082815260200191505060405180910390f35b341561024057600080fd5b61029e600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506109c7565b604051808215151515815260200191505060405180910390f35b34156102c357600080fd5b6102cb610aa3565b6040518082815260200191505060405180910390f35b34156102ec57600080fd5b61033c600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080356fffffffffffffffffffffffffffffffff169060200190929190505050610aa9565b604051808215151515815260200191505060405180910390f35b341561036157600080fd5b6103836004803603810190808035600019169060200190929190505050610acf565b005b341561039057600080fd5b6103c060048036038101908080356fffffffffffffffffffffffffffffffff169060200190929190505050610b13565b005b34156103cd57600080fd5b610401600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610cd4565b6040518082815260200191505060405180910390f35b341561042257600080fd5b61042a610d1d565b604051808215151515815260200191505060405180910390f35b341561044f57600080fd5b610483600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610d30565b005b341561049057600080fd5b6104e0600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080356fffffffffffffffffffffffffffffffff169060200190929190505050610e0f565b604051808215151515815260200191505060405180910390f35b341561050557600080fd5b61050d610e36565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b341561055a57600080fd5b61058a60048036038101908080356fffffffffffffffffffffffffffffffff169060200190929190505050610e5c565b005b341561059757600080fd5b61059f61101d565b60405180826000191660001916815260200191505060405180910390f35b34156105c857600080fd5b610606600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050611023565b604051808215151515815260200191505060405180910390f35b341561062b57600080fd5b6106336110fd565b005b341561064057600080fd5b6106486111fd565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b341561069557600080fd5b6106e9600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611223565b6040518082815260200191505060405180910390f35b60075481565b61073b610736336000357fffffffff00000000000000000000000000000000000000000000000000000000166112aa565b61151a565b60008060043591506024359050806000191682600019163373ffffffffffffffffffffffffffffffffffffffff166000357fffffffff00000000000000000000000000000000000000000000000000000000167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19163460003660405180848152602001806020018281038252848482818152602001925080828437820191505094505050505060405180910390a46001600460146101000a81548160ff0219169083151502179055505050565b6000610820600460149054906101000a900460ff161561151a565b60008060043591506024359050806000191682600019163373ffffffffffffffffffffffffffffffffffffffff166000357fffffffff00000000000000000000000000000000000000000000000000000000167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19163460003660405180848152602001806020018281038252848482818152602001925080828437820191505094505050505060405180910390a46108d58585611529565b9250505092915050565b610915610910336000357fffffffff00000000000000000000000000000000000000000000000000000000166112aa565b61151a565b80600460006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167fce241d7ca1f669fee44b6fc00b8eba2df3bb514eed0f6f668f8f89096e81ed9460405160405180910390a250565b60008054905090565b60006109e2600460149054906101000a900460ff161561151a565b60008060043591506024359050806000191682600019163373ffffffffffffffffffffffffffffffffffffffff166000357fffffffff00000000000000000000000000000000000000000000000000000000167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19163460003660405180848152602001806020018281038252848482818152602001925080828437820191505094505050505060405180910390a4610a9886868661161b565b925050509392505050565b60065481565b6000610ac783836fffffffffffffffffffffffffffffffff16611023565b905092915050565b610b05610b00336000357fffffffff00000000000000000000000000000000000000000000000000000000166112aa565b61151a565b806007816000191690555050565b610b49610b44336000357fffffffff00000000000000000000000000000000000000000000000000000000166112aa565b61151a565b610b62600460149054906101000a900460ff161561151a565b60008060043591506024359050806000191682600019163373ffffffffffffffffffffffffffffffffffffffff166000357fffffffff00000000000000000000000000000000000000000000000000000000167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19163460003660405180848152602001806020018281038252848482818152602001925080828437820191505094505050505060405180910390a4610c68600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054846fffffffffffffffffffffffffffffffff1661197e565b600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550610cc9600054846fffffffffffffffffffffffffffffffff1661197e565b600081905550505050565b6000600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b600460149054906101000a900460ff1681565b610d66610d61336000357fffffffff00000000000000000000000000000000000000000000000000000000166112aa565b61151a565b80600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f1abebea81bfa2637f28358c371278fb15ede7ea8dd28d2e03b112ff6d936ada460405160405180910390a250565b6000610e2e8333846fffffffffffffffffffffffffffffffff166109c7565b905092915050565b600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b610e92610e8d336000357fffffffff00000000000000000000000000000000000000000000000000000000166112aa565b61151a565b610eab600460149054906101000a900460ff161561151a565b60008060043591506024359050806000191682600019163373ffffffffffffffffffffffffffffffffffffffff166000357fffffffff00000000000000000000000000000000000000000000000000000000167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19163460003660405180848152602001806020018281038252848482818152602001925080828437820191505094505050505060405180910390a4610fb1600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054846fffffffffffffffffffffffffffffffff16611997565b600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550611012600054846fffffffffffffffffffffffffffffffff16611997565b600081905550505050565b60055481565b600061103e600460149054906101000a900460ff161561151a565b60008060043591506024359050806000191682600019163373ffffffffffffffffffffffffffffffffffffffff166000357fffffffff00000000000000000000000000000000000000000000000000000000167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19163460003660405180848152602001806020018281038252848482818152602001925080828437820191505094505050505060405180910390a46110f385856119b0565b9250505092915050565b61113361112e336000357fffffffff00000000000000000000000000000000000000000000000000000000166112aa565b61151a565b60008060043591506024359050806000191682600019163373ffffffffffffffffffffffffffffffffffffffff166000357fffffffff00000000000000000000000000000000000000000000000000000000167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19163460003660405180848152602001806020018281038252848482818152602001925080828437820191505094505050505060405180910390a46000600460146101000a81548160ff0219169083151502179055505050565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000600260008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b60003073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1614156112e95760019050611514565b600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1614156113485760019050611514565b600073ffffffffffffffffffffffffffffffffffffffff16600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614156113a85760009050611514565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663b70096138430856040518463ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff191681526020019350505050602060405180830381600087803b15156114de57600080fd5b5af115156114eb57600080fd5b5050506040513d602081101561150057600080fd5b810190808051906020019092919050505090505b92915050565b80151561152657600080fd5b50565b600081600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040518082815260200191505060405180910390a36001905092915050565b600081600160008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020541015151561166857fe5b81600260008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054101515156116f057fe5b611776600260008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205483611997565b600260008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555061183f600160008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205483611997565b600160008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506118cb600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548361197e565b600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a3600190509392505050565b6000828284019150811015151561199157fe5b92915050565b600082828403915081111515156119aa57fe5b92915050565b600081600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054101515156119fd57fe5b611a46600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205483611997565b600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550611ad2600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548361197e565b600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a360019050929150505600a165627a7a7230582020c10c08719c000ed33735cf57af3dd1ae897d2ebf0c7b10b98436e9ecf91cb00029";
