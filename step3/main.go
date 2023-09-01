package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/defiweb/go-eth/abi"
	"github.com/defiweb/go-eth/rpc"
	"github.com/defiweb/go-eth/rpc/transport"
	"github.com/defiweb/go-eth/types"
)

var (
	WETH = types.MustAddressFromHex("0xb4fbf271143f4fbf7b91a5ded31805e42b2208d6")
	USDC = types.MustAddressFromHex("0x07865c6e87b9f70255377e024ace6630c1eaa37f")
)

var (
	erc20Name      = abi.MustParseMethod(`function name() public view returns (string)`)
	erc20Decimals  = abi.MustParseMethod(`function decimals() public view returns (uint8)`)
	erc20BalanceOf = abi.MustParseMethod(`function balanceOf(address account) public view returns (uint256)`)
)

type Token struct {
	Name     string
	Decimals uint8
	Balance  *big.Int
}

func main() {
	// Tokens to swap.
	var (
		tokenIn  = WETH
		tokenOut = USDC
	)

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// Create a JSON-RPC transport.
	rpcTransport, err := transport.NewHTTP(transport.HTTPOptions{
		URL: "https://rpc.ankr.com/eth_goerli",
	})
	if err != nil {
		panic(err)
	}

	// Create a JSON-RPC client.
	client, err := rpc.NewClient(
		rpc.WithTransport(rpcTransport),
	)
	if err != nil {
		panic(err)
	}

	// Get token information.
	var tokens = make(map[types.Address]Token)
	for _, address := range []types.Address{tokenIn, tokenOut} {
		name, err := callERC20Name(ctx, client, address)
		if err != nil {
			panic(err)
		}
		decimals, err := callERC20Decimals(ctx, client, address)
		if err != nil {
			panic(err)
		}
		balance, err := callERC20BalanceOf(ctx, client, address, types.MustAddressFromHex("0x69B352cbE6Fc5C130b6F62cc8f30b9d7B0DC27d0"))
		if err != nil {
			panic(err)
		}

		tokens[address] = Token{
			Name:     name,
			Decimals: decimals,
			Balance:  balance,
		}
	}

	// Print token information.
	for _, token := range tokens {
		fmt.Printf("%s:\n", token.Name)
		fmt.Printf("\tToken balance: %s\n", token.Balance.String())
		fmt.Printf("\tToken decimals: %d\n", token.Decimals)
	}
}

// callERC20Name calls the name method of an ERC20 token.
func callERC20Name(ctx context.Context, client rpc.RPC, tokenAddr types.Address) (name string, err error) {
	callData, _ := erc20Name.EncodeArgs()
	response, _, err := client.Call(
		ctx,
		types.Call{To: &tokenAddr, Input: callData},
		types.LatestBlockNumber,
	)
	if err != nil {
		return "", err
	}
	if err := erc20Name.DecodeValues(response, &name); err != nil {
		return "", err
	}
	return name, nil
}

// callERC20Decimals calls the decimals method of an ERC20 token.
func callERC20Decimals(ctx context.Context, client rpc.RPC, tokenAddr types.Address) (decimals uint8, err error) {
	callData, _ := erc20Decimals.EncodeArgs()
	response, _, err := client.Call(
		ctx,
		types.Call{To: &tokenAddr, Input: callData},
		types.LatestBlockNumber,
	)
	if err != nil {
		return 0, err
	}
	if err := erc20Decimals.DecodeValues(response, &decimals); err != nil {
		return 0, err
	}
	return decimals, nil
}

// callERC20BalanceOf calls the balanceOf method of an ERC20 token.
func callERC20BalanceOf(ctx context.Context, client rpc.RPC, tokenAddr, accountAddr types.Address) (balance *big.Int, err error) {
	callData, _ := erc20BalanceOf.EncodeArgs(accountAddr)
	response, _, err := client.Call(
		ctx,
		types.Call{To: &tokenAddr, Input: callData},
		types.LatestBlockNumber,
	)
	if err != nil {
		return nil, err
	}
	if err := erc20BalanceOf.DecodeValues(response, &balance); err != nil {
		return nil, err
	}
	return balance, nil
}
