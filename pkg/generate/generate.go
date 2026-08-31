package generate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"gopkg.in/yaml.v3"
)

// GenerateAIContext reads an OpenAPI spec, builds a condensed API contracts
// summary, and writes it to api_contracts.yaml in the output directory.
func GenerateAIContext(cfg Config) error {
	specPath := filepath.Join(cfg.InputDir, cfg.SpecFileName)
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("failed to read spec '%s': %w", specPath, err)
	}

	contracts, err := BuildContracts(specBytes)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	out, err := MarshalContracts(contracts)
	if err != nil {
		return fmt.Errorf("failed to marshal api_contracts.yaml: %w", err)
	}

	outPath := filepath.Join(cfg.OutputDir, "api_contracts.yaml")
	return os.WriteFile(outPath, out, 0o644)
}

// MarshalContracts serialises an APIContracts value to YAML bytes.
func MarshalContracts(contracts *APIContracts) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(contracts); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildContracts parses an OpenAPI spec from raw bytes and returns a
// structured APIContracts summary suitable for LLM consumption.
func BuildContracts(specBytes []byte) (*APIContracts, error) {
	document, err := libopenapi.NewDocument(specBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI document: %w", err)
	}

	model, errs := document.BuildV3Model()
	if errs != nil {
		return nil, fmt.Errorf("failed to build V3 model: %w", errs)
	}

	return convertToContracts(&model.Model), nil
}

func convertToContracts(v3doc *v3.Document) *APIContracts {
	contracts := &APIContracts{
		Endpoints: make(map[string][]Endpoint),
	}

	contracts.BasePath = extractBasePath(v3doc)
	contracts.Auth = extractAuth(v3doc)

	hasGlobalSecurity := len(v3doc.Security) > 0

	if v3doc.Paths != nil && v3doc.Paths.PathItems != nil {
		for pair := v3doc.Paths.PathItems.First(); pair != nil; pair = pair.Next() {
			pathStr := pair.Key()
			pathItem := pair.Value()
			extractPathEndpoints(contracts, pathStr, pathItem, hasGlobalSecurity)
		}
	}

	for tag := range contracts.Endpoints {
		sortEndpoints(contracts.Endpoints[tag])
	}

	return contracts
}

func extractBasePath(v3doc *v3.Document) string {
	if len(v3doc.Servers) > 0 && v3doc.Servers[0].URL != "" {
		return v3doc.Servers[0].URL
	}
	return ""
}

func extractAuth(v3doc *v3.Document) string {
	if v3doc.Components == nil || v3doc.Components.SecuritySchemes == nil {
		return ""
	}
	for pair := v3doc.Components.SecuritySchemes.First(); pair != nil; pair = pair.Next() {
		scheme := pair.Value()
		if scheme.Type == "http" && scheme.Scheme == "bearer" {
			if scheme.BearerFormat != "" {
				return "Bearer " + scheme.BearerFormat
			}
			return "Bearer token"
		}
		if scheme.Type == "apiKey" {
			return fmt.Sprintf("API key via %s %s", scheme.In, scheme.Name)
		}
	}
	return ""
}

func extractPathEndpoints(contracts *APIContracts, pathStr string, item *v3.PathItem, hasGlobalSecurity bool) {
	type methodOp struct {
		method string
		op     *v3.Operation
	}
	ops := []methodOp{
		{"GET", item.Get},
		{"POST", item.Post},
		{"PUT", item.Put},
		{"PATCH", item.Patch},
		{"DELETE", item.Delete},
	}

	for _, mo := range ops {
		if mo.op == nil {
			continue
		}

		tag := "default"
		if len(mo.op.Tags) > 0 {
			tag = mo.op.Tags[0]
		}

		ep := Endpoint{
			Method: mo.method,
			Path:   pathStr,
		}

		if hasGlobalSecurity && isAuthDisabled(mo.op) {
			authFalse := false
			ep.Auth = &authFalse
		}

		if mo.op.RequestBody != nil {
			ep.Body = extractRequestBodyName(mo.op.RequestBody)
		}

		ep.Response = extractResponseName(mo.op.Responses)

		for _, p := range mo.op.Parameters {
			if p.In == "query" {
				ep.Filters = append(ep.Filters, p.Name)
			}
		}

		contracts.Endpoints[tag] = append(contracts.Endpoints[tag], ep)
	}
}

// isAuthDisabled returns true when an operation explicitly sets an empty
// security array, opting out of global authentication.
func isAuthDisabled(op *v3.Operation) bool {
	if op.Security == nil {
		return false
	}
	return len(op.Security) == 0
}

func extractRequestBodyName(rb *v3.RequestBody) string {
	if rb.Content == nil {
		return ""
	}
	jsonMT, ok := rb.Content.Get("application/json")
	if !ok || jsonMT == nil || jsonMT.Schema == nil {
		return ""
	}
	ref := jsonMT.Schema.GetReference()
	if ref != "" {
		return refToName(ref)
	}
	return ""
}

func extractResponseName(responses *v3.Responses) string {
	if responses == nil || responses.Codes == nil {
		return ""
	}
	for _, code := range []string{"200", "201", "202"} {
		resp, ok := responses.Codes.Get(code)
		if !ok || resp == nil || resp.Content == nil {
			continue
		}
		jsonMT, ok := resp.Content.Get("application/json")
		if !ok || jsonMT == nil || jsonMT.Schema == nil {
			continue
		}
		ref := jsonMT.Schema.GetReference()
		if ref != "" {
			return refToName(ref)
		}
		resolved := jsonMT.Schema.Schema()
		if resolved == nil {
			continue
		}
		if name := tryArrayResponse(resolved); name != "" {
			return name
		}
		if name := tryWrappedArrayResponse(resolved); name != "" {
			return name
		}
	}
	return ""
}

// tryArrayResponse handles a top-level array schema: type: array, items.$ref.
func tryArrayResponse(s *base.Schema) string {
	if !sliceContains(s.Type, "array") {
		return ""
	}
	if s.Items == nil || s.Items.A == nil {
		return ""
	}
	itemRef := s.Items.A.GetReference()
	if itemRef != "" {
		return refToName(itemRef) + "[]"
	}
	return ""
}

// tryWrappedArrayResponse handles { type: object, properties: { data: { type: array, items.$ref } } }.
func tryWrappedArrayResponse(s *base.Schema) string {
	if !sliceContains(s.Type, "object") || s.Properties == nil {
		return ""
	}
	dataProxy, ok := s.Properties.Get("data")
	if !ok || dataProxy == nil {
		return ""
	}
	dataSchema := dataProxy.Schema()
	if dataSchema == nil || !sliceContains(dataSchema.Type, "array") {
		return ""
	}
	if dataSchema.Items == nil || dataSchema.Items.A == nil {
		return ""
	}
	itemRef := dataSchema.Items.A.GetReference()
	if itemRef != "" {
		return refToName(itemRef) + "[]"
	}
	return ""
}

func refToName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

func sliceContains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

func sortEndpoints(eps []Endpoint) {
	sort.SliceStable(eps, func(i, j int) bool {
		if eps[i].Path != eps[j].Path {
			return eps[i].Path < eps[j].Path
		}
		return methodRank(eps[i].Method) < methodRank(eps[j].Method)
	})
}

func methodRank(method string) int {
	switch method {
	case "GET":
		return 0
	case "POST":
		return 1
	case "PUT":
		return 2
	case "PATCH":
		return 3
	case "DELETE":
		return 4
	default:
		return 5
	}
}
