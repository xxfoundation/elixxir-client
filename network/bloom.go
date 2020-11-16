package network

import (
	bloom "gitlab.com/elixxir/bloomfilter"
)

func a(){
	bloom.InitByParameters(bloomFilterSize, bloomFilterHashes)
}