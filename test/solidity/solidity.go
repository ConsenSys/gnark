// Package solidity checks gnark-exported Solidity verifiers end-to-end.
//
// [Generate] compiles the contract with solc and generates Go bindings with
// abigen. [Verify] then generates a throwaway Go program that deploys the
// contract on a geth simulated backend and verifies a proof against it.
//
// Requirements (must be reachable in PATH):
//   - solc (https://docs.soliditylang.org/en/latest/installing-solidity.html)
//   - abigen (go install github.com/ethereum/go-ethereum/cmd/abigen@v1.16.8)
//
// The generated program lives in its own Go module, so go-ethereum is not a
// dependency of gnark; it is fetched by `go mod tidy` when running [Verify].
package solidity

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
)

const (
	bindingFile = "gnark_solidity.go"
	// frBytes is the size of a scalar field element for all supported curves.
	frBytes = 32
)

var (
	groth16Main = template.Must(template.New("groth16").Parse(tmplGroth16))
	plonkMain   = template.Must(template.New("plonk").Parse(tmplPlonK))
)

// Generate compiles dir/solidityFile with solc and generates the Go bindings
// of the resulting contracts with abigen. All outputs are written to dir.
func Generate(dir, solidityFile string) error {
	solPath := filepath.Join(dir, solidityFile)
	if _, err := os.Stat(solPath); err != nil {
		return fmt.Errorf("solidity file: %w", err)
	}
	if err := run(exec.Command("solc", "--evm-version", "cancun", "--combined-json", "abi,bin", solPath, "-o", dir, "--overwrite")); err != nil {
		return err
	}
	return run(exec.Command("abigen", "--combined-json", filepath.Join(dir, "combined.json"), "--pkg", "main", "--out", filepath.Join(dir, bindingFile)))
}

// Proof is a proof together with the data needed to verify it on chain.
type Proof struct {
	Backend backend.ID // [backend.GROTH16] or [backend.PLONK]
	Curve   ecc.ID     // [ecc.BN254] or [ecc.BLS12_381]
	// Proof is the output of MarshalSolidity() of the proof.
	Proof []byte
	// PublicInputs are the concatenated big-endian encoded public inputs.
	PublicInputs []byte
	// NbCommitments is the number of commitments in the proof (Groth16 only).
	NbCommitments int
}

// Verify deploys the contract previously processed by [Generate] in dir on a
// geth simulated backend and verifies p with it. It returns an error if the
// proof is rejected.
func Verify(dir string, p Proof) error {
	if _, err := os.Stat(filepath.Join(dir, bindingFile)); err != nil {
		return fmt.Errorf("go bindings (run Generate first): %w", err)
	}

	var curve string
	switch p.Curve {
	case ecc.BN254:
		curve = "bn254"
	case ecc.BLS12_381:
		curve = "bls12-381"
	default:
		return fmt.Errorf("unsupported curve %s", p.Curve)
	}

	var tmpl *template.Template
	switch p.Backend {
	case backend.GROTH16:
		tmpl = groth16Main
	case backend.PLONK:
		tmpl = plonkMain
	default:
		return fmt.Errorf("unsupported backend %s", p.Backend)
	}

	if len(p.PublicInputs)%frBytes != 0 {
		return errors.New("public inputs length is not a multiple of the field element size")
	}

	data := struct {
		Proof          string
		PublicInputs   string
		NbPublicInputs int
		NbCommitments  int
		Curve          string
	}{
		Proof:          hex.EncodeToString(p.Proof),
		PublicInputs:   hex.EncodeToString(p.PublicInputs),
		NbPublicInputs: len(p.PublicInputs) / frBytes,
		NbCommitments:  p.NbCommitments,
		Curve:          curve,
	}
	if err := executeTemplate(tmpl, filepath.Join(dir, "main.go"), data); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(tmplGoMod), 0o600); err != nil {
		return err
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	if err := run(tidy); err != nil {
		return err
	}
	goRun := exec.Command("go", "run", "main.go", bindingFile)
	goRun.Dir = dir
	return run(goRun)
}

func executeTemplate(tmpl *template.Template, path string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(f, data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func run(cmd *exec.Cmd) error {
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w\n%s", cmd, err, out)
	}
	return nil
}
