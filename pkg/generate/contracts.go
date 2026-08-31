package generate

// APIContracts is the top-level output written to api_contracts.yaml.
type APIContracts struct {
	BasePath  string                `yaml:"base_path"`
	Auth      string                `yaml:"auth,omitempty"`
	OrgScope  string                `yaml:"org_scope,omitempty"`
	Endpoints map[string][]Endpoint `yaml:"endpoints"`
}

// Endpoint represents a single API endpoint in the contracts summary.
type Endpoint struct {
	Method   string   `yaml:"method"`
	Path     string   `yaml:"path"`
	Auth     *bool    `yaml:"auth,omitempty"`
	Body     string   `yaml:"body,omitempty"`
	Response string   `yaml:"response,omitempty"`
	Roles    []string `yaml:"roles,omitempty,flow"`
	Filters  []string `yaml:"filters,omitempty,flow"`
}
