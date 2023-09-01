# Ethereum Workshop: Interacting with Ethereum using Go

In this workshop, we will create a Go program that interacts with the Ethereum blockchain to check balances, approve
tokens, and perform a swap on Uniswap.

## Workshop Structure

1. **Introduction**

    - Brief overview of Ethereum, smart contracts, and the ERC20 token standard.
    - Introduction to the [go-eth](https://github.com/defiweb/go-eth) package and its capabilities.
    - Sending transactions to the Uniswap V3 smart contracts.

2. **Setting up the Environment**

    - Installing Go.
    - Setting up the [go-eth](https://github.com/defiweb/go-eth) package and other dependencies.
    - Setting up MetaMask and getting some testnet ETH.

3. **Coding Session**

    1. **Environment Setup and Project Initialization**:
        - Setting up the Go development environment by installing the necessary Go packages.
        - Creating a Go project and writing the basic structure of the Go program, which includes importing the required
          packages and defining the main function.

    2. **Executing ERC20 Contract Calls**:
        - Extending the program to interact with the Ethereum blockchain by executing simple ERC20 contract calls.
        - Retrieving the token balance of an Ethereum address by interacting with the ERC20 smart contract.

    3. **Interacting with ERC20 Smart Contract**:
        - Further extending the program by adding more functionality to interact with the ERC20 smart contract.

    4. **Loading Private Key and Approving Tokens:**:
        - Extending the program to load an Ethereum private key.
        - Adding the functionality to approve a token for spending by interacting with the ERC20 smart contract. This
          approval will be necessary to interact with the Uniswap V3 smart contracts in the next steps.

    5. **Retrieving Token Price from Uniswap V3**:
        - Adding the functionality to get the current price of a token pair from the Uniswap V3 smart contracts.
        - This involves interacting with the Uniswap V3 smart contracts to retrieve the current state of the pool and
          then calculating the price of the token pair.

    6. **Sending Swap Transaction to Uniswap V3**:
        - Extending the program to send a swap transaction to the Uniswap V3 smart contracts.
        - This transaction will swap one token for another by interacting with the Uniswap V3 smart contracts. This
          involves specifying the token pair, the amount to swap, and the price limit.

   Each step involves extending the Go program to add more functionalities and interact with the Ethereum blockchain and
   Uniswap V3 smart contracts. The final program will be able to approve tokens for spending and execute a token swap on
   Uniswap V3.

4. **Conclusion and Q&A**

    - Summary of the workshop.
    - Open the floor for questions.

## Prerequisites

- Basic knowledge of Go programming language.
- Basic understanding of Ethereum and smart contracts.

## Environment Setup

### Installing Go

#### Linux

1. Update your package list and install Go.

    ```
    sudo apt update
    sudo apt install golang-go
    ```

2. Verify the installation.

    ```
    go version
    ```

#### macOS

1. You can install Go on macOS using Homebrew.

    ```
    brew install go
    ```

2. Verify the installation.

    ```
    go version
    ```

#### Windows

1. Download the Go installer from the [official website](https://golang.org/dl/).
2. Run the installer and follow the prompts to install Go.
3. Verify the installation by opening a command prompt and running:

```
go version
```

### Setting up MetaMask

1. Install the MetaMask browser extension from the [official website](https://metamask.io/).
2. Create an account and switch to the Goerli Test Network.
3. Get some testnet ETH from the [Goerli faucet](https://goerlifaucet.com/).

## Running the Examples

The examples are stored in separate directories: `step1`, `step2`, `step3`, and so on. To run the examples, you need to
first install the dependencies and then run the code.

1. After cloning the repository, change to the repository directory.

    ```
    cd ethereum-workshop
    ```

2. Install the dependencies.

    ```
    go mod vendor
    ```

3. Run the examples.

    ```
    go run ./step1/main.go
    go run ./step2/main.go
    go run ./step3/main.go
    ...
    ```

## License

[MIT](LICENSE)
