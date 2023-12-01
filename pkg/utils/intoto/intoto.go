package intoto

import (
	"fmt"
	"time"

	"github.com/laurentsimon/slsa-policy/pkg/errs"
)

type DigestSet map[string]string

type Subject ResourceDescriptor

type Header struct {
	Type          string    `json:"_type"`
	PredicateType string    `json:"predicateType"`
	Subjects      []Subject `json:"subjects"`
}

// Author is the author of the attestation.
type Author struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type Policy struct {
	URI     string    `json:"uri,omitempty"`
	Digests DigestSet `json:"digest,omitempty"`
}

type ResourceDescriptor struct {
	// A URI used to identify the resource or artifact globally. This field is
	// REQUIRED unless either digest or content is set.
	URI string `json:"uri,omitempty"`

	// A set of cryptographic digests of the contents of the resource or
	// artifact. This field is REQUIRED unless either uri or content is set.
	Digests DigestSet `json:"digest,omitempty"`

	// Machine-readable identifier for distinguishing between descriptors.
	Name string `json:"name,omitempty"`

	// The location of the described resource or artifact, if different from the
	// uri.
	DownloadLocation string `json:"downloadLocation,omitempty"`

	// The MIME Type (i.e., media type) of the described resource or artifact.
	MediaType string `json:"mediaType,omitempty"`

	// The contents of the resource or artifact. This field is REQUIRED unless
	// either uri or digest is set.
	Content []byte `json:"content,omitempty"`

	// This field MAY be used to provide additional information or metadata
	// about the resource or artifact that may be useful to the consumer when
	// evaluating the attestation against a policy.
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

func (s Subject) Validate() error {
	if s.URI == "" {
		return fmt.Errorf("%w: subject URI is empty", errs.ErrorInvalidInput)
	}
	return s.Digests.Validate()
}

func (ds DigestSet) Validate() error {
	if len(ds) == 0 {
		return fmt.Errorf("%w: digests empty", errs.ErrorInvalidInput)
	}
	for k, v := range ds {
		if k == "" {
			return fmt.Errorf("%w: digests has empty key", errs.ErrorInvalidInput)
		}
		if v == "" {
			return fmt.Errorf("%w: digests key (%q) has empty value", errs.ErrorInvalidInput, k)
		}
	}
	return nil
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
