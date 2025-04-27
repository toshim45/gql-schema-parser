package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

var opts struct {
	SourceFile string `long:"source" description:"Input source file"`
	SchemaFile string `long:"schema" description:"Input raw schema file"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		// Check the specific error type
		if flagsErr, ok := err.(*flags.Error); ok {
			// If it's a help request, print the help message and exit gracefully
			if flagsErr.Type == flags.ErrHelp {
				return
			}
		}
		// For other errors, print the error message and exit with a non-zero status
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("schema file: ", opts.SchemaFile)
	fmt.Println("source file: ", opts.SourceFile)

	// schemaFilePath := "/home/artikow/Documents/Sources/gtl-wms-graphql/hasura-20250225.graphqls"
	// gqlQueryFilePath := "/home/artikow/Documents/Sources/gql-schema-parser/raw/InboundV3Test.ts"
	gqlQuery := extractGQLFromFile(opts.SourceFile)

	run(opts.SchemaFile, gqlQuery)
}

func extractGQLFromFile(filePath string) (output string) {
	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading file", filePath, ":", err)
		os.Exit(1)
	}

	// Convert the content to a string
	fileContent := string(content)

	// Define the regular expression pattern to find GraphQL queries
	// This pattern looks for 'gql`' or 'graphql`' followed by any characters until a closing backtick.
	// It also captures the content between the backticks.
	re := regexp.MustCompile(`(?s)(?:gql|graphql)` + "`" + `(.*?)` + "`")

	// Find all matches in the file content
	matches := re.FindAllStringSubmatch(fileContent, -1)

	// Iterate through the matches and print the extracted queries
	if len(matches) > 0 {
		output = matches[0][1]
	} else {
		fmt.Println("No GraphQL queries found in the file.")
	}
	return
}

func run(schemaFilePath, gqlQuery string) {
	// Load schema
	schemaData, err := os.ReadFile(schemaFilePath)
	if err != nil {
		panic(err)
	}
	schema, err := gqlparser.LoadSchema(&ast.Source{Input: string(schemaData), Name: "schema.graphql"})
	if err != nil {
		panic(err)
	}

	// Load query

	queryDoc, err := parser.ParseQuery(&ast.Source{Input: gqlQuery, Name: "query.graphql"})
	if err != nil {
		panic(err)
	}

	visited := map[string]map[string]bool{}
	var output []string

	for _, op := range queryDoc.Operations {
		for _, sel := range op.SelectionSet {
			if field, ok := sel.(*ast.Field); ok {
				processField(field, schema.Query, schema, visited, &output)
			}
		}
	}

	for _, def := range output {
		fmt.Println(def)
		fmt.Println()
	}
}

// Recursive field processor
func processField(field *ast.Field, parentType *ast.Definition, schema *ast.Schema, visited map[string]map[string]bool, output *[]string) {
	fieldDef := schema.Types[parentType.Name].Fields.ForName(field.Name)
	if fieldDef == nil {
		return
	}

	fieldType := unwrapType(fieldDef.Type)
	typeDef := schema.Types[fieldType]

	if typeDef == nil || typeDef.BuiltIn {
		return
	}

	if visited[typeDef.Name] == nil {
		visited[typeDef.Name] = map[string]bool{}
	}
	for _, sel := range field.SelectionSet {
		if f, ok := sel.(*ast.Field); ok {
			visited[typeDef.Name][f.Name] = true
			processField(f, typeDef, schema, visited, output)
		}
	}

	// If not yet printed, add the trimmed type
	if !typeAlreadyAdded(typeDef.Name, *output) {
		*output = append(*output, buildPartialType(typeDef, visited[typeDef.Name]))
	}
}

func unwrapType(t *ast.Type) string {
	for t.Elem != nil {
		t = t.Elem
	}
	return t.NamedType
}

func buildPartialType(def *ast.Definition, fields map[string]bool) string {
	var sb strings.Builder
	sb.WriteString("type " + def.Name + " {\n")
	for _, f := range def.Fields {
		if fields[f.Name] {
			sb.WriteString("  " + f.Name + ": " + f.Type.String() + "\n")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func typeAlreadyAdded(name string, output []string) bool {
	for _, o := range output {
		if strings.HasPrefix(o, "type "+name+" ") {
			return true
		}
	}
	return false
}
