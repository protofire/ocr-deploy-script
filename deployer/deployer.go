package deployer

import (
	"context"
	"errors"
	"math/big"
	"ocr-deploy-script/client"
	"time"

	ifClient "github.com/smartcontractkit/integrations-framework/client"
	bindings "github.com/smartcontractkit/integrations-framework/contracts"
	"github.com/smartcontractkit/integrations-framework/contracts/ethereum"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ocrConfigHelper "github.com/smartcontractkit/libocr/offchainreporting/confighelper"
)

// ContractDeployer is an interface for abstracting the contract deployment methods across network implementations
type IContractDeployer interface {
	DeployStorageContract(fromWallet ifClient.BlockchainWallet) (bindings.Storage, error)
	DeployFluxAggregatorContract(
		fromWallet ifClient.BlockchainWallet,
		fluxOptions bindings.FluxAggregatorOptions,
	) (bindings.FluxAggregator, error)
	DeployLinkTokenContract(fromWallet ifClient.BlockchainWallet) (bindings.LinkToken, error)
	InstanceLinkTokenContract(addr common.Address, bcClient ifClient.BlockchainClient, fromWallet ifClient.BlockchainWallet) (bindings.LinkToken, error)
	DeployOffChainAggregator(
		fromWallet ifClient.BlockchainWallet,
		offchainOptions bindings.OffchainOptions,
	) (bindings.OffchainAggregator, error)
	DeployVRFContract(fromWallet ifClient.BlockchainWallet) (bindings.VRF, error)
}

// NewContractDeployer returns an instance of a contract deployer based on the client type
func NewContractDeployer(bcClient client.BlockchainClient) (*RskContractDeployer, error) {
	switch clientImpl := bcClient.Get().(type) {
	case *ifClient.EthereumClient:
		return NewEthereumContractDeployer(clientImpl), nil
	}
	return nil, errors.New("unknown blockchain client implementation")
}

// EthereumContractDeployer provides the implementations for deploying ETH (EVM) based contracts
type RskContractDeployer struct {
	eth *ifClient.EthereumClient
}

// InstanceLinkTokenContract returns an instance of an existant LinkToken Contract
func (e *RskContractDeployer) InstanceLinkTokenContract(
	addr common.Address,
	bcClient client.BlockchainClient,
	fromWallet ifClient.BlockchainWallet,
) (bindings.LinkToken, error) {
	ethClient := bcClient.GetClient()
	link, err := ethereum.NewLinkToken(addr, ethClient.Client)

	return &EthereumLinkToken{
		client:       e.eth,
		linkToken:    link,
		callerWallet: fromWallet,
		address:      addr,
	}, err
}

// NewEthereumContractDeployer returns an instantiated instance of the ETH contract deployer
func NewEthereumContractDeployer(ethClient *ifClient.EthereumClient) *RskContractDeployer {
	return &RskContractDeployer{
		eth: ethClient,
	}
}

// DefaultFluxAggregatorOptions produces some basic defaults for a flux aggregator contract
func DefaultFluxAggregatorOptions() bindings.FluxAggregatorOptions {
	return bindings.FluxAggregatorOptions{
		PaymentAmount: big.NewInt(1),
		Timeout:       uint32(30),
		MinSubValue:   big.NewInt(3),
		MaxSubValue:   big.NewInt(7),
		Decimals:      uint8(0),
		Description:   "Hardhat Flux Aggregator",
	}
}

// DeployFluxAggregatorContract deploys the Flux Aggregator Contract on an EVM chain
func (e *RskContractDeployer) DeployFluxAggregatorContract(
	fromWallet ifClient.BlockchainWallet,
	fluxOptions bindings.FluxAggregatorOptions,
) (bindings.FluxAggregator, error) {
	address, _, instance, err := e.eth.DeployContract(fromWallet, "Flux Aggregator", func(
		auth *bind.TransactOpts,
		backend bind.ContractBackend,
	) (common.Address, *types.Transaction, interface{}, error) {
		gasPrice, err := e.AdjustGasPrice()
		if err != nil {
			return common.Address{}, nil, nil, err
		}
		auth.GasPrice = gasPrice
		linkAddress := common.HexToAddress(e.eth.Network.Config().LinkTokenAddress)
		return ethereum.DeployFluxAggregator(auth,
			backend,
			linkAddress,
			fluxOptions.PaymentAmount,
			fluxOptions.Timeout,
			fluxOptions.Validator,
			fluxOptions.MinSubValue,
			fluxOptions.MaxSubValue,
			fluxOptions.Decimals,
			fluxOptions.Description)
	})
	if err != nil {
		return nil, err
	}
	return &EthereumFluxAggregator{
		client:         e.eth,
		fluxAggregator: instance.(*ethereum.FluxAggregator),
		callerWallet:   fromWallet,
		address:        address,
	}, nil
}

// DeployLinkTokenContract deploys a Link Token contract to an EVM chain
func (e *RskContractDeployer) DeployLinkTokenContract(fromWallet ifClient.BlockchainWallet) (bindings.LinkToken, error) {
	linkTokenAddress, _, instance, err := e.eth.DeployContract(fromWallet, "LINK Token", func(
		auth *bind.TransactOpts,
		backend bind.ContractBackend,
	) (common.Address, *types.Transaction, interface{}, error) {
		gasPrice, err := e.AdjustGasPrice()
		if err != nil {
			return common.Address{}, nil, nil, err
		}
		auth.GasPrice = gasPrice
		return ethereum.DeployLinkToken(auth, backend)
	})
	if err != nil {
		return nil, err
	}
	// Set config address
	e.eth.Network.Config().LinkTokenAddress = linkTokenAddress.Hex()
	return &EthereumLinkToken{
		client:       e.eth,
		linkToken:    instance.(*ethereum.LinkToken),
		callerWallet: fromWallet,
		address:      *linkTokenAddress,
	}, err
}

