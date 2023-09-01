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
	WETH           = types.MustAddressFromHex("0xb4fbf271143f4fbf7b91a5ded31805e42b2208d6")
	erc20BalanceOf = abi.MustParseMethod(`function balanceOf(address account) public view returns (uint256)`)
)

func main() {
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

	address := types.MustAddressFromHex("0x69B352cbE6Fc5C130b6F62cc8f30b9d7B0DC27d0")
	balance, err := callERC20BalanceOf(ctx, client, WETH, address)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Token balance: %s\n", balance.String())
}

// callERC20BalanceOf calls the balanceOf method of an ERC20 token,
func callERC20BalanceOf(ctx context.Context, client rpc.RPC, tokenAddr, accountAddr types.Address) (balance *big.Int, err error) {
	calldata, _ := erc20BalanceOf.EncodeArgs(accountAddr)
	response, _, err := client.Call(
		ctx,
		types.Call{To: &tokenAddr, Input: calldata},
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
