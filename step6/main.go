package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/defiweb/go-eth/abi"
	"github.com/defiweb/go-eth/crypto"
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

	uniswapSlot0 = abi.MustParseMethod(`
		function slot0() public view returns (
			uint160 sqrtPriceX96, 
			int24 tick, 
			uint16 observationIndex, 
			uint16 observationCardinality, 
			uint16 observationCardinalityNext, 
			uint8 feeProtocol, 
			bool unlocked
		)
	`)

	uniswapSwap = abi.MustParseMethod(`
		function swap(
			address pool,
			address recipient,
			bool zeroForOne,
			int256 amountSpecified,
			uint160 sqrtPriceLimitX96
		)
	`)
)

type UniswapSlot0 struct {
	SqrtPriceX96               *big.Int `abi:"sqrtPriceX96"`
	Tick                       int32    `abi:"tick"`
	ObservationIndex           uint16   `abi:"observationIndex"`
	observationCardinality     uint16   `abi:"observationCardinality"`
	ObservationCardinalityNext uint16   `abi:"observationCardinality"`
	FeeProtocol                uint8    `abi:"feeProtocol"`
	Unlocked                   bool     `abi:"unlocked"`
}

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

	// Compute the pool address.
	inverted, poolAddress := computePoolAddress(tokenIn, tokenOut, 10000)
	fmt.Printf("Pool address: %s\n", poolAddress.String())

	// Get the current slot0 of the Uniswap pool.
	slot0, err := callUniswapSlot0(ctx, client, poolAddress)
	if err != nil {
		panic(err)
	}

	// Print the current price.
	var currentPrice float64
	if inverted {
		currentPrice = 1 / sqrtPriceX96ToFloat(slot0.SqrtPriceX96, tokens[tokenOut].Decimals, tokens[tokenIn].Decimals)
	} else {
		currentPrice = sqrtPriceX96ToFloat(slot0.SqrtPriceX96, tokens[tokenIn].Decimals, tokens[tokenOut].Decimals)
	}
	fmt.Printf("Current price: %f\n", currentPrice)

	// Swap tokens.
	if inverted {
		fmt.Printf("Swapping %s for %s\n", tokens[tokenOut].Name, tokens[tokenIn].Name)
		hash, err := sendUniswapSwap(ctx, client, inverted, poolAddress, key.Address(), tokens[tokenIn].Balance)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Swap TX hash: %s\n", hash.String())
	} else {
		fmt.Printf("Swapping %s for %s\n", tokens[tokenOut].Name, tokens[tokenIn].Name)
		hash, err := sendUniswapSwap(ctx, client, inverted, poolAddress, key.Address(), tokens[tokenIn].Balance)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Swap TX hash: %s\n", hash.String())
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

// Uniswap factory and pool initialization code hash
var (
	uniswapFactory      = types.MustAddressFromHex("0x1F98431c8aD98523631AE4a59f267346ea31F984")
	uniswapPoolInitHash = types.MustHashFromHex("0xe34f199b19b2b4f47f68442619d555527d244f78a3297ea89325f843f87b8b54", types.PadNone)
)

func callUniswapSlot0(ctx context.Context, client rpc.RPC, poolAddr types.Address) (slot0 UniswapSlot0, err error) {
	callData, _ := uniswapSlot0.EncodeArgs()
	response, _, err := client.Call(
		ctx,
		types.Call{To: &poolAddr, Input: callData},
		types.LatestBlockNumber,
	)
	if err != nil {
		return UniswapSlot0{}, err
	}
	if err := uniswapSlot0.DecodeValue(response, &slot0); err != nil {
		return UniswapSlot0{}, err
	}
	return slot0, nil
}

// sendUniswapSwap sends a swap transaction to the Uniswap wrapper
func sendUniswapSwap(ctx context.Context, client rpc.RPC, inverted bool, poolAddr, recipientAddr types.Address, amountIn *big.Int) (*types.Hash, error) {
	minTickSqrtRatio, _ := new(big.Int).SetString("4295128740", 10)
	maxTickSqrtRatio, _ := new(big.Int).SetString("1461446703485210103287273052203988822378723970341", 10)
	sqrtPriceLimitX96 := minTickSqrtRatio
	if inverted {
		sqrtPriceLimitX96 = maxTickSqrtRatio
	}
	callData, err := uniswapSwap.EncodeArgs(poolAddr, recipientAddr, !inverted, amountIn, sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	}
	tx := types.Transaction{
		Call: types.Call{
			To:    &SwapContract,
			Input: callData,
		},
	}
	hash, _, err := client.SendTransaction(ctx, tx)
	return hash, err
}

// computePoolAddress computes the address of an Uniswap V3 pool.
//
// It returns the token0, token1, and pool address.
func computePoolAddress(token0, token1 types.Address, fee uint32) (inverted bool, pool types.Address) {
	if bytes.Compare(token0.Bytes(), token1.Bytes()) > 0 {
		token0, token1, inverted = token1, token0, true
	}
	var b bytes.Buffer
	b.WriteByte(0xff)
	b.Write(uniswapFactory.Bytes())
	b.Write(
		crypto.Keccak256(
			types.MustHashFromBytes(token0.Bytes(), types.PadLeft).Bytes(),
			types.MustHashFromBytes(token1.Bytes(), types.PadLeft).Bytes(),
			types.MustHashFromBigInt(big.NewInt(int64(fee))).Bytes()).Bytes(),
	)
	b.Write(uniswapPoolInitHash.Bytes())
	return inverted, types.MustAddressFromBytes(crypto.Keccak256(b.Bytes()).Bytes()[12:])
}

// sqrtPriceX96ToFloat converts a sqrtPriceX96 value to a float64 price
func sqrtPriceX96ToFloat(x *big.Int, token0Decimals, token1Decimals uint8) float64 {
	pow2n := new(big.Float).SetMantExp(big.NewFloat(1), 96)
	sqrt, _ := new(big.Float).Quo(new(big.Float).SetInt(x), pow2n).Float64()
	return sqrt * sqrt * math.Pow(10, float64(token0Decimals)-float64(token1Decimals))
}

// floatToSqrtPriceX96 converts a float64 price to a sqrtPriceX96 value
func floatToSqrtPriceX96(y float64, token0Decimals, token1Decimals uint8) *big.Int {
	adjustedY := y / math.Pow(10, float64(token0Decimals)-float64(token1Decimals))
	sqrtY := math.Sqrt(adjustedY)
	pow2n := new(big.Float).SetMantExp(big.NewFloat(1), 96)
	x, _ := new(big.Float).Mul(new(big.Float).SetFloat64(sqrtY), pow2n).Int(nil)
	return x
}
