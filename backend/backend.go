// Copyright 2020 ConsenSys AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package backend provides circuit arithmetic representations and zero knowledge proof APIs.
//
// Functions is this package are not curve specific and point to internal/backend curve specific
// implementation when needed
package backend

// ID represent a unique ID for a proving scheme
type ID uint16

const (
	UNKNOWN ID = iota
	GROTH16
	PLONK
)
