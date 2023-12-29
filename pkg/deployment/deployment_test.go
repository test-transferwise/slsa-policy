package deployment

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/laurentsimon/slsa-policy/pkg/deployment/internal/common"
	"github.com/laurentsimon/slsa-policy/pkg/deployment/internal/organization"
	"github.com/laurentsimon/slsa-policy/pkg/deployment/internal/project"
	"github.com/laurentsimon/slsa-policy/pkg/errs"
	"github.com/laurentsimon/slsa-policy/pkg/utils/intoto"
)

func Test_AttestationNew(t *testing.T) {
	t.Parallel()
	digests := intoto.DigestSet{
		"sha256":    "some_value",
		"gitCommit": "another_value",
	}
	subject := intoto.Subject{
		Digests: digests,
	}
	creatorID := "creator_id"
	creatorVersion := "creator_version"
	policy := map[string]intoto.Policy{
		"org": intoto.Policy{
			URI: "policy1_uri",
			Digests: intoto.DigestSet{
				"sha256":    "value1",
				"commitSha": "value2",
			},
		},
		"project": intoto.Policy{
			URI: "policy2_uri",
			Digests: intoto.DigestSet{
				"sha256":    "value3",
				"commitSha": "value4",
			},
		},
	}
	principal := project.Principal{
		URI: "principal_uri",
	}
	result := PolicyEvaluationResult{
		digests:   digests,
		principal: &principal,
	}
	opts := []AttestationCreationOption{
		SetCreatorVersion(creatorVersion),
		SetPolicy(policy),
	}
	tests := []struct {
		name           string
		creatorID      string
		result         PolicyEvaluationResult
		options        []AttestationCreationOption
		subject        intoto.Subject
		creatorVersion string
		policy         map[string]intoto.Policy
		expected       error
	}{
		{
			name:           "all fields set",
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			result:         result,
			options:        opts,
			subject:        subject,
			policy:         policy,
		},
		{
			name:      "no creator version",
			creatorID: creatorID,
			result:    result,
			options: []AttestationCreationOption{
				SetPolicy(policy),
			},
			subject: subject,
			policy:  policy,
		},
		{
			name:           "no policy",
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			result:         result,
			options: []AttestationCreationOption{
				SetCreatorVersion(creatorVersion),
			},
			subject: subject,
		},
		{
			name:           "error result",
			expected:       errs.ErrorInternal,
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			result: PolicyEvaluationResult{
				err: errs.ErrorMismatch,
			},
			options: opts,
			subject: subject,
			policy:  policy,
		},
		{
			name:      "invalid result",
			creatorID: creatorID,
			expected:  errs.ErrorInternal,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			att, err := tt.result.AttestationNew(tt.creatorID, tt.options...)
			if diff := cmp.Diff(tt.expected, err, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(statementType, att.Header.Type); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(predicateType, att.Header.PredicateType); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(tt.creatorID, att.attestation.Predicate.Creator.ID); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff([]intoto.Subject{tt.subject}, att.attestation.Header.Subjects); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			c := map[string]string{
				contextPrincipal: tt.result.principal.URI,
			}
			if diff := cmp.Diff(c, att.attestation.Predicate.Context); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(contextTypePrincipal, att.attestation.Predicate.ContextType); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(tt.creatorVersion, att.attestation.Predicate.Creator.Version); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(tt.policy, att.attestation.Predicate.Policy); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
		})
	}
}

