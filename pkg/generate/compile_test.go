package generate_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kalo-build/go-util/assertfile"
	"github.com/kalo-build/plugin-openapi-ai-context/pkg/generate"
	"github.com/stretchr/testify/suite"
)

type CompileTestSuite struct {
	assertfile.FileSuite

	TestDataPath    string
	GroundTruthPath string
}

func TestCompileTestSuite(t *testing.T) {
	suite.Run(t, new(CompileTestSuite))
}

func (suite *CompileTestSuite) SetupTest() {
	_, filename, _, _ := runtime.Caller(0)
	suite.TestDataPath = filepath.Join(filepath.Dir(filename), "..", "..", "testdata")
	suite.GroundTruthPath = filepath.Join(suite.TestDataPath, "ground-truth")
}

func (suite *CompileTestSuite) TestGenerateAIContext() {
	inputDir := filepath.Join(suite.TestDataPath, "input")
	outputDir := suite.T().TempDir()

	cfg := generate.Config{
		InputDir:     inputDir,
		OutputDir:    outputDir,
		SpecFileName: "openapi.yaml",
	}

	err := generate.GenerateAIContext(cfg)
	suite.NoError(err)

	actualPath := filepath.Join(outputDir, "api_contracts.yaml")
	gtPath := filepath.Join(suite.GroundTruthPath, "api_contracts.yaml")
	suite.FileExists(actualPath)
	suite.FileEquals(actualPath, gtPath)
}
