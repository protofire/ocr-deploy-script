package main

import (
	"context"
	"fmt"
	"math/big"
	"ocr-deploy-script/client"
	"ocr-deploy-script/deployer"
	"time"

	"github.com/avast/retry-go"
	"github.com/ethereum/go-ethereum/common"
	ifclient "github.com/smartcontractkit/integrations-framework/client"
	"github.com/smartcontractkit/integrations-framework/config"
	"github.com/smartcontractkit/integrations-framework/contracts"
	"github.com/smartcontractkit/integrations-framework/suite"
)

func main() {
	config := contracts.DefaultOffChainAggregatorConfig()
	fmt.Println(config)

	// s, err := SetupEnvironment(client.IEthClient.NewRskDevNetwork)
	s, err := SetupEnvironment(client.NewRskDevNetwork)
	if err != nil {
		fmt.Println(err)
	}

	ocrOptions := contracts.DefaultOffChainAggregatorOptions()

	// Connect to running chainlink nodes
	chainlinkNodes, _, err := suite.ConnectToTemplateNodes()
	if err != nil {
		fmt.Println(err)
	}

	// Deploy and config OCR contract
	ocrInstance, err := s.Deployer.DeployOffChainAggregator(s.Wallets.Default(), ocrOptions)
	if err != nil {
		fmt.Println(err)
	}

	err = ocrInstance.SetConfig(s.Wallets.Default(), chainlinkNodes, contracts.DefaultOffChainAggregatorConfig())
	if err != nil {
		fmt.Println(err)
	}

	linkaddr, err := ocrInstance.Link(context.Background())
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(linkaddr)

	err = ocrInstance.Fund(s.Wallets.Default(), big.NewInt(100000000000000), big.NewInt(2000000000000000))
	if err != nil {
		fmt.Println(err)
	}

	balance, err := s.Link.BalanceOf(context.Background(), common.HexToAddress("0x84eA74d481Ee0A5332c457a4d796187F6Ba67fEB"))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(balance)

	// Initialize bootstrap node
	bootstrapNode := chainlinkNodes[0]
	bootstrapP2PIds, err := bootstrapNode.ReadP2PKeys()
	if err != nil {
		fmt.Println(err)
	}
	bootstrapP2PId := bootstrapP2PIds.Data[0].Attributes.PeerID
	bootstrapSpec := &ifclient.OCRBootstrapJobSpec{
		ContractAddress: ocrInstance.Address(),
		P2PPeerID:       bootstrapP2PId,
		IsBootstrapPeer: true,
	}

	_, err = bootstrapNode.CreateJob(bootstrapSpec)
	if err != nil {
		fmt.Println(err)
	}

	// Send OCR job to other nodes
	for index := 1; index < len(chainlinkNodes); index++ {
		nodeP2PIds, err := chainlinkNodes[index].ReadP2PKeys()
		if err != nil {
			fmt.Println(err)
		}

		nodeP2PId := nodeP2PIds.Data[0].Attributes.PeerID
		nodeTransmitterAddresses, err := chainlinkNodes[index].ReadETHKeys()
		if err != nil {
			fmt.Println(err)
		}

		nodeTransmitterAddress := nodeTransmitterAddresses.Data[0].Attributes.Address
		nodeOCRKeys, err := chainlinkNodes[index].ReadOCRKeys()
		if err != nil {
			fmt.Println(err)
		}

		nodeOCRKeyId := nodeOCRKeys.Data[0].ID

		ocrSpec := &ifclient.OCRTaskJobSpec{
			ContractAddress:    ocrInstance.Address(),
			P2PPeerID:          nodeP2PId,
			P2PBootstrapPeers:  []string{bootstrapP2PId},
			KeyBundleID:        nodeOCRKeyId,
			TransmitterAddress: nodeTransmitterAddress,
			ObservationSource:  ifclient.ObservationSourceSpec("http://cryptocompare:8080"),
		}
		_, err = chainlinkNodes[index].CreateJob(ocrSpec)

		if err != nil {
			fmt.Println(err)
		}

	}

}

func SetupEnvironment(initFunc ifclient.BlockchainNetworkInit) (*suite.DefaultSuiteSetup, error) {
	conf, err := config.NewWithPath(config.LocalConfig, "config")
	if err != nil {
		return nil, err
	}
	networkConfig, err := initFunc(conf)
	if err != nil {
		return nil, err
	}
	blockchainClient, err := client.NewBlockchainClient(networkConfig)
	if err != nil {
		return nil, err
	}
	wallets, err := networkConfig.Wallets()
	if err != nil {
		return nil, err
	}
	contractDeployer, err := deployer.NewContractDeployer(blockchainClient)
	if err != nil {
		return nil, err
	}

	linkTokenAddress := common.HexToAddress(blockchainClient.GetClient().Network.Config().LinkTokenAddress)

	link, err := contractDeployer.InstanceLinkTokenContract(linkTokenAddress, blockchainClient, wallets.Default())
	fmt.Println(link.Address())

	if err != nil {
		return nil, err
	}

	// configure default retry
	retry.DefaultAttempts = conf.Retry.Attempts
	// linear waiting
	retry.DefaultDelayType = func(n uint, err error, config *retry.Config) time.Duration {
		return conf.Retry.LinearDelay
	}
	return &suite.DefaultSuiteSetup{
		Config:   conf,
		Client:   blockchainClient,
		Wallets:  wallets,
		Deployer: contractDeployer,
		Link:     link,
	}, nil
}
