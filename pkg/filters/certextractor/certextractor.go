/*
 * Copyright (c) 2017, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package certextractor

import (
	"crypto/x509/pkix"
	"fmt"

	"github.com/megaease/easegress/pkg/context"
	"github.com/megaease/easegress/pkg/filters"
	"github.com/megaease/easegress/pkg/protocols/httpprot"
)

const (
	// Kind is the kind of CertExtractor.
	Kind = "CertExtractor"
)

var kind = &filters.Kind{
	Name:        Kind,
	Description: "CertExtractor extracts given field from TLS certificates and sets it to request headers.",
	Results:     []string{},
	DefaultSpec: func() filters.Spec {
		return &Spec{}
	},
	CreateInstance: func() filters.Filter {
		return &CertExtractor{}
	},
}

func init() {
	filters.Register(kind)
}

type (
	// CertExtractor extracts given field from TLS certificates and sets it to request headers.
	CertExtractor struct {
		spec      *Spec
		headerKey string
	}

	// Spec describes the CertExtractor.
	Spec struct {
		filters.BaseSpec `yaml:",inline"`

		CertIndex int16  `yaml:"certIndex" jsonschema:"required"`
		Target    string `yaml:"target" jsonschema:"required,enum=subject,enum=issuer"`
		// Different field options listed here https://pkg.go.dev/crypto/x509/pkix#Name
		Field     string `yaml:"field" jsonschema:"required,enum=Country,enum=Organization,enum=OrganizationalUnit,enum=Locality,enum=Province,enum=StreetAddress,enum=PostalCode,enum=SerialNumber,enum=CommonName"`
		HeaderKey string `yaml:"headerKey" jsonschema:"required"`
	}
)

// Validate is dummy as yaml rules already validate Spec.
func (spec *Spec) Validate() error { return nil }

// Name returns the name of the CertExtractor filter instance.
func (ce *CertExtractor) Name() string {
	return ce.spec.Name()
}

// Kind returns the kind of CertExtractor.
func (ce *CertExtractor) Kind() *filters.Kind {
	return kind
}

// Spec returns the spec used by the CertExtractor
func (ce *CertExtractor) Spec() filters.Spec {
	return ce.spec
}

// Init initializes CertExtractor.
func (ce *CertExtractor) Init(spec filters.Spec) {
	ce.spec = spec.(*Spec)

	ce.headerKey = fmt.Sprintf("tls-%s-%s", ce.spec.Target, ce.spec.Field)
	if ce.spec.HeaderKey != "" {
		ce.headerKey = ce.spec.HeaderKey
	}
}

// Inherit inherits previous generation of CertExtractor.
func (ce *CertExtractor) Inherit(spec filters.Spec, previousGeneration filters.Filter) {
	previousGeneration.Close()
	ce.Init(spec)
}

// Close closes CertExtractor.
func (ce *CertExtractor) Close() {}

// Handle retrieves header values and sets request headers.
func (ce *CertExtractor) Handle(ctx context.Context) string {
	return ce.handle(ctx)
}

// CertExtractor extracts given field from TLS certificates and sets it to request headers.
func (ce *CertExtractor) handle(ctx context.Context) string {
	r := ctx.Request().(*httpprot.Request)
	connectionState := r.Std().TLS
	if connectionState == nil {
		return ""
	}

	certs := connectionState.PeerCertificates
	if certs == nil || len(certs) < 1 {
		return ""
	}

	n := int16(len(certs))
	// positive ce.spec.CertIndex from the beginning, negative from the end
	relativeIndex := ce.spec.CertIndex % n
	index := (n + relativeIndex) % n
	cert := certs[index]

	var target pkix.Name
	if ce.spec.Target == "subject" {
		target = cert.Subject
	} else {
		target = cert.Issuer
	}

	var result []string
	switch ce.spec.Field {
	case "Country":
		result = target.Country
	case "Organization":
		result = target.Organization
	case "OrganizationalUnit":
		result = target.OrganizationalUnit
	case "Locality":
		result = target.Locality
	case "Province":
		result = target.Province
	case "StreetAddress":
		result = target.StreetAddress
	case "PostalCode":
		result = target.PostalCode
	case "SerialNumber":
		result = append(result, target.SerialNumber)
	case "CommonName":
		result = append(result, target.CommonName)
	}
	for _, res := range result {
		if res != "" {
			r.Header().Add(ce.headerKey, res)
		}
	}
	return ""
}

// Status returns status.
func (ce *CertExtractor) Status() interface{} { return nil }