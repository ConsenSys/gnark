/*
Copyright © 2020 ConsenSys

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package frontend

import (
	"fmt"
	"math/big"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/internal/backend/untyped"
	"github.com/consensys/gurvy"

	bls377r1cs "github.com/consensys/gnark/internal/backend/bls377/cs"
	bls381r1cs "github.com/consensys/gnark/internal/backend/bls381/cs"
	bn256r1cs "github.com/consensys/gnark/internal/backend/bn256/cs"
	bw761r1cs "github.com/consensys/gnark/internal/backend/bw761/cs"
)

// ConstraintSystem represents a Groth16 like circuit
//
// All the APIs to define a circuit (see Circuit.Define) like Add, Sub, Mul, ...
// may take as input interface{}
//
// these interfaces are either Variables (/LinearExpressions) or constants (big.Int, strings, uint, fr.Element)
type ConstraintSystem struct {
	// Variables (aka wires)
	public struct {
		variables []Variable       // public inputs
		booleans  map[int]struct{} // keep track of boolean variables (we constrain them once)
	}
	secret struct {
		variables []Variable       // secret inputs
		booleans  map[int]struct{} // keep track of boolean variables (we constrain them once)
	}
	internal struct {
		variables []Variable       // internal variables
		booleans  map[int]struct{} // keep track of boolean variables (we constrain them once)
	}

	// Constraints
	constraints []backend.R1C // list of R1C that yield an output (for example v3 == v1 * v2, return v3)
	assertions  []backend.R1C // list of R1C that yield no output (for example ensuring v1 == v2)

	// Coefficients in the constraints
	coeffs    []big.Int      // list of unique coefficients.
	coeffsIDs map[string]int // map to fast check existence of a coefficient (key = coeff.Text(16))

	// debug info
	logs           []logEntry // list of logs to be printed when solving a circuit. The logs are called with the method Println
	debugInfo      []logEntry // list of logs storing information about assertions. If an assertion fails, it prints it in a friendly format
	unsetVariables []logEntry // unset variables. If a variable is unset, the error is caught when compiling the circuit

}

func (cs *ConstraintSystem) buildVarFromPartialVar(pv Wire) Variable {
	return Variable{pv, cs.LinearExpression(cs.makeTerm(pv, bOne)), false}
}

// this has quite some impact on frontend performance, especially on large circuits size
// we may want to add build tags to tune that
const initialCapacity = 1e6

func newConstraintSystem() ConstraintSystem {
	cs := ConstraintSystem{
		coeffs:      make([]big.Int, 0),
		coeffsIDs:   make(map[string]int),
		constraints: make([]backend.R1C, 0, initialCapacity),
		assertions:  make([]backend.R1C, 0),
	}

	cs.public.variables = make([]Variable, 0)
	cs.public.booleans = make(map[int]struct{})

	cs.secret.variables = make([]Variable, 0)
	cs.secret.booleans = make(map[int]struct{})

	cs.internal.variables = make([]Variable, 0, initialCapacity)
	cs.internal.booleans = make(map[int]struct{})

	// by default the circuit is given on public wire equal to 1
	cs.public.variables[0] = cs.newPublicVariable()

	return cs
}

type logEntry struct {
	format    string
	toResolve []backend.Term
}

var (
	bMinusOne = new(big.Int).SetInt64(-1)
	bZero     = new(big.Int)
	bOne      = new(big.Int).SetInt64(1)
	bTwo      = new(big.Int).SetInt64(2)
)

// debug info in case a variable is not set
func debugInfoUnsetVariable(term backend.Term) logEntry {
	entry := logEntry{}
	stack := getCallStack()
	entry.format = stack[len(stack)-1]
	entry.toResolve = append(entry.toResolve, term)
	return entry
}

func (cs *ConstraintSystem) getOneTerm() backend.Term {
	return cs.public.variables[0].linExp[0]
}

func (cs *ConstraintSystem) getOneVariable() Variable {
	return cs.public.variables[0]
}

// Term packs a variable and a coeff in a backend.Term and returns it.
func (cs *ConstraintSystem) makeTerm(v Wire, coeff *big.Int) backend.Term {

	term := backend.Pack(v.id, cs.coeffID(coeff), v.visibility)

	if coeff.Cmp(bZero) == 0 {
		term.SetCoeffValue(0)
	} else if coeff.Cmp(bOne) == 0 {
		term.SetCoeffValue(1)
	} else if coeff.Cmp(bTwo) == 0 {
		term.SetCoeffValue(2)
	} else if coeff.Cmp(bMinusOne) == 0 {
		term.SetCoeffValue(-1)
	}
	return term
}

// NbConstraints enables circuit profiling and helps debugging
// It returns the number of constraints created at the current stage of the circuit construction.
//
// The number returns included both the assertions and the non-assertion constraints
// (eg: the constraints which creates a new variable)
func (cs *ConstraintSystem) NbConstraints() int {
	return len(cs.constraints) + len(cs.assertions)
}

// LinearExpression packs a list of backend.Term in a backend.LinearExpression and returns it.
func (cs *ConstraintSystem) LinearExpression(terms ...backend.Term) backend.LinearExpression {
	res := make(backend.LinearExpression, len(terms))
	for i, args := range terms {
		res[i] = args
	}
	return res
}

// reduces redundancy in a linear expression
// Non deterministic function
func (cs *ConstraintSystem) partialReduce(linExp backend.LinearExpression, visibility backend.Visibility) backend.LinearExpression {

	if len(linExp) == 0 {
		return backend.LinearExpression{}
	}

	coeffRecord := make(map[int]big.Int) // id variable -> coeff
	varRecord := make(map[int]Wire)      // id variable -> Wire

	// the variables are collected and the coefficients are accumulated
	for _, t := range linExp {

		_, coeffID, variableID, vis := t.Unpack()

		if vis == visibility {
			tmp := Wire{vis, variableID, nil}

			if _, ok := varRecord[variableID]; !ok {
				varRecord[variableID] = tmp
				var coef, coefCopy big.Int
				coef = cs.coeffs[coeffID]
				coefCopy.Set(&coef)
				coeffRecord[variableID] = coefCopy
			} else {
				ccoef := coeffRecord[variableID]
				ccoef.Add(&ccoef, &cs.coeffs[coeffID])
				coeffRecord[variableID] = ccoef
			}
		}
	}

	// creation of the reduced linear expression
	var res backend.LinearExpression
	for k := range coeffRecord {
		bCoeff := coeffRecord[k]
		res = append(res, cs.makeTerm(varRecord[k], &bCoeff))
	}

	return res
}

// complete allocate linExp if linExp is empty. If a variable
// is created like 'var a Variable', it will be unset but Compile(..)
// will not understand it since a.linExp is empty
func (cs *ConstraintSystem) completeDanglingVariable(v *Variable) {
	if len(v.linExp) == 0 {
		tmp := Wire{backend.Unset, v.id, v.val}
		tmpVar := cs.buildVarFromPartialVar(tmp)
		cs.unsetVariables = append(cs.unsetVariables, debugInfoUnsetVariable(tmpVar.linExp[0]))
		v.linExp = tmpVar.getLinExpCopy()
	}
}

// reduces redundancy in linear expression
// Non deterministic function
func (cs *ConstraintSystem) reduce(linExp backend.LinearExpression) backend.LinearExpression {
	reducePublic := cs.partialReduce(linExp, backend.Public)
	reduceSecret := cs.partialReduce(linExp, backend.Secret)
	reduceInternal := cs.partialReduce(linExp, backend.Internal)
	reduceUnset := cs.partialReduce(linExp, backend.Unset) // we collect also the unset variables so it stays consistant (useful for debugging)
	res := make(backend.LinearExpression, len(reducePublic)+len(reduceSecret)+len(reduceInternal)+len(reduceUnset))
	accSize := 0
	copy(res[:], reducePublic)
	accSize += len(reducePublic)
	copy(res[accSize:], reduceSecret)
	accSize += len(reduceSecret)
	copy(res[accSize:], reduceInternal)
	accSize += len(reduceInternal)
	copy(res[accSize:], reduceUnset)
	return res
}

func (cs *ConstraintSystem) addAssertion(constraint backend.R1C, debugInfo logEntry) {

	cs.assertions = append(cs.assertions, constraint)
	cs.debugInfo = append(cs.debugInfo, debugInfo)
}

// toR1CS constructs a rank-1 constraint sytem
func (cs *ConstraintSystem) toR1CS(curveID gurvy.ID) (backend.ConstraintSystem, error) {

	// wires = intermediatevariables | secret inputs | public inputs

	// setting up the result
	res := untyped.R1CS{
		NbInternalWires: len(cs.internal.variables),
		NbPublicWires:   len(cs.public.variables),
		NbSecretWires:   len(cs.secret.variables),
		NbConstraints:   len(cs.constraints) + len(cs.assertions),
		NbCOConstraints: len(cs.constraints),
		Constraints:     make([]backend.R1C, len(cs.constraints)+len(cs.assertions)),
		Logs:            make([]backend.LogEntry, len(cs.logs)),
		DebugInfo:       make([]backend.LogEntry, len(cs.debugInfo)),
	}

	// computational constraints (= gates)
	copy(res.Constraints, cs.constraints)
	copy(res.Constraints[len(cs.constraints):], cs.assertions)

	// we just need to offset our ids, such that wires = [internalVariables | secretVariables | publicVariables]
	offsetIDs := func(exp backend.LinearExpression) error {
		for j := 0; j < len(exp); j++ {
			_, _, cID, cVisibility := exp[j].Unpack()
			switch cVisibility {
			case backend.Public:
				exp[j].SetVariableID(cID + len(cs.internal.variables) + len(cs.secret.variables))
			case backend.Secret:
				exp[j].SetVariableID(cID + len(cs.internal.variables))
			case backend.Unset:
				return fmt.Errorf("%w: %s", backend.ErrInputNotSet, cs.unsetVariables[0].format)
			}
		}
		return nil
	}

	var err error
	for i := 0; i < len(res.Constraints); i++ {
		err = offsetIDs(res.Constraints[i].L)
		if err != nil {
			return &res, err
		}
		err = offsetIDs(res.Constraints[i].R)
		if err != nil {
			return &res, err
		}
		err = offsetIDs(res.Constraints[i].O)
		if err != nil {
			return &res, err
		}
	}

	// we need to offset the ids in logs too
	for i := 0; i < len(cs.logs); i++ {
		entry := backend.LogEntry{
			Format: cs.logs[i].format,
		}
		for j := 0; j < len(cs.logs[i].toResolve); j++ {
			_, _, cID, cVisibility := cs.logs[i].toResolve[j].Unpack()
			switch cVisibility {
			case backend.Public:
				cID += len(cs.internal.variables) + len(cs.secret.variables)
			case backend.Secret:
				cID += len(cs.internal.variables)
			case backend.Unset:
				panic("encountered unset visibility on a variable in logs id offset routine")
			}
			entry.ToResolve = append(entry.ToResolve, cID)
		}

		res.Logs[i] = entry
	}

	// offset ids in the debugInfo
	for i := 0; i < len(cs.debugInfo); i++ {
		entry := backend.LogEntry{
			Format: cs.debugInfo[i].format,
		}
		for j := 0; j < len(cs.debugInfo[i].toResolve); j++ {
			_, _, cID, cVisibility := cs.debugInfo[i].toResolve[j].Unpack()
			switch cVisibility {
			case backend.Public:
				cID += len(cs.internal.variables) + len(cs.secret.variables)
			case backend.Secret:
				cID += len(cs.internal.variables)
			case backend.Unset:
				panic("encountered unset visibility on a variable in debugInfo id offset routine")
			}
			entry.ToResolve = append(entry.ToResolve, cID)
		}

		res.DebugInfo[i] = entry
	}

	switch curveID {
	case gurvy.BLS377:
		return bls377r1cs.NewR1CS(res, cs.coeffs), nil
	case gurvy.BLS381:
		return bls381r1cs.NewR1CS(res, cs.coeffs), nil
	case gurvy.BN256:
		return bn256r1cs.NewR1CS(res, cs.coeffs), nil
	case gurvy.BW761:
		return bw761r1cs.NewR1CS(res, cs.coeffs), nil
	case gurvy.UNKNOWN:
		return &res, nil
	default:
		panic("not implemtented")
	}
}

// coeffID tries to fetch the entry where b is if it exits, otherwise appends b to
// the list of coeffs and returns the corresponding entry
func (cs *ConstraintSystem) coeffID(b *big.Int) int {

	// if the coeff is already stored, fetch its ID from the cs.coeffsIDs map
	key := b.Text(16)
	if idx, ok := cs.coeffsIDs[key]; ok {
		return idx
	}

	// else add it in the cs.coeffs map and update the cs.coeffsIDs map
	var bCopy big.Int
	bCopy.Set(b)
	resID := len(cs.coeffs)
	cs.coeffs = append(cs.coeffs, bCopy)
	cs.coeffsIDs[key] = resID
	return resID
}

// if v is unset and linExp is non empty, the variable is allocated
// resulting in one more constraint in the system. If v is set OR v is
// unset and linexp is emppty, it does nothing.
func (cs *ConstraintSystem) allocate(v Variable) Variable {
	if v.visibility == backend.Unset && len(v.linExp) > 0 {
		iv := cs.newInternalVariable()
		one := cs.getOneVariable()
		constraint := backend.R1C{L: v.getLinExpCopy(), R: one.getLinExpCopy(), O: iv.getLinExpCopy(), Solver: backend.SingleOutput}
		cs.constraints = append(cs.constraints, constraint)
		return iv
	}
	return v
}

// Println enables circuit debugging and behaves almost like fmt.Println()
//
// the print will be done once the R1CS.Solve() method is executed
//
// if one of the input is a Variable, its value will be resolved avec R1CS.Solve() method is called
func (cs *ConstraintSystem) Println(a ...interface{}) {
	var sbb strings.Builder

	// prefix log line with file.go:line
	if _, file, line, ok := runtime.Caller(1); ok {
		sbb.WriteString(filepath.Base(file))
		sbb.WriteByte(':')
		sbb.WriteString(strconv.Itoa(line))
		sbb.WriteByte(' ')
	}

	// for each argument, if it is a circuit structure and contains variable
	// we add the variables in the logEntry.toResolve part, and add %s to the format string in the log entry
	// if it doesn't contain variable, call fmt.Sprint(arg) instead
	entry := logEntry{}

	// this is call recursively on the arguments using reflection on each argument
	foundVariable := false

	var handler logValueHandler = func(name string, tInput reflect.Value) {

		v := tInput.Interface().(Variable)

		// if the variable is only in linExp form, we allocate it
		_v := cs.allocate(v)

		entry.toResolve = append(entry.toResolve, backend.Pack(_v.id, 0, _v.visibility))

		if name == "" {
			sbb.WriteString("%s")
		} else {
			sbb.WriteString(fmt.Sprintf("%s: %%s ", name))
		}

		foundVariable = true
	}

	for i, arg := range a {
		if i > 0 {
			sbb.WriteByte(' ')
		}
		foundVariable = false
		parseLogValue(arg, "", handler)
		if !foundVariable {
			sbb.WriteString(fmt.Sprint(arg))
		}
	}
	sbb.WriteByte('\n')

	// set format string to be used with fmt.Sprintf, once the variables are solved in the R1CS.Solve() method
	entry.format = sbb.String()

	cs.logs = append(cs.logs, entry)
}

// newInternalVariable creates a new wire, appends it on the list of wires of the circuit, sets
// the wire's id to the number of wires, and returns it
func (cs *ConstraintSystem) newInternalVariable() Variable {
	resVar := Wire{
		id:         len(cs.internal.variables),
		visibility: backend.Internal,
	}
	res := cs.buildVarFromPartialVar(resVar)
	cs.internal.variables = append(cs.internal.variables, res)
	return res
}

// newPublicVariable creates a new public input
func (cs *ConstraintSystem) newPublicVariable() Variable {

	idx := len(cs.public.variables)
	resVar := Wire{backend.Public, idx, nil}

	res := cs.buildVarFromPartialVar(resVar)
	cs.public.variables = append(cs.public.variables, res)
	return res
}

// newSecretVariable creates a new secret input
func (cs *ConstraintSystem) newSecretVariable() Variable {
	idx := len(cs.secret.variables)
	resVar := Wire{backend.Secret, idx, nil}

	res := cs.buildVarFromPartialVar(resVar)
	cs.secret.variables = append(cs.secret.variables, res)
	return res
}

type logValueHandler func(name string, tValue reflect.Value)

func appendName(baseName, name string) string {
	if baseName == "" {
		return name
	}
	return baseName + "_" + name
}

func parseLogValue(input interface{}, name string, handler logValueHandler) {
	tVariable := reflect.TypeOf(Variable{})

	tValue := reflect.ValueOf(input)
	if tValue.Kind() == reflect.Ptr {
		tValue = tValue.Elem()
	}
	switch tValue.Kind() {
	case reflect.Struct:
		switch tValue.Type() {
		case tVariable:
			handler(name, tValue)
			return
		default:
			for i := 0; i < tValue.NumField(); i++ {
				value := tValue.Field(i).Interface()
				_name := appendName(name, tValue.Type().Field(i).Name)
				parseLogValue(value, _name, handler)
			}
		}
	case reflect.Slice, reflect.Array:
		if tValue.Len() == 0 {
			fmt.Println("warning, got unitizalized slice (or empty array). Ignoring;")
			return
		}
		for j := 0; j < tValue.Len(); j++ {
			value := tValue.Index(j).Interface()
			entry := "[" + strconv.Itoa(j) + "]"
			_name := appendName(name, entry)
			parseLogValue(value, _name, handler)
		}
	}
}

// derived from: https://golang.org/pkg/runtime/#example_Frames
// we stop when func name == Define as it is where the gnark circuit code should start
func getCallStack() []string {
	// Ask runtime.Callers for up to 10 pcs
	pc := make([]uintptr, 10)
	n := runtime.Callers(3, pc)
	if n == 0 {
		// No pcs available. Stop now.
		// This can happen if the first argument to runtime.Callers is large.
		return nil
	}
	pc = pc[:n] // pass only valid pcs to runtime.CallersFrames
	frames := runtime.CallersFrames(pc)
	// Loop to get frames.
	// A fixed number of pcs can expand to an indefinite number of Frames.
	var toReturn []string
	for {
		frame, more := frames.Next()
		fe := strings.Split(frame.Function, "/")
		function := fe[len(fe)-1]
		toReturn = append(toReturn, fmt.Sprintf("%s\n\t%s:%d", function, frame.File, frame.Line))
		if !more {
			break
		}
		if strings.HasSuffix(function, "Define") {
			break
		}
	}
	return toReturn
}
