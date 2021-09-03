# OCR Deploy Script

A Go script that uses the integrations-framework as base, with small modifications, to ease the process of deploying and configuring an OCR Contract on an RSK Network.

## WIP

This script is in a very early and rough stage, expect it to have significant modifications in the future, and that it may require a slight tweak for it to work as expected

## Modifications

This script imports modules from the integrations-framework tests, with some slight modifications to run better on RSK nodes. Some private methods and struct fields from the modules were copied over to be able to use them. Also some methods were added:

GetClient: A method added to the BlockchainClient to allow it to be used as backend for calling InstanceLinkContract.

InstanceLinkContract: A method added to ContractDeployer that allows to Instantiate a previously deployed Link Contract.

AdjustGasPrice: A function that adds 2% to gas price read from eth_gasPrice, that works better on Rsk Testnet.

Rsk Networks: The configuration and logic needed for the client to be able to recognize and connect to RSK Reg Test and TestNet networks.

## The script

The script main function sets up an environment, deploys an OCR contract, funds the nodes accounts, configures the contract and creates the jobs. A valid adapter URL is required to properly function (replace the cryptocompare one)

SetupEnvironment: A function based on the DefaultLocalSetup function from the integrations-framework test suite, adapted to use an existing Link Token contract instead of deploying a new one.
