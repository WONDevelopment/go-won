contract dpos{
	


	mapping(address => string) public producers;

	mapping(address => uint)  public producers_stakes;

	mapping(address => mapping (address => uint) ) public voter_to_producers;

	address [21]  public selected_producers; // only 21 bp

	uint    total_vote_stake;



	function comparestr(string _a, string _b) returns (int) {
        bytes memory a = bytes(_a);
        bytes memory b = bytes(_b);
        uint minLength = a.length;
        if (b.length < minLength) minLength = b.length;
        //@todo unroll the loop into increments of 32 and do full 32 byte comparisons
        for (uint i = 0; i < minLength; i ++)
            if (a[i] < b[i])
                return -1;
            else if (a[i] > b[i])
                return 1;
        if (a.length < b.length)
            return -1;
        else if (a.length > b.length)
            return 1;
        else
            return 0;
    }


	function regProducer(string url) 
	{
		assert(comparestr(url,"")!=0);
		producers[msg.sender] = url;
	}

	function unregProducer()
	{
		 assert(comparestr(producers[msg.sender],"") !=0);
		 delete producers[msg.sender];
	}

	function updateActivesProducer(address pb) internal 
	{

		uint c = producers_stakes[pb];
		for (uint i=0;i < 21; i++)
		{
			if(selected_producers[i] != address(0))
			{
				if(c > producers_stakes[selected_producers[i]] )
				{
					address last = selected_producers[i];
					selected_producers[i] = pb;
					for(uint k=i+1; k < 21; k++)
					{
						address rpa =   selected_producers[k];
						selected_producers[k] = last;
						last = rpa;
					}

					break;
				}
			}
			else
			{
				selected_producers[i] = pb;
				break;
			}
		}
	}


	function voteProducer(address pb) public payable{
		assert(comparestr(producers[pb],"")!=0);
		voter_to_producers[msg.sender][pb] += msg.value;
		producers_stakes[pb] += msg.value;
		total_vote_stake += msg.value;

		if(total_vote_stake >= 15000000)
		{
			updateActivesProducer(pb);
		}

	}

	function unVoteProducer(address pb) public payable{
		voter_to_producers[msg.sender][pb] -= msg.value;
	}

	function getActiveProducers() public returns (address [21] producers)
	{
		return selected_producers;
	}


	function getProducersInfo(address pb) public returns (string url)
	{
		return  producers[pb];
	}

}