package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jessevdk/go-flags"
)

type GraphQLSchema struct {
	Types   map[string]GraphQLType
	Queries map[string]GraphQLQuery
}

type GraphQLType struct {
	Fields map[string]GraphQLField
}

type GraphQLField struct {
	Type        string
	IsEnum      bool
	IsRequired  bool
	Description string
}

type GraphQLQuery struct {
	Name   string
	Fields []string
}

func run(path string, isParse, isGenerate bool) {
	files, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Error reading directory:", err)
	}

	var queries []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		// Read TypeScript file(s)
		fileFullpath := filepath.Join(path, file.Name())
		fileContent, err := ioutil.ReadFile(fileFullpath)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}

		// Extract GraphQL queries from the TypeScript file
		qs := extractGraphQLQueries(string(fileContent))
		queries = append(queries, qs...)

	}

	if isParse {
		generateGraphQLObject(queries)
	} else if isGenerate {
		schema := generateGraphQLSchema(queries)
		// Print the generated GraphQL schema
		fmt.Println("Generated GraphQL Schema:")
		printSchema(schema)
	} else {
		fmt.Println("help: gqlsch -h")
	}
}

// Extract GraphQL queries from TypeScript files
func extractGraphQLQueries(tsContent string) []string {
	// Simple regex to find GraphQL queries inside gql``
	// Assumes all GraphQL queries are wrapped in gql`...`
	re := regexp.MustCompile(`gql\s*` + "`([^`]+)`")
	matches := re.FindAllStringSubmatch(tsContent, -1)

	var queries []string
	for _, match := range matches {
		queries = append(queries, match[1])
	}
	return queries
}

// Generate GraphQL schema based on the extracted queries
func generateGraphQLSchema(queries []string) GraphQLSchema {
	schema := GraphQLSchema{
		Types:   make(map[string]GraphQLType), // this one need to be initiated somewhere
		Queries: make(map[string]GraphQLQuery),
	}

	for _, query := range queries {
		// For simplicity, this basic example only extracts type names and fields
		queryName := extractQueryName(query)
		fields := extractFieldsFromQuery(query)

		schema.Queries[queryName] = GraphQLQuery{
			Name:   queryName,
			Fields: fields,
		}

		// Generate types for each field (very basic implementation)
		for _, field := range fields {
			// Create types for fields if not already defined
			if _, exists := schema.Types[field]; !exists {
				schema.Types[field] = GraphQLType{
					Fields: map[string]GraphQLField{
						"dummyField": {
							Type: "String",
						},
					},
				}
			}
		}
	}

	return schema
}

// Extract query name (this assumes the query name is the first word after 'query' keyword)
func extractQueryName(query string) string {
	re := regexp.MustCompile(`query\s+([A-Za-z0-9_]+)`)
	match := re.FindStringSubmatch(query)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// Extract fields from a query (this assumes fields are inside curly braces {})
func extractFieldsFromQuery(query string) []string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	match := re.FindStringSubmatch(query)
	if len(match) > 1 {
		lines := strings.Split(match[1], "\n")
		var result []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			result = append(result, line)
		}
		return result
	}
	return nil
}

// Print the GraphQL schema
func printSchema(schema GraphQLSchema) {
	for queryName, query := range schema.Queries {
		fmt.Printf("query %s {\n", queryName)
		for _, field := range query.Fields {
			fmt.Printf("  %s\n", field)
		}
		fmt.Println("}")
	}

	// Print Types
	for typeName, gqlType := range schema.Types {
		fmt.Printf("type %s {\n", typeName)
		for fieldName, field := range gqlType.Fields {
			fmt.Printf("  %s: %s\n", fieldName, field.Type)
		}
		fmt.Println("}")
	}
}

/**
* GraphQLObject approach
**/
var uniqueGQLObject map[string]*GraphQLObject

type GraphQLObject struct {
	Name     string
	Fields   map[string]bool // field-name:is-field-parent
	Children []*GraphQLObject
}

func (gqlo *GraphQLObject) printObjectList() {
	fmt.Println("[", gqlo.Name, "]")
	for n := range gqlo.Fields {
		fmt.Println(n)
	}
}

// parse GraphQLObject from lines
func parseGraphQLObject(lines []string) (*GraphQLObject, int) {
	lls := len(lines)
	if lls == 0 {
		return nil, 0
	}

	o := &GraphQLObject{}
	o.Fields = make(map[string]bool)

	namePattern := `(\w+)\(`
	reNamePattern := regexp.MustCompile(namePattern)

	skipFnParameter := false

	i := 0
	line := lines[0]

	for i < lls {
		// fmt.Println("[debug] processing ", i, "[", lls, "]", line)
		if line == ") {" {
			skipFnParameter = false
			i++
			line = lines[i]
			continue
		}

		if skipFnParameter {
			i++
			line = lines[i]
			continue
		}

		if line == "}" {
			break
		}

		ll := len(line)
		isLineAParent := line[ll-1] == '{'

		if match := reNamePattern.FindStringSubmatch(line); len(match) > 1 {
			if eo, exist := uniqueGQLObject[match[1]]; exist {
				o = eo
			} else {
				o.Name = match[1]
			}
			if line[ll-3:ll] != ") {" {
				skipFnParameter = true
			}
		} else {
			o.Fields[line] = isLineAParent
		}

		if i > 0 && isLineAParent {
			c, processedCount := parseGraphQLObject(lines[i+1 : lls])
			i = i + processedCount + 1
			o.Children = append(o.Children, c)
		}

		i++
		if i < lls {
			line = lines[i]
		}
	}

	return o, i
}

func generateGraphQLObject(queries []string) {
	uniqueGQLObject = make(map[string]*GraphQLObject)
	for _, query := range queries {
		fields := extractFieldsFromQuery(query)
		o, c := parseGraphQLObject(fields)
		if c > 0 {
			uniqueGQLObject[o.Name] = o
		}
	}

	for _, o := range uniqueGQLObject {
		o.printObjectList()
	}
}

var opts struct {
	Parse      bool `short:"p" long:"parse" description:"Function to parse graphql from file(s)"`
	Generate   bool `short:"g" long:"generate" description:"Function to generate graphql from file(s)"`
	Positional struct {
		Pathname string
	} `positional-args:"true" required:"true"`
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

	run(opts.Positional.Pathname, opts.Parse, opts.Generate)
}
