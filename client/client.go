package client

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
	ifClient "github.com/smartcontractkit/integrations-framework/client"
	"github.com/smartcontractkit/integrations-framework/config"
)

const (
	RskRegTestID ifClient.BlockchainNetworkID = "rsk_regtest"
	RskTestnetID ifClient.BlockchainNetworkID = "rsk_testnet"
)

// BlockchainClient is the interface that wraps a given client implementation for a blockchain, to allow for switching
// of network types within the test suite
type BlockchainClient interface {
	Get() interface{}
	GetClient() *ifClient.EthereumClient
	Fund(fromWallet ifClient.BlockchainWallet, toAddress string, nativeAmount, linkAmount *big.Int) error
}

// RskClient wraps the client and the BlockChain network to interact with an EVM based Blockchain
type RskClient struct {
	*ifClient.EthereumClient
}

// GetClient returns the actual EthereumClient instance to be used as backend
func (c *RskClient) GetClient() *ifClient.EthereumClient {
	return c.EthereumClient
}

// NewRskDevNetwork prepares settings for a connection to an RSK development network
func NewRskDevNetwork(conf *config.Config) (ifClient.BlockchainNetwork, error) {
	return newEthereumNetwork(conf, RskRegTestID)
}

// NewRskTestNetwork prepares settings for a connection to the RSK testnet
func NewRskTestNetwork(conf *config.Config) (ifClient.BlockchainNetwork, error) {
	return newEthereumNetwork(conf, RskTestnetID)
}

// NewBlockchainClient returns an instantiated network client implementation based on the network configuration given
func NewBlockchainClient(network ifClient.BlockchainNetwork) (*RskClient, error) {
	switch network.ID() {
	case ifClient.EthereumHardhatID, ifClient.EthereumKovanID, ifClient.EthereumGoerliID, RskRegTestID, RskTestnetID:
		return NewEthereumClient(network)
	}
	return nil, errors.New("invalid blockchain network ID, not found")
}

// NewEthereumClient returns an instantiated instance of the Ethereum client that has connected to the server
func NewEthereumClient(network ifClient.BlockchainNetwork) (*RskClient, error) {
	cl, err := ethclient.Dial(network.URL())
	if err != nil {
		return nil, err
	}

	return &RskClient{
		&ifClient.EthereumClient{
			Network: network,
			Client:  cl,
		},
	}, nil
}
