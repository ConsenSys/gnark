package solcheck

// tmplMain is the program deploying the verifier contract on a simulated
// backend and verifying a proof with it. It uses the abigen v2 bindings in
// [bindingFile].
const tmplMain = `package main

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
)

const (
	proofHex       = "{{ .Proof }}"
	inputHex       = "{{ .PublicInputs }}"
	nbPublicInputs = {{ .NbPublicInputs }}
)

func main() {
	// setup simulated backend (chain ID 1337)
	key, err := crypto.GenerateKey()
	checkErr(err, "generate key")
	auth := bind.NewKeyedTransactor(key, big.NewInt(1337))
	sim := simulated.NewBackend(types.GenesisAlloc{
		auth.From: {Balance: big.NewInt(1000000000000000000)}, // 1 Eth
	})
	defer sim.Close()
	client := sim.Client()

	// deploy verifier contract
	addr, tx, err := bind.DeployContract(auth, common.FromHex({{ .Contract }}MetaData.Bin), client, nil)
	checkErr(err, "deploy verifier contract")
	sim.Commit()
	_, err = bind.WaitDeployed(context.Background(), client, tx.Hash())
	checkErr(err, "wait for verifier contract deployment")
	verifier := New{{ .Contract }}()
	instance := verifier.Instance(client, addr)

	proofBytes, err := hex.DecodeString(proofHex)
	checkErr(err, "decode proof hex")
	inputBytes, err := hex.DecodeString(inputHex)
	checkErr(err, "decode input hex")

	// convert public inputs (32 bytes big-endian each)
	var input [nbPublicInputs]*big.Int
	for i := range input {
		input[i] = new(big.Int).SetBytes(inputBytes[32*i : 32*(i+1)])
	}
{{ if eq .Backend "groth16" }}
	// verifyProof takes the raw MarshalSolidity() output and reverts on invalid proofs
	_, err = instance.CallRaw(nil, verifier.PackVerifyProof(proofBytes, input))
	checkErr(err, "calling verifier on chain")
{{- if eq .Curve "bn254" }}

	// compress proof (only available for BN254)
	compressed, err := bind.Call(instance, nil, verifier.PackCompressProof(proofBytes), verifier.UnpackCompressProof)
	checkErr(err, "compressing proof")
{{- if eq .NbCommitments 0 }}
	_, err = instance.CallRaw(nil, verifier.PackVerifyCompressedProof(compressed, input))
{{- else }}
	_, err = instance.CallRaw(nil, verifier.PackVerifyCompressedProof(compressed.Compressed, compressed.CompressedCommitments, compressed.CompressedCommitmentPok, input))
{{- end }}
	checkErr(err, "calling verifier with compressed proof on chain")
{{- end }}
{{- else }}
	// Verify returns false on invalid proofs
	valid, err := bind.Call(instance, nil, verifier.PackVerify(proofBytes, input[:]), verifier.UnpackVerify)
	checkErr(err, "calling verifier on chain")
	if !valid {
		panic("proof is invalid")
	}
{{- end }}
}

func checkErr(err error, ctx string) {
	if err != nil {
		panic(ctx + ": " + err.Error())
	}
}
`
