package main

import (
	"context"
	"fmt"

	"github.com/defiweb/go-eth/rpc"
	"github.com/defiweb/go-eth/rpc/transport"
	"github.com/defiweb/go-eth/types"
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
	balance, err := client.GetBalance(ctx, address, types.LatestBlockNumber)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ETH balance: %s\n", balance.String())
}
