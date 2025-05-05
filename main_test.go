package main_test

import (
	"fmt"
	"testing"

	"github.com/toshim45/gqlsch"
)

const (
	PREFIX_PATH = "/Users/admin/Documents/gtlsource/gtl-wms-core-ui"
)

func TestMerge(t *testing.T) {
	t.Log("---start---")
	t.Log("---done---")
}

func TestGetImportFromDir(t *testing.T) {
	t.Log("---start---")
	dirPath := PREFIX_PATH + "/wms-ui-v2/src/ui/pages"
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
	dirPath := PREFIX_PATH + "/wms-ui-v2/src/ui/pages"
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
	dirPath := PREFIX_PATH + "/packages/hooks/mutations/warehouse"
	results := main.GetGQLImport(dirPath)
	for _, imp := range results {
		fmt.Printf("%s.ts\n", imp)
	}
	if len(results) == 0 {
		t.Error("no result from: " + dirPath)
	}
	t.Log("---done---")
}
