package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/defiweb/go-eth/abi"
	"github.com/defiweb/go-eth/hexutil"
	"github.com/defiweb/go-eth/rpc"
	"github.com/defiweb/go-eth/rpc/transport"
	"github.com/defiweb/go-eth/txmodifier"
	"github.com/defiweb/go-eth/types"
	"github.com/defiweb/go-eth/wallet"
)

var (
	WETH = types.MustAddressFromHex("0xb4fbf271143f4fbf7b91a5ded31805e42b2208d6")
	USDC = types.MustAddressFromHex("0x07865c6e87b9f70255377e024ace6630c1eaa37f")
)

var (
	SwapContract = types.MustAddressFromHex("0x1aa862951c58aEc5f2745F63575d91BaCCF8fc41")
)

var (
	erc20Name      = abi.MustParseMethod(`function name() public view returns (string)`)
	erc20Decimals  = abi.MustParseMethod(`function decimals() public view returns (uint8)`)
	erc20BalanceOf = abi.MustParseMethod(`function balanceOf(address account) public view returns (uint256)`)
	erc20Allowance = abi.MustParseMethod(`function allowance(address owner, address spender) public view returns (uint256)`)
	erc20Approve   = abi.MustParseMethod(`function approve(address spender, uint256 amount) public returns (bool)`)
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

	// Load the private key.
	key := wallet.NewKeyFromBytes(hexutil.MustHexToBytes("YOUR_KEY_HERE"))

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
		rpc.WithKeys(key),
		rpc.WithDefaultAddress(key.Address()),
		rpc.WithChainID(5),
		rpc.WithTXModifiers(
			txmodifier.NewNonceProvider(false),
			txmodifier.NewGasLimitEstimator(1.25, 0, 0),
			txmodifier.NewEIP1559GasFeeEstimator(1.5, 1.25, nil, nil, nil, nil),
		),
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
		balance, err := callERC20BalanceOf(ctx, client, address, key.Address())
		if err != nil {
			panic(err)
		}

		tokens[address] = Token{
			Name:     name,
			Decimals: decimals,
			Balance:  balance,
		}
	}

	// Approve the swap contract to spend the tokenIn.
	tokenInAllowance, err := callERC20Allowance(ctx, client, tokenIn, key.Address(), SwapContract)
	if err != nil {
		panic(err)
	}
	if tokenInAllowance.Cmp(tokens[tokenIn].Balance) < 0 {
		fmt.Printf("Approving %s %s\n", tokens[tokenIn].Balance.String(), tokens[tokenIn].Name)
		hash, err := sendERC20Approve(ctx, client, tokenIn, SwapContract, tokens[tokenIn].Balance)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Approve TX hash: %s\n", hash.String())
		fmt.Printf("Waiting for approval to be mined...\n")
		for {
			tx, err := client.GetTransactionByHash(ctx, *hash)
			if err != nil {
				panic(err)
			}
			if tx.BlockHash != nil {
				break
			}
			time.Sleep(5 * time.Second)
		}
	}

	fmt.Printf("Token approval complete!\n")
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

// callERC20Allowance calls the allowance method of an ERC20 token.
func callERC20Allowance(ctx context.Context, client rpc.RPC, tokenAddr, ownerAddr, spenderAddr types.Address) (allowance *big.Int, err error) {
	callData, _ := erc20Allowance.EncodeArgs(ownerAddr, spenderAddr)
	response, _, err := client.Call(
		ctx,
		types.Call{To: &tokenAddr, Input: callData},
		types.LatestBlockNumber,
	)
	if err != nil {
		return nil, err
	}
	if err := erc20Allowance.DecodeValues(response, &allowance); err != nil {
		return nil, err
	}
	return allowance, nil
}

// sendERC20Approve sends an approve transaction for an ERC20 token.
func sendERC20Approve(ctx context.Context, client rpc.RPC, tokenAddr, spenderAddr types.Address, amount *big.Int) (*types.Hash, error) {
	callData, err := erc20Approve.EncodeArgs(spenderAddr, amount)
	if err != nil {
		return nil, err
	}
	tx := types.Transaction{
		Call: types.Call{
			To:    &tokenAddr,
			Input: callData,
		},
	}
	hash, _, err := client.SendTransaction(ctx, tx)
	return hash, err
}
