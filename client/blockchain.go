package client

import (
	"fmt"
	"math/big"
	"strings"

	ifClient "github.com/smartcontractkit/integrations-framework/client"
	"github.com/smartcontractkit/integrations-framework/config"
)

// EthereumNetwork is the implementation of BlockchainNetwork for the local ETH dev server
type EthereumNetwork struct {
	networkID     ifClient.BlockchainNetworkID
	networkConfig *config.NetworkConfig
}

// ID returns the readable name of the EVM network
func (e *EthereumNetwork) ID() ifClient.BlockchainNetworkID {
	return e.networkID
}

// URL returns the RPC URL used for connecting to the network
func (e *EthereumNetwork) URL() string {
	return e.networkConfig.URL
}

// ChainID returns the on-chain ID of the network being connected to
func (e *EthereumNetwork) ChainID() *big.Int {
	return big.NewInt(e.networkConfig.ChainID)
}

// Config returns the blockchain network configuration
func (e *EthereumNetwork) Config() *config.NetworkConfig {
	return e.networkConfig
}

// Wallets returns all the viable wallets used for testing on chain
func (e *EthereumNetwork) Wallets() (ifClient.BlockchainWallets, error) {
	return newEthereumWallets(e.networkConfig.PrivateKeyStore)
}

// Wallets is the default implementation of BlockchainWallets that holds a slice of wallets with the default
type Wallets struct {
	defaultWallet int
	wallets       []ifClient.BlockchainWallet
}

// Default returns the default wallet to be used for a transaction on-chain
func (w *Wallets) Default() ifClient.BlockchainWallet {
	return w.wallets[w.defaultWallet]
}

// All returns the raw representation of Wallets
func (w *Wallets) All() []ifClient.BlockchainWallet {
	return w.wallets
}

// SetDefault changes the default wallet to be used for on-chain transactions
func (w *Wallets) SetDefault(i int) error {
	if err := walletSliceIndexInRange(w.wallets, i); err != nil {
		return err
	}
	w.defaultWallet = i
	return nil
}

// Wallet returns a wallet based on a given index in the slice
func (w *Wallets) Wallet(i int) (ifClient.BlockchainWallet, error) {
	if err := walletSliceIndexInRange(w.wallets, i); err != nil {
		return nil, err
	}
	return w.wallets[i], nil
}

// newEthereumNetwork creates a way to interact with any specified EVM blockchain
func newEthereumNetwork(conf *config.Config, networkID ifClient.BlockchainNetworkID) (ifClient.BlockchainNetwork, error) {
	networkConf, err := conf.GetNetworkConfig(string(networkID))
	if err != nil {
		return nil, err
	}
	return &EthereumNetwork{
		networkID:     networkID,
		networkConfig: networkConf,
	}, nil
}

func newEthereumWallets(pkStore config.PrivateKeyStore) (ifClient.BlockchainWallets, error) {
	// Check private keystore value, create wallets from such
	var processedWallets []ifClient.BlockchainWallet
	keys, err := pkStore.Fetch()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		wallet, err := ifClient.NewEthereumWallet(strings.TrimSpace(key))
		if err != nil {
			return &Wallets{}, err
		}
		processedWallets = append(processedWallets, wallet)
	}

	return &Wallets{
		defaultWallet: 0,
		wallets:       processedWallets,
	}, nil
}

func walletSliceIndexInRange(wallets []ifClient.BlockchainWallet, i int) error {
	if i > len(wallets)-1 {
		return fmt.Errorf("invalid index in list of wallets")
	}
	return nil
}
