package main_test

import (
	"fmt"
	"testing"

	"github.com/toshim45/gqlsch"
)

func TestMerge(t *testing.T) {
	t.Log("---start---")
	t.Log("---done---")
}

func TestGetImportFromDir(t *testing.T) {
	t.Log("---start---")
	dirPath := "../gtl-wms-core-ui/wms-ui-v2/src/ui/pages"
	results := main.GetImportFromDir(dirPath)
	for _, result := range results {
		fmt.Printf("Name: %s, FromPath: %s\n", result.Name, result.FromPath)
	}
	t.Log("---done---")
}

func TestGetEligiblePage(t *testing.T) {
	t.Log("---start---")
	dirPath := "../gtl-wms-core-ui/wms-ui-v2/src/ui/pages"
	results := main.GetEligiblePage(dirPath)
	for dp := range results {
		fmt.Printf("Directory Path: %s\n", dp)
	}
	t.Log("---done---")
}

func TestGetGQLImport(t *testing.T) {
	t.Log("---start---")
	dirPath := "../gtl-wms-core-ui/packages/hooks/mutations"
	results := main.GetGQLImport(dirPath)
	for _, imp := range results {
		fmt.Printf("%s.ts\n", imp)
	}
	t.Log("---done---")
}
