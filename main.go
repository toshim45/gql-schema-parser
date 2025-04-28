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

	gqlQuery := extractGQLFromFile(opts.SourceFile)

	if gqlQuery == "" {
		fmt.Fprintln(os.Stderr, "Error: no graphql query/mutation extracted")
		os.Exit(1)
	}

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
				if op.Operation == "query" {
					processField(field, schema.Query, schema, visited, &output)
				} else if op.Operation == "mutation" {
					processField(field, schema.Mutation, schema, visited, &output)
				} else {
					panic("gql operation type is not supported: " + op.Operation)
				}
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

	fieldHasArgs := map[string]bool{}
	for _, sel := range field.SelectionSet {
		if f, ok := sel.(*ast.Field); ok {
			visited[typeDef.Name][f.Name] = true
			if len(f.Arguments) > 0 {
				fieldHasArgs[f.Name] = true
			}
			processField(f, typeDef, schema, visited, output)
		}
	}

	processArgumentList(fieldHasArgs, typeDef, schema, output)

	// If not yet printed, add the trimmed type
	if !typeAlreadyAdded(typeDef.Name, *output) {
		*output = append(*output, buildPartialType(typeDef, visited[typeDef.Name]))
	}
}

func processArgumentList(fieldHasArgs map[string]bool, def *ast.Definition, schema *ast.Schema, output *[]string) {
	for _, f := range def.Fields {
		if _, exist := fieldHasArgs[f.Name]; exist && len(f.Arguments) > 0 {
			for _, a := range f.Arguments {
				t := unwrapType(a.Type)
				d := schema.Types[t]
				if d == nil || d.BuiltIn {
					continue
				}
				if !typeAlreadyAdded(d.Name, *output) {
					*output = append(*output, processArgument(d))
				}
			}
		}
	}
}

func processArgument(def *ast.Definition) string {
	var sb strings.Builder
	sb.WriteString("input " + def.Name + " {\n")
	if strings.HasSuffix(def.Name, "order_by") {
		sb.WriteString("  id : order_by\n")
	} else if strings.HasSuffix(def.Name, "bool_exp") {
		sb.WriteString("  _and: [" + def.Name + "!]\n")
		sb.WriteString("  _not: " + def.Name + "\n")
		sb.WriteString("  _or: [" + def.Name + "!]\n")
	}
	sb.WriteString("}")
	return sb.String()
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
			argStr := ""
			if len(f.Arguments) > 0 {
				var argStrList []string
				for _, arg := range f.Arguments {
					argStrList = append(argStrList, arg.Name+":"+arg.Type.Name())
				}

				argStr = "(" + strings.Join(argStrList, ",") + ")"
			}
			sb.WriteString("  " + f.Name + argStr + ": " + f.Type.String() + "\n")
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
