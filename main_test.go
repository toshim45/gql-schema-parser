package main_test

import (
	"fmt"
	"os"
	"testing"

	main "github.com/toshim45/gqlsch"
)

const (
	PREFIX_PATH = "/Users/bytedance/Documents/Sources/gtl-core-ui"
)

func TestMerge(t *testing.T) {
	t.Log("---start---")
	t.Log("---done---")
}

func TestGetImportFromDir(t *testing.T) {
	t.Log("---start---")
	dirPath := getPrefixPath() + "/wms-ui-v2/src/ui/pages"
	results := main.GetImportFromDir(dirPath)
	if len(results) == 0 {
		t.Error("no result from: " + dirPath)
	}
	for _, result := range results {
		fmt.Printf("Name: %s, FromPath: %s\n", result.Name, result.FromPath)
	}
	t.Log("---done---")
}

func TestGetEligiblePage(t *testing.T) {
	t.Log("---start---")
	dirPath := getPrefixPath() + "/wms-ui-v2/src/ui/pages"
	results := main.GetEligiblePage(dirPath)
	for dp := range results {
		fmt.Printf("Directory Path: %s\n", dp)
	}
	if len(results) == 0 {
		t.Error("no result from: " + dirPath)
	}
	t.Log("---done---")
}

func TestGetGQLImport(t *testing.T) {
	t.Log("---start---")
	dirPath := getPrefixPath() + "/packages/hooks/mutations/" + getSuffixPath()
	results := main.GetGQLImport(dirPath)
	for _, imp := range results {
		fmt.Printf("%s.ts\n", imp)
	}
	if len(results) == 0 {
		t.Error("no result from: " + dirPath)
	}
	t.Log("---done---")
}

func getSuffixPath() string {
	//TODO: this function only required for TestGetGQLImport
	panic("unimplemented")
}

func TestParser(t *testing.T) {
	t.Log("---start---")
	schemaFilePath := "mini.graphql"
	schemaType := "InboundV3Input"
	schema := main.LoadSchema(schemaFilePath)
	def := schema.Types[schemaType]
	t.Log("def: ", def.Name, def.Kind)
	t.Log("---done---")
}

func getPrefixPath() string {
	envPrefixPath := os.Getenv("PREFIX_PATH")
	if envPrefixPath == "" {
		return PREFIX_PATH
	}

	return envPrefixPath
}
