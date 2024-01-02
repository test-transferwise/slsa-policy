package common

import (
	"bytes"
	"fmt"
	"io"

	"github.com/laurentsimon/slsa-policy/pkg/errs"
	"github.com/laurentsimon/slsa-policy/pkg/release/internal/options"
	"github.com/laurentsimon/slsa-policy/pkg/utils/intoto"
	"github.com/laurentsimon/slsa-policy/pkg/utils/iterator"
)

func AsPointer[K interface{}](o K) *K {
	return &o
}

// Bytes iterator.
func NewBytesIterator(values [][]byte) iterator.ReadCloserIterator {
	return &bytesIterator{values: values, index: -1}
}

type bytesIterator struct {
	values [][]byte
	index  int
	err    error
}

func (iter *bytesIterator) Next() io.ReadCloser {
	if iter.err != nil {
		return nil
	}
	iter.index++
	return io.NopCloser(bytes.NewReader(iter.values[iter.index]))
}

func (iter *bytesIterator) HasNext() bool {
	if iter.err != nil {
		return false
	}
	return iter.index+1 < len(iter.values)
}

func (iter *bytesIterator) Error() error {
	return nil
}

// Attestation verifier.
func NewAttestationVerifier(digests intoto.DigestSet, packageName, builderID, sourceName string) options.AttestationVerifier {
	return &attesationVerifier{packageName: packageName,
		builderID: builderID, sourceName: sourceName,
		digests: digests}
}

type attesationVerifier struct {
	packageName string
	builderID  string
	sourceName  string
	digests    intoto.DigestSet
}

func (v *attesationVerifier) VerifyBuildAttestation(digests intoto.DigestSet, packageName, builderID, sourceName string) error {
	if packageName == v.packageName && builderID == v.builderID && sourceName == v.sourceName && mapEq(digests, v.digests) {
		return nil
	}
	return fmt.Errorf("%w: cannot verify package Name (%q) builder ID (%q) source Name (%q) digests (%q)",
		errs.ErrorVerification, packageName, builderID, sourceName, digests)
}

func mapEq(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v := range m1 {
		vv, exists := m2[k]
		if !exists {
			return false
		}
		if vv != v {
			return false
		}
	}
	return true
}