// DefaultOffChainAggregatorOptions returns some base defaults for deploying an OCR contract
func DefaultOffChainAggregatorOptions() bindings.OffchainOptions {
	return bindings.OffchainOptions{
		MaximumGasPrice:         uint32(500000000),
		ReasonableGasPrice:      uint32(28000),
		MicroLinkPerEth:         uint32(500),
		LinkGweiPerObservation:  uint32(500),
		LinkGweiPerTransmission: uint32(500),
		MinimumAnswer:           big.NewInt(1),
		MaximumAnswer:           big.NewInt(5000),
		Decimals:                8,
		Description:             "Test OCR",
	}
}

// DefaultOffChainAggregatorConfig returns some base defaults for configuring an OCR contract
func DefaultOffChainAggregatorConfig() bindings.OffChainAggregatorConfig {
	return bindings.OffChainAggregatorConfig{
		AlphaPPB:         1,
		DeltaC:           time.Minute * 10,
		DeltaGrace:       time.Second,
		DeltaProgress:    time.Second * 30,
		DeltaStage:       time.Second * 10,
		DeltaResend:      time.Second * 10,
		DeltaRound:       time.Second * 20,
		RMax:             4,
		S:                []int{1, 1, 1, 1, 1},
		N:                5,
		F:                1,
		OracleIdentities: []ocrConfigHelper.OracleIdentityExtra{},
	}
}

// DeployOffChainAggregator deploys the offchain aggregation contract to the EVM chain
func (e *RskContractDeployer) DeployOffChainAggregator(
	fromWallet ifClient.BlockchainWallet,
	offchainOptions bindings.OffchainOptions,
) (bindings.OffchainAggregator, error) {
	address, _, instance, err := e.eth.DeployContract(fromWallet, "OffChain Aggregator", func(
		auth *bind.TransactOpts,
		backend bind.ContractBackend,
	) (common.Address, *types.Transaction, interface{}, error) {
		gasPrice, err := e.AdjustGasPrice()
		if err != nil {
			return common.Address{}, nil, nil, err
		}
		auth.GasPrice = gasPrice
		linkAddress := common.HexToAddress(e.eth.Network.Config().LinkTokenAddress)
		return ethereum.DeployOffchainAggregator(auth,
			backend,
			offchainOptions.MaximumGasPrice,
			offchainOptions.ReasonableGasPrice,
			offchainOptions.MicroLinkPerEth,
			offchainOptions.LinkGweiPerObservation,
			offchainOptions.LinkGweiPerTransmission,
			linkAddress,
			offchainOptions.MinimumAnswer,
			offchainOptions.MaximumAnswer,
			offchainOptions.BillingAccessController,
			offchainOptions.RequesterAccessController,
			offchainOptions.Decimals,
			offchainOptions.Description)
	})
	if err != nil {
		return nil, err
	}
	return &EthereumOffchainAggregator{
		client:       e.eth,
		ocr:          instance.(*ethereum.OffchainAggregator),
		callerWallet: fromWallet,
		address:      address,
	}, err
}

// DeployStorageContract deploys a vanilla storage contract that is a value store
func (e *RskContractDeployer) DeployStorageContract(fromWallet ifClient.BlockchainWallet) (bindings.Storage, error) {
	_, _, instance, err := e.eth.DeployContract(fromWallet, "Storage", func(
		auth *bind.TransactOpts,
		backend bind.ContractBackend,
	) (common.Address, *types.Transaction, interface{}, error) {
		gasPrice, err := e.AdjustGasPrice()
		if err != nil {
			return common.Address{}, nil, nil, err
		}
		auth.GasPrice = gasPrice
		return ethereum.DeployStore(auth, backend)
	})
	if err != nil {
		return nil, err
	}
	return &EthereumStorage{
		client:       e.eth,
		store:        instance.(*ethereum.Store),
		callerWallet: fromWallet,
	}, err
}

func (e *RskContractDeployer) DeployVRFContract(fromWallet ifClient.BlockchainWallet) (bindings.VRF, error) {
	address, _, instance, err := e.eth.DeployContract(fromWallet, "VRF", func(
		auth *bind.TransactOpts,
		backend bind.ContractBackend,
	) (common.Address, *types.Transaction, interface{}, error) {
		gasPrice, err := e.AdjustGasPrice()
		if err != nil {
			return common.Address{}, nil, nil, err
		}
		auth.GasPrice = gasPrice
		return ethereum.DeployVRF(auth, backend)
	})
	if err != nil {
		return nil, err
	}
	return &EthereumVRF{
		client:       e.eth,
		vrf:          instance.(*ethereum.VRF),
		callerWallet: fromWallet,
		address:      address,
	}, err
}

func (e *RskContractDeployer) AdjustGasPrice() (*big.Int, error) {
	gasPrice, err := e.eth.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	chainId := e.eth.Network.ChainID()
	if chainId == big.NewInt(30) || chainId == big.NewInt(31) || chainId == big.NewInt(33) {
		x, y, z := big.NewInt(0), big.NewInt(0), big.NewInt(0)
		x.Add(gasPrice, y.Div(z.Mul(gasPrice, big.NewInt(2)), big.NewInt(100)))
		return x, nil
	} else {
		return gasPrice, nil
	}
}
