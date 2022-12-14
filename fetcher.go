package dexsdk

import (
	"errors"
	"math/big"
	"sync"

	"github.com/Jonescy/dex-sdk/abi/erc20"
	"github.com/Jonescy/dex-sdk/abi/pair-bsc"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type Fetcher struct {
	backend bind.ContractBackend
	ChainId ChainId
	caller  *bind.CallOpts
}

func NewFetcher(b bind.ContractBackend, chainId ChainId) Fetcher {
	return Fetcher{
		b,
		chainId,
		&bind.CallOpts{},
	}
}

// GetReverses currently support bsc
func (f Fetcher) GetReverses(token0, token1 Token) (Pair, error) {
	if !token0.SortsBefore(token1) {
		token0, token1 = token1, token0
	}
	contract, err := pair.NewPair(GetAddress(token0, token1), f.backend)
	if err != nil {
		return Pair{}, err
	}
	result, err := contract.GetReserves(f.caller)
	if err != nil {
		return Pair{}, err
	}
	reverse0 := decimal.NewFromBigInt(result.Reserve0, 0)
	reverse1 := decimal.NewFromBigInt(result.Reserve1, 0)
	if BigOne.Equal(reverse0) || BigOne.Equal(reverse1) {
		return Pair{}, InsufficientAmountError
	}
	return NewPair(NewTokenAmount(token0, reverse0), NewTokenAmount(token1, reverse1)), nil
}

func (f Fetcher) GetTokenInfo(address common.Address) (Token, error) {
	var decimals uint8
	var name string
	var symbol string
	var wg sync.WaitGroup
	contract, err := erc20.NewErc20Caller(address, f.backend)
	if err != nil {
		return Token{}, err
	}
	wg.Add(3)
	go func(caller *bind.CallOpts) {
		decimals, _ = contract.Decimals(caller)
		wg.Done()
	}(f.caller)
	go func(caller *bind.CallOpts) {
		name, _ = contract.Name(caller)
		wg.Done()
	}(f.caller)
	go func(caller *bind.CallOpts) {
		symbol, _ = contract.Symbol(caller)
		wg.Done()
	}(f.caller)
	wg.Wait()
	if decimals == 0 {
		return Token{}, errors.New("no decimal")
	}
	if name == "" {
		return Token{}, errors.New("no name")
	}
	if symbol == "" {
		return Token{}, errors.New("no symbol")
	}
	return NewToken(address.String(), int8(decimals), f.ChainId, name, symbol), nil
}

func (f Fetcher) GetTotalSupply(address common.Address) (*big.Int, error) {
	contract, err := erc20.NewErc20Caller(address, f.backend)
	if err != nil {
		return nil, err
	}
	return contract.TotalSupply(f.caller)
}

func (f Fetcher) GetKLast(address common.Address) (*big.Int, error) {
	var contract, err = pair.NewPairCaller(address, f.backend)
	if err != nil {
		return nil, err
	}
	return contract.KLast(f.caller)
}
