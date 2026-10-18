package gkrcore

import "github.com/consensys/gnark-crypto/hash"

// HashTranscript is a Fiat-Shamir transcript backed by a running field hasher. Field elements are
// written via Bind; challenges are derived via Challenge. The hasher is never reset — all previous
// data is implicitly part of future challenges.
type HashTranscript[E any] struct {
	h     hash.FieldHasher[E, E]
	bound bool // whether Bind was called since the last Challenge
}

// NewHashTranscript returns a transcript backed by h. Bind writes the elements with one WriteField call.
// Challenge writes the zero element if nothing was bound since the last challenge, then returns
// h.SumField().
func NewHashTranscript[E any](h hash.FieldHasher[E, E]) *HashTranscript[E] {
	return &HashTranscript[E]{h: h}
}

// Bind writes field elements to the transcript as bindings for the next challenge.
func (t *HashTranscript[E]) Bind(elements ...E) {
	if len(elements) == 0 {
		return
	}
	t.h.WriteField(elements...)
	t.bound = true
}

// Challenge binds elements as Bind would, then squeezes a challenge from the current hasher state.
// If no bindings were added since the last challenge, the zero element is written first to advance
// the state and prevent repeated values.
func (t *HashTranscript[E]) Challenge(elements ...E) E {
	t.Bind(elements...)
	if !t.bound {
		var zero E
		t.h.WriteField(zero)
	}
	t.bound = false
	return t.h.SumField()
}
