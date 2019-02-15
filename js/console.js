var token;

won.defaultAccount = won.accounts[0];

setToken(tpcTokenAddress);

function checkAllBalances(type) {
    for (var acctNum in won.accounts) {
        var acct = won.accounts[acctNum];
        var wonBalance;
        var tpcBalance;
        var str = "  won.accounts[" + acctNum + "]:\t" + acct;
        if (typeof type === "undefined" || type === 1) {
            wonBalance = web3.fromWei(won.getBalance(acct), "won");
            str += "\tWON: " + wonBalance + " won,";
        }
        if (typeof type === "undefined" || type === 2) {
            tpcBalance = web3.fromWei(token.balanceOf(acct), "won");
            str += "\t"+web3.toUtf8(token.symbol())+": " + tpcBalance + " won";
        }
        console.log(str)
    }
}

function setToken(address) {
    token = web3.won.contract(tokenAbi).at(address);
}

function newToken(symbol, quantity) {
    if (typeof symbol === "undefined" || symbol === "") {
        console.info("Please input token symbol");
        return;
    }

    if (typeof quantity === "undefined" || quantity === "") {
        console.info("Please input token coin quantity");
        return;
    }

    var tokenContract = web3.won.contract(tokenAbi);
    tokenContract.new(symbol, {from: web3.won.defaultAccount, data: tokenByteCode, gas: 0x47b760}, function (e, contract) {
        if (e) {
            console.log("err creating contract", e);
        } else {
            if (!contract.address) {
                console.log("Contract transaction send: TransactionHash: " + contract.transactionHash + " waiting to be mined...");
            } else {
                var token = web3.won.contract(tokenAbi).at(contract.address);
                console.log("Contract mined! Token: "+web3.toUtf8(token.symbol())+", Address: " + contract.address);
                console.log("Start mint " + quantity + " coins.");
                var txid = token.mint(quantity);
                console.info(txid);
            }
        }
    });
}

function checkAllKycInfo() {
    won.accounts.forEach(function (e) {
        info = won.getKycInfo(e);
        console.log(e + " \tlevel: " + info.level + " \tzone:" + info.zone + " \tprovider:" + info.provider)
    })
}

function addPeer() {
    web3.admin.addPeer(nodePeer);
    web3.admin.peers
}

function findToken(startBlock, endBlock){
    for (var x=startBlock; x < endBlock; x++) {
        var transactions = won.getBlock(x).transactions;
        for (var y=0; y < transactions.length; y++) {
            var contractAddr = won.getTransactionReceipt(transactions[y]).contractAddress;
            if (contractAddr != null) {
                setToken(contractAddr);
                var symbol = web3.toUtf8(token.symbol());
                var decimals = token.decimals();
                console.log("Contract Address: " + contractAddr + "\tSymbol: " + symbol + "\tDecimals: " + decimals + "\tBlock: "+x);
            }
        }
    }
}

function listTransactions(startBlock, endBlock){
    try{
        for (var x=startBlock; x < endBlock; x++) {
            var transactions = won.getBlock(x).transactions;
            for (var y=0; y < transactions.length; y++) {
                var transaction = won.getTransaction(transactions[y]);
                var receipt = won.getTransactionReceipt(transactions[y]);
                if(receipt ==null || receipt==""){
                    continue;
                }

                var ouput = {
                    "tx":transaction.hash,
                    "from":receipt.from,
                    "to":receipt.to,
                    "blockNumber":transaction.blockNumber,
                    "gasPrice":transaction.gasPrice,
                    "gasSupply":transaction.gas,
                    "gasUsed":receipt.gasUsed,
                    "status":receipt.status,
                };
                console.log(JSON.stringify(ouput));
            }
        }
    }catch (e){
        console.log(e);
    }

}
