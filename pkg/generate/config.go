package generate

// Config holds configuration for AI context generation.
type Config struct {
	InputDir     string
	OutputDir    string
	SpecFileName string
}

// Resolve applies default values for unset fields.
func (c *Config) Resolve() {
	if c.SpecFileName == "" {
		c.SpecFileName = "openapi.yaml"
	}
}