func Test_e2e(t *testing.T) {
	t.Parallel()
	digests := intoto.DigestSet{
		"sha256": "val256",
		"sha512": "val512",
	}
	policy := map[string]intoto.Policy{
		"org": intoto.Policy{
			URI: "policy1_uri",
			Digests: intoto.DigestSet{
				"sha256":    "value1",
				"commitSha": "value2",
			},
		},
		"project": intoto.Policy{
			URI: "policy2_uri",
			Digests: intoto.DigestSet{
				"sha256":    "value3",
				"commitSha": "value4",
			},
		},
	}
	releaserID1 := "releaser_id1"
	releaserID2 := "releaser_id2"
	packageURI1 := "package_uri1"
	packageURI2 := "package_uri2"
	packageURI3 := "package_uri3"
	packageURI4 := "package_uri4"
	pricipalURI1 := "principal_uri1"
	pricipalURI2 := "principal_uri2"
	// NOTE: the test iterator indexes policies starting at 0.
	policyID2 := "policy_id1"
	org := organization.Policy{
		Format: 1,
		Roots: organization.Roots{
			Release: []organization.Root{
				{
					ID: releaserID1,
					Build: organization.Build{
						MaxSlsaLevel: common.AsPointer(2),
					},
				},
				{
					ID: releaserID2,
					Build: organization.Build{
						MaxSlsaLevel: common.AsPointer(3),
					},
				},
			},
		},
	}
	projects := []project.Policy{
		{
			Format: 1,
			BuildRequirements: project.BuildRequirements{
				RequireSlsaLevel: common.AsPointer(2),
			},
			Principal: project.Principal{
				URI: pricipalURI1,
			},
			Packages: []project.Package{
				{
					URI: packageURI3,
					Environment: project.Environment{
						AnyOf: []string{"dev", "prod"},
					},
				},
				{
					URI: packageURI4,
					Environment: project.Environment{
						AnyOf: []string{"dev", "prod"},
					},
				},
			},
		},
		{
			Format: 1,
			Principal: project.Principal{
				URI: pricipalURI2,
			},
			BuildRequirements: project.BuildRequirements{
				RequireSlsaLevel: common.AsPointer(3),
			},
			Packages: []project.Package{
				{
					URI: packageURI1,
					Environment: project.Environment{
						AnyOf: []string{"dev", "prod"},
					},
				},
				{
					URI: packageURI2,
					Environment: project.Environment{
						AnyOf: []string{"dev", "prod"},
					},
				},
			},
		},
	}
	creatorID := "creator_id"
	creatorVersion := "creator_version"
	opts := []AttestationCreationOption{
		SetCreatorVersion(creatorVersion),
		SetPolicy(policy),
	}
	tests := []struct {
		name             string
		org              organization.Policy
		projects         []project.Policy
		packageURI       string
		digests          intoto.DigestSet
		options          []AttestationCreationOption
		policyID         string
		policy           map[string]intoto.Policy
		env              string
		releaserID       string
		creatorID        string
		creatorVersion   string
		principalURI     string
		expected         error
		errorEvaluate    error
		errorAttestation error
		errorVerify      error
	}{
		{
			name: "all fields set",
			// Policies to evaluate.
			org:      org,
			projects: projects,
			policyID: policyID2,
			// Options to create the attestation.
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			options:        opts,
			env:            "prod",
			// Fields to validate the created attestation.
			digests:      digests,
			packageURI:   packageURI1,
			policy:       policy,
			principalURI: pricipalURI2,
			// Releaser that the verifier will use.
			releaserID: releaserID2,
		},
		{
			name: "env not provided",
			// Policies to evaluate.
			org:      org,
			projects: projects,
			policyID: policyID2,
			// Options to create the attestation.
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			options:        opts,
			// Fields to validate the created attestation.
			digests:      digests,
			packageURI:   packageURI1,
			policy:       policy,
			principalURI: pricipalURI2,
			// Releaser that the verifier will use.
			releaserID:       releaserID2,
			errorEvaluate:    errs.ErrorVerification,
			errorAttestation: errs.ErrorInternal,
		},
		{
			name: "env not in policy",
			// Policies to evaluate.
			org: org,
			projects: []project.Policy{
				{
					Format: 1,
					BuildRequirements: project.BuildRequirements{
						RequireSlsaLevel: common.AsPointer(2),
					},
					Principal: project.Principal{
						URI: pricipalURI1,
					},
					Packages: []project.Package{
						{
							URI: packageURI3,
							Environment: project.Environment{
								AnyOf: []string{"dev", "prod"},
							},
						},
						{
							URI: packageURI4,
							Environment: project.Environment{
								AnyOf: []string{"dev", "prod"},
							},
						},
					},
				},
				{
					Format: 1,
					Principal: project.Principal{
						URI: pricipalURI2,
					},
					BuildRequirements: project.BuildRequirements{
						RequireSlsaLevel: common.AsPointer(3),
					},
					Packages: []project.Package{
						{
							URI: packageURI1,
							// NOTE: no env set.
						},
						{
							URI: packageURI2,
							Environment: project.Environment{
								AnyOf: []string{"dev", "prod"},
							},
						},
					},
				},
			},
			policyID: policyID2,
			// Options to create the attestation.
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			options:        opts,
			env:            "prod",
			// Fields to validate the created attestation.
			digests:      digests,
			packageURI:   packageURI1,
			policy:       policy,
			principalURI: pricipalURI2,
			// Releaser that the verifier will use.
			releaserID:       releaserID2,
			errorEvaluate:    errs.ErrorVerification,
			errorAttestation: errs.ErrorInternal,
		},
		{
			name: "mismatch env",
			// Policies to evaluate.
			org:      org,
			projects: projects,
			policyID: policyID2,
			// Options to create the attestation.
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			options:        opts,
			env:            "mismatch",
			// Fields to validate the created attestation.
			digests:      digests,
			packageURI:   packageURI1,
			policy:       policy,
			principalURI: pricipalURI2,
			// Releaser that the verifier will use.
			releaserID:       releaserID2,
			errorEvaluate:    errs.ErrorVerification,
			errorAttestation: errs.ErrorInternal,
		},
		{
			name: "no env",
			// Policies to evaluate.
			org: org,
			projects: []project.Policy{
				{
					Format: 1,
					BuildRequirements: project.BuildRequirements{
						RequireSlsaLevel: common.AsPointer(2),
					},
					Principal: project.Principal{
						URI: pricipalURI1,
					},
					Packages: []project.Package{
						{
							URI: packageURI3,
							Environment: project.Environment{
								AnyOf: []string{"dev", "prod"},
							},
						},
						{
							URI: packageURI4,
							Environment: project.Environment{
								AnyOf: []string{"dev", "prod"},
							},
						},
					},
				},
				{
					Format: 1,
					Principal: project.Principal{
						URI: pricipalURI2,
					},
					BuildRequirements: project.BuildRequirements{
						RequireSlsaLevel: common.AsPointer(3),
					},
					Packages: []project.Package{
						{
							URI: packageURI1,
							// NOTE: no env set.
						},
						{
							URI: packageURI2,
							Environment: project.Environment{
								AnyOf: []string{"dev", "prod"},
							},
						},
					},
				},
			},
			policyID: policyID2,
			// Options to create the attestation.
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			options:        opts,
			// Fields to validate the created attestation.
			digests:      digests,
			packageURI:   packageURI1,
			policy:       policy,
			principalURI: pricipalURI2,
			// Releaser that the verifier will use.
			releaserID: releaserID2,
		},
		{
			name: "no author version",
			// Policies to evaluate.
			org:      org,
			projects: projects,
			policyID: policyID2,
			// Options to create the attestation.
			creatorID: creatorID,
			options:   opts,
			env:       "prod",
			// Fields to validate the created attestation.
			digests:      digests,
			packageURI:   packageURI1,
			policy:       policy,
			principalURI: pricipalURI2,
			// Releaser that the verifier will use.
			releaserID: releaserID2,
		},
		{
			name: "no policy",
			// Policies to evaluate.
			org:      org,
			projects: projects,
			policyID: policyID2,
			// Options to create the attestation.
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			options:        opts,
			env:            "prod",
			// Fields to validate the created attestation.
			digests:      digests,
			packageURI:   packageURI1,
			principalURI: pricipalURI2,
			// Releaser that the verifier will use.
			releaserID: releaserID2,
		},
		{
			name: "evaluation error",
			// Policies to evaluate.
			org:      org,
			projects: projects,
			policyID: policyID2,
			// Options to create the attestation.
			creatorID:      creatorID,
			creatorVersion: creatorVersion,
			options:        opts,
			env:            "prod",
			// Fields to validate the created attestation.
			digests:      digests,
			packageURI:   packageURI1,
			policy:       policy,
			principalURI: pricipalURI2,
			// Releaser that the verifier will use.
			releaserID:       releaserID1, // NOTE: mismatch releaser ID.
			errorEvaluate:    errs.ErrorVerification,
			errorAttestation: errs.ErrorInternal,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create the reader for the org policy.
			orgContent, err := json.Marshal(tt.org)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			orgReader := io.NopCloser(bytes.NewReader(orgContent))
			// Create the readers for the projects policy.
			// Marshal the project policies into bytes.
			policies := make([][]byte, len(tt.projects), len(tt.projects))
			for i := range tt.projects {
				content, err := json.Marshal(tt.projects[i])
				if err != nil {
					t.Fatalf("failed to marshal: %v", err)
				}
				policies[i] = content
			}
			projectsReader := common.NewNamedBytesIterator(policies, true)
			pol, err := PolicyNew(orgReader, projectsReader)
			if err != nil {
				t.Fatalf("failed to create policy: %v", err)
			}
			verifier := common.NewAttestationVerifier(tt.digests, tt.packageURI, tt.env, tt.releaserID)
			opts := ReleaseVerificationOption{
				Verifier: verifier,
			}
			result := pol.Evaluate(tt.digests, tt.packageURI, tt.policyID, opts)
			if diff := cmp.Diff(tt.errorEvaluate, result.Error(), cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if err != nil {
				return
			}
			att, err := result.AttestationNew(tt.creatorID, tt.options...)
			if diff := cmp.Diff(tt.errorAttestation, err, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if err != nil {
				return
			}
			attBytes, err := att.ToBytes()
			if err != nil {
				t.Fatalf("failed to get attestation bytes: %v\n", err)
			}
			verifReader := io.NopCloser(bytes.NewReader(attBytes))
			verification, err := VerificationNew(verifReader)
			if err != nil {
				t.Fatalf("failed to creation verification: %v", err)
			}

			// Create verification options.
			options := []AttestationVerificationOption{}
			if tt.creatorVersion != "" {
				options = append(options, IsCreatorVersion(tt.creatorVersion))
			}
			for name, policy := range tt.policy {
				options = append(options, HasPolicy(name, policy.URI, policy.Digests))
			}
			// Verify.
			context := map[string]string{
				contextPrincipal: tt.principalURI,
			}
			err = verification.Verify(tt.creatorID, tt.digests, contextTypePrincipal, context, options...)
			if diff := cmp.Diff(tt.errorVerify, err, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
		})
	}
}