package solidity

const tmplGroth16 = `package main

import (
	"encoding/hex"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/{{ .Curve }}/fr"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	proofHex = "{{ .Proof }}"
	inputHex = "{{ .PublicInputs }}"
	nbPublicInputs = {{ .NbPublicInputs }}
)

func main() {
	const gasLimit uint64 = 4712388

	// setup simulated backend
	key, _ := crypto.GenerateKey()
	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	checkErr(err, "init keyed transactor")

	genesis := map[common.Address]core.GenesisAccount{
		auth.From: {Balance: big.NewInt(1000000000000000000)}, // 1 Eth
	}
	backend := backends.NewSimulatedBackend(genesis, gasLimit)

	// deploy verifier contract
	_, _, verifierContract, err := DeployVerifier(auth, backend)
	checkErr(err, "deploy verifier contract failed")
	backend.Commit()


	proofBytes, err := hex.DecodeString(proofHex)
	checkErr(err, "decode proof hex failed")

	inputBytes, err := hex.DecodeString(inputHex)
	checkErr(err, "decode input hex failed")

	if len(inputBytes)%fr.Bytes != 0 {
		panic("inputBytes mod fr.Bytes !=0")
	}

	// convert public inputs
	nbInputs := len(inputBytes) / fr.Bytes
	if nbInputs != nbPublicInputs {
		panic("nbInputs != nbPublicInputs")
	}
	var input [nbPublicInputs]*big.Int
	for i := 0; i < nbInputs; i++ {
		var e fr.Element
		e.SetBytes(inputBytes[fr.Bytes*i : fr.Bytes*(i+1)])
		input[i] = new(big.Int)
		e.BigInt(input[i])
	}

	// verifyProof takes bytes calldata proof
	// proofBytes is the raw MarshalSolidity() output passed directly
	err = verifierContract.VerifyProof(&bind.CallOpts{}, proofBytes, input)
	checkErr(err, "calling verifier on chain gave error")
{{ if eq .Curve "bn254" }}
	// compress proof (only available for BN254)
	compressed, err := verifierContract.CompressProof(&bind.CallOpts{}, proofBytes)
	checkErr(err, "compressing proof gave error")
{{ if eq .NbCommitments 0 }}
	// verify compressed proof
	err = verifierContract.VerifyCompressedProof(&bind.CallOpts{}, compressed, input)
	checkErr(err, "calling verifier with compressed proof on chain gave error")
{{ else }}
	// verify compressed proof
	err = verifierContract.VerifyCompressedProof(&bind.CallOpts{}, compressed.Compressed, compressed.CompressedCommitments, compressed.CompressedCommitmentPok, input)
	checkErr(err, "calling verifier with compressed proof on chain gave error")
{{ end }}
{{ end }}
}

func checkErr(err error, ctx string) {
	if err != nil {
		panic(ctx + " " + err.Error())
	}
}

`

const tmplPlonK = `package main


import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"

	"github.com/consensys/gnark-crypto/ecc/{{ .Curve }}/fr"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	proofHex = "{{ .Proof }}"
	inputHex = "{{ .PublicInputs }}"
	nbPublicInputs = {{ .NbPublicInputs }}
)

func main() {
	const gasLimit uint64 = 4712388

	// setup simulated backend
	key, _ := crypto.GenerateKey()
	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	checkErr(err, "init keyed transactor")

	genesis := map[common.Address]core.GenesisAccount{
		auth.From: {Balance: big.NewInt(1000000000000000000)}, // 1 Eth
	}
	backend := backends.NewSimulatedBackend(genesis, gasLimit)

	// deploy verifier contract
	_, _, verifierContract, err := DeployPlonkVerifier(auth, backend)
	checkErr(err, "deploy verifier contract failed")
	backend.Commit()


	proofBytes, err := hex.DecodeString(proofHex)
	checkErr(err, "decode proof hex failed")


	inputBytes, err := hex.DecodeString(inputHex)
	checkErr(err, "decode input hex failed")

	if len(inputBytes)%fr.Bytes != 0 {
		panic("inputBytes mod fr.Bytes !=0")
	}

	// convert public inputs
	nbInputs := len(inputBytes) / fr.Bytes
	if nbInputs != nbPublicInputs {
		panic("nbInputs != nbPublicInputs")
	}
	var input [nbPublicInputs]*big.Int
	for i := 0; i < nbInputs; i++ {
		var e fr.Element
		e.SetBytes(inputBytes[fr.Bytes*i : fr.Bytes*(i+1)])
		input[i] = new(big.Int)
		e.BigInt(input[i])
	}


	// call the contract
	res, err := verifierContract.Verify(&bind.CallOpts{}, proofBytes[:], input[:])
	checkErr(err, "calling verifier on chain gave error")
	if res {
		fmt.Println("proof is valid")
	} else {
		fmt.Println("proof is invalid")
		os.Exit(42)
	}
}

func checkErr(err error, ctx string) {
	if err != nil {
		panic(ctx + " " + err.Error())
	}
}

`

const tmplGoMod = `module tmpsolidity

go 1.25.6

require (
	github.com/consensys/gnark-crypto v0.19.3-0.20260210161253-243aa4749a4a
	github.com/ethereum/go-ethereum v1.16.8
)
`
