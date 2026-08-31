package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kalo-build/plugin-openapi-ai-context/pkg/generate"
)

type StoreConfig struct {
	ID        uint32 `json:"id"`
	Type      string `json:"type"`
	MountPath string `json:"mountPath,omitempty"`
}

type PluginConfig struct {
	Stores     map[string]StoreConfig `json:"stores,omitempty"`
	InputPath  string                 `json:"inputPath,omitempty"`
	OutputPath string                 `json:"outputPath,omitempty"`
	Config     PluginConfigFields     `json:"config"`
	Verbose    bool                   `json:"verbose,omitempty"`
}

type PluginConfigFields struct {
	SpecFileName string `json:"specFileName,omitempty"`
}

const (
	ErrMissingConfig      = 3
	ErrInvalidConfig      = 4
	ErrInputPathRequired  = 12
	ErrOutputPathRequired = 13
	ErrGenerateFailed     = 1
)

func logInfo(verbose bool, format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: plugin-openapi-ai-context <config>")
		os.Exit(ErrMissingConfig)
	}

	rawConfig := os.Args[1]
	var pluginConfig PluginConfig
	if err := json.Unmarshal([]byte(rawConfig), &pluginConfig); err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing config JSON:", err)
		os.Exit(ErrInvalidConfig)
	}

	var inputPath, outputPath string

	if pluginConfig.Stores != nil {
		for _, store := range pluginConfig.Stores {
			switch store.MountPath {
			case "/input":
				inputPath = "/input"
			case "/output":
				outputPath = "/output"
			}
		}
	}

	if inputPath == "" && pluginConfig.InputPath != "" {
		inputPath = pluginConfig.InputPath
	}
	if outputPath == "" && pluginConfig.OutputPath != "" {
		outputPath = pluginConfig.OutputPath
	}

	if inputPath == "" {
		fmt.Fprintln(os.Stderr, "Error: Input path is required (directory containing OpenAPI spec)")
		os.Exit(ErrInputPathRequired)
	}
	if outputPath == "" {
		fmt.Fprintln(os.Stderr, "Error: Output path is required")
		os.Exit(ErrOutputPathRequired)
	}

	if inputPath[0] != '/' {
		if abs, err := filepath.Abs(inputPath); err == nil {
			inputPath = abs
		}
	}
	if outputPath[0] != '/' {
		if abs, err := filepath.Abs(outputPath); err == nil {
			outputPath = abs
		}
	}

	cfg := generate.Config{
		InputDir:     inputPath,
		OutputDir:    outputPath,
		SpecFileName: pluginConfig.Config.SpecFileName,
	}
	cfg.Resolve()

	logInfo(pluginConfig.Verbose, "Reading OpenAPI spec from: '%s'", filepath.Join(cfg.InputDir, cfg.SpecFileName))
	logInfo(pluginConfig.Verbose, "Generating AI context to: '%s'", cfg.OutputDir)

	if err := generate.GenerateAIContext(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "AI context generation failed:", err)
		os.Exit(ErrGenerateFailed)
	}

	logInfo(pluginConfig.Verbose, "AI context generated successfully")
	os.Exit(0)
}
