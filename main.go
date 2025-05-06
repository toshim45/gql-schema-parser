package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

var (
	opts struct {
		SchemaFile  string `long:"schema" description:"Input raw schema file"`
		SourceFile  string `long:"source" description:"Input source file which countain graphql (.js,.ts)"`
		FieldGQL    string `long:"field" description:"Input field GQL string"`
		TypeGQL     string `long:"type" description:"Input type GQL string"`
		Depth       uint   `short:"d" long:"depth" description:"Type recursion depth, default 5" default:"5"`
		IgnoredFile string `short:"i" long:"ignored" description:"Type ignored file path"`
	}

	scalarUnq    map[string]bool = map[string]bool{}
	ignoredTypes map[string]bool = map[string]bool{}
)

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
		fmt.Fprintf(os.Stderr, "⚠️ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("schema file: ", opts.SchemaFile)
	fmt.Println("source file: ", opts.SourceFile)
	fmt.Println("field string: ", opts.FieldGQL)
	fmt.Println("type string: ", opts.TypeGQL)
	fmt.Println("depth uint: ", opts.Depth)
	fmt.Println("ignored file: ", opts.IgnoredFile)

	fmt.Printf("-------\n\n")

	if opts.IgnoredFile != "" {
		parseIgnoredFile(opts.IgnoredFile)
	}

	if opts.SourceFile != "" {
		gqlQuery := extractGQLFromFile(opts.SourceFile)

		if gqlQuery == "" {
			fmt.Fprintln(os.Stderr, "⚠️ Error: no graphql query/mutation extracted")
			os.Exit(1)
		}

		fetchByQuery(opts.SchemaFile, gqlQuery)
	} else if opts.FieldGQL != "" {
		fetchByField(opts.SchemaFile, opts.FieldGQL, &opts.Depth)
	} else if opts.TypeGQL != "" {
		fetchByType(opts.SchemaFile, opts.TypeGQL, &opts.Depth)
	}
}

func parseIgnoredFile(filePath string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		panic("⚠️ Error reading ignored file" + filePath + ":" + err.Error())
	}

	lines := strings.Split(string(content), "\n")
	total := 0
	for i, line := range lines {
		ignoredTypes[line] = true
		total = i
	}

	total++

	fmt.Println("ignored:", total, "types")
	fmt.Printf("-------\n\n")
}

func LoadSchema(schemaFilePath string) *ast.Schema {
	schemaData, err := os.ReadFile(schemaFilePath)
	if err != nil {
		panic("schema read file: " + err.Error())
	}
	schema, err := gqlparser.LoadSchema(&ast.Source{Input: string(schemaData), Name: "schema.graphql"})
	if err != nil {
		panic("schema load: " + err.Error())
	}

	return schema
}

func extractGQLFromFile(filePath string) (output string) {
	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		panic("⚠️ Error reading gql file" + filePath + ":" + err.Error())
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

func fetchByQuery(schemaFilePath, gqlQuery string) {
	// Load schema
	schema := LoadSchema(schemaFilePath)

	// Load query

	queryDoc, err := parser.ParseQuery(&ast.Source{Input: gqlQuery, Name: "query.graphql"})
	if err != nil {
		panic(err)
	}

	visited := map[string]map[string]bool{}
	var output []string

	for _, op := range queryDoc.Operations {
		for _, vd := range op.VariableDefinitions {
			d := schema.Types[vd.Type.NamedType]
			if d == nil || d.BuiltIn {
				continue
			}

			output = append(output, processArgument(d))
		}
		for _, sel := range op.SelectionSet {
			if field, ok := sel.(*ast.Field); ok {
				if op.Operation == "query" {
					processField(field, schema.Query, schema, visited, &output)
				} else if op.Operation == "mutation" {
					processField(field, schema.Mutation, schema, visited, &output)
				} else {
					panic("⚠️ Error: gql operation type is not supported: " + op.Operation)
				}
			}
		}
	}

	for _, o := range output {
		fmt.Println(o)
	}
}

func fetchByType(schemaFilePath, gqlType string, depth *uint) {
	// Load schema
	schema := LoadSchema(schemaFilePath)
	visited := map[string]bool{}
	outputs := []string{}

	printSchemaField(schema, gqlType, visited, &outputs, depth)

	for _, output := range outputs {
		fmt.Println(output)
	}
}

func fetchByField(schemaFilePath, gqlField string, depth *uint) {
	// Load schema
	schema := LoadSchema(schemaFilePath)

	fields := strings.Split(gqlField, " ")
	if len(fields) != 2 {
		fmt.Println("⚠️ Error: the format must be query/mutation field_name, example: mutation create_job")
		return
	}
	opType := fields[0]
	opName := fields[1]

	var fieldDef *ast.FieldDefinition
	if opType == "query" {
		fieldDef = schema.Types[schema.Query.Name].Fields.ForName(opName)
	} else if opType == "mutation" {
		fieldDef = schema.Types[schema.Mutation.Name].Fields.ForName(opName)
	} else {
		panic("⚠️ Error: gql operation type is not supported: " + opType)
	}

	if fieldDef == nil {
		return
	}

	visited := map[string]bool{}
	outputs := []string{}

	// Print field info
	fmt.Printf("Field Name: %s\n", fieldDef.Name)
	fmt.Printf("Type: %s\n", fieldDef.Type.String())
	fmt.Printf("Description: %s\n", fieldDef.Description)
	fmt.Println("Arguments:")
	for _, arg := range fieldDef.Arguments {
		fmt.Printf("- %s: %s\n", arg.Name, arg.Type.String())
		argBase := arg.Type.Name()
		if isCustomType(argBase) {
			printSchemaField(schema, argBase, visited, &outputs, depth)
		}
	}

	if isCustomType(fieldDef.Type.Name()) {
		printSchemaField(schema, fieldDef.Type.Name(), visited, &outputs, depth)
	}

	fmt.Printf("\n-------\n\n")

	for _, def := range outputs {
		fmt.Println(def)
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

	if strings.HasSuffix(def.Name, "select_column") {
		sb.WriteString("enum " + def.Name + " {\n")
		sb.WriteString("  id\n")
	} else if strings.HasSuffix(def.Name, "order_by") {
		sb.WriteString("input " + def.Name + " {\n")
		sb.WriteString("  id : order_by\n")
	} else if strings.HasSuffix(def.Name, "bool_exp") {
		sb.WriteString("input " + def.Name + " {\n")
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
	if def.Name == "uuid" || def.Name == "json" || def.Name == "timestamptz" {
		if _, exist := scalarUnq[def.Name]; exist {
			return ""
		}

		scalarUnq[def.Name] = true
		return "scalar " + def.Name
	}
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

var builtInScalars = map[string]bool{
	"Int":     true,
	"Float":   true,
	"String":  true,
	"Boolean": true,
	"ID":      true,
}

// isCustomType checks if a type is not a built-in scalar or introspection type
func isCustomType(typeName string) bool {
	return !builtInScalars[typeName] && !strings.HasPrefix(typeName, "__")
}

// printSchemaField prints a type definition and recursively prints nested types
func printSchemaField(schema *ast.Schema, typeName string, visited map[string]bool, outputs *[]string, depth *uint) {
	if _, exist := ignoredTypes[typeName]; exist {
		return
	}

	if *depth > 0 {
		*depth--
	} else {
		return
	}
	if visited[typeName] {
		return
	}
	visited[typeName] = true

	typ := schema.Types[typeName]
	if typ == nil {
		panic("⚠️ Error: Type " + typeName + " not found in schema\n")
	}

	var sb strings.Builder
	switch typ.Kind {
	case ast.InputObject:
		sb.WriteString("input " + typ.Name + " {\n")
		for _, f := range typ.Fields {
			sb.WriteString("  " + f.Name + ": " + f.Type.String() + "\n")
		}
		sb.WriteString("}")
		for _, f := range typ.Fields {
			nestedType := f.Type.Name()
			if isCustomType(nestedType) {
				printSchemaField(schema, nestedType, visited, outputs, depth)
			}
		}
	case ast.Object:
		sb.WriteString("type " + typ.Name + " {\n")
		for _, f := range typ.Fields {
			sb.WriteString("  " + f.Name + ": " + f.Type.String() + "\n")
		}
		sb.WriteString("}")
		for _, f := range typ.Fields {
			nestedType := f.Type.Name()
			if isCustomType(nestedType) {
				printSchemaField(schema, nestedType, visited, outputs, depth)
			}
		}
	case ast.Enum:
		sb.WriteString(fmt.Sprintf("enum %s: %v\n", typ.Name, typ.EnumValues))
	case ast.Interface:
		sb.WriteString("interface " + typ.Name + " {\n")
		for _, f := range typ.Fields {
			fmt.Printf("  %s: %s\n", f.Name, f.Type.String())
		}
		sb.WriteString("}")
	case ast.Scalar:
		sb.WriteString("scalar: " + typ.Name + "\n")
	default:
		panic("⚠️ Error: Unknown type kind " + string(typ.Kind) + " for type " + typ.Name + "\n")
	}

	*outputs = append(*outputs, sb.String())
}

func typeAlreadyAdded(name string, output []string) bool {
	for _, o := range output {
		if strings.HasPrefix(o, "type "+name+" ") {
			return true
		}
	}
	return false
}

type ImportResult struct {
	Name     string
	FromPath string
}

func GetImportFromDir(directoryPath string) []ImportResult {
	var results []ImportResult
	unq := map[string]bool{}

	// Regular expression to match import statements with destructuring
	importRegex := regexp.MustCompile(`import\s*{([^}]+)}\s*from\s*['"](@wms/hooks/[^'"]+)['"]`)
	// Regex for single import statements
	singleImportRegex := regexp.MustCompile(`import\s+{?\s*(\w+)\s*}?\s*from\s*['"](@wms/hooks/[^'"]+)['"]`)

	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		paths := strings.Split(path, "/pages/")
		if len(paths) < 2 {
			// fmt.Println("[DEBUG] invalid path", path)
			return nil
		}
		lastIdxOfSlash := strings.LastIndex(paths[1], "/")

		var pagePath string
		if lastIdxOfSlash != -1 {
			pagePath = paths[1][:lastIdxOfSlash]
		}
		if _, exist := eligiblePage[pagePath]; !exist {
			// fmt.Println("[DEBUG] invalid page", paths[1])
			return nil
		}

		// Skip if not a TypeScript/JavaScript file
		if !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Find all destructured imports
		matches := importRegex.FindAllStringSubmatch(fileContent, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				imports := strings.Split(match[1], ",")
				fromPath := strings.TrimPrefix(match[2], "@")

				for _, imp := range imports {
					// Clean up the import name
					name := strings.TrimSpace(imp)
					name = strings.Trim(name, "{}")
					if name == "" {
						continue
					}

					if _, exist := unq[name]; !exist {
						unq[name] = true
						results = append(results, ImportResult{
							Name:     name,
							FromPath: fromPath,
						})
					}
				}
			}
		}

		// Find all single imports
		singleMatches := singleImportRegex.FindAllStringSubmatch(fileContent, -1)
		for _, match := range singleMatches {
			if len(match) >= 3 {
				name := strings.TrimSpace(match[1])
				fromPath := strings.TrimPrefix(match[2], "@")

				if _, exist := unq[name]; !exist {
					unq[name] = true
					results = append(results, ImportResult{
						Name:     name,
						FromPath: fromPath,
					})
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil
	}

	return results
}

func GetEligiblePage(directoryPath string) map[string]bool {
	unqEligiblePage := map[string]bool{}
	// Regex for eligigle route
	routeRegex := regexp.MustCompile(`path:\s*['"](/[a-zA-Z-]+(?:/[a-zA-Z-]+)*)['"]`)

	var eligibleRoute map[string]bool = map[string]bool{
		"/inbound-v3/schedule":                    true,
		"/inbound-v3/unload":                      true,
		"/inbound-v3/backlog-monitoring":          true,
		"/inbound-v3/inventory-group-monitoring":  true,
		"/inbound/schedule":                       true,
		"/inventory/move-stock":                   true,
		"/inventory/move-return-stock":            true,
		"/inventory/move-totes-stock":             true,
		"/inventory/move-available-stock":         true,
		"/inventory/move-allocated-stock":         true,
		"/inventory/move-group":                   true,
		"/inventory/admin":                        true,
		"/inventory/adjustment-log":               true,
		"/inventory-group/admin":                  true,
		"/inventory/counting":                     true,
		"/inventory/warehouse-transfer":           true,
		"/inventory/task-management":              true,
		"/product":                                true,
		"/replenishment":                          true,
		"/replenishment/operator":                 true,
		"/outbound/admin":                         true,
		"/outbound/wave-config/schedule":          true,
		"/outbound/wave":                          true,
		"/outbound/wave-picking-task":             true,
		"/outbound/wave-picking-task/queue":       true,
		"/outbound/wave-consolidation-task":       true,
		"/outbound/checkpack":                     true,
		"/outbound/station-management":            true,
		"/outbound/station/dashboard":             true,
		"/outbound/class-management":              true,
		"/outbound/packaging-simulator":           true,
		"/outbound/packaging-visualizer":          true,
		"/outbound/packer":                        true,
		"/outbound/packing-task":                  true,
		"/outbound/peso/detail":                   true,
		"/shipment/receiving":                     true,
		"/shipment/labeling":                      true,
		"/shipment/grouping":                      true,
		"/shipment/loading":                       true,
		"/shipment/routing-manifest":              true,
		"/shipment/manifest-v2":                   true,
		"/shipment/manifest-destroy":              true,
		"/shipment/handover":                      true,
		"/shipment/waybill":                       true,
		"/shipment/waypoint":                      true,
		"/shipment/courier":                       true,
		"/shipment/databank":                      true,
		"/shipment/databank/group":                true,
		"/integration/virtual-bundling-v2":        true,
		"/integration/wsn-admin":                  true,
		"/integration/gifting":                    true,
		"/tenant":                                 true,
		"/tenant/group":                           true,
		"/job":                                    true,
		"/admin/location":                         true,
		"/admin/user":                             true,
		"/intools/packaging-config":               true,
		"/intools/packaging-recommender/rule-set": true,
		"/intools/remote-config":                  true,
		"/intools/warehouse/zone":                 true,
		"/intools/outbound/schedule":              true,
		"/tell-me-why":                            true,
	}

	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a TypeScript/JavaScript file
		if !strings.HasSuffix(path, "index.ts") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// whitelist route
		routeMatches := routeRegex.FindAllStringSubmatch(fileContent, 1)
		for _, match := range routeMatches {
			if len(match) == 2 {
				if _, exist := eligibleRoute[match[1]]; !exist {
					fmt.Println("[DEBUG] invalid route", path, match[1])
					return nil
				}
			}
		}

		unqEligiblePage[path[:len(path)-9]] = true

		return nil
	})

	if err != nil {
		return nil
	}

	return unqEligiblePage
}

var eligiblePage map[string]bool = map[string]bool{
	"InventoryMoveStock/V2/utils":                true,
	"OutboundWavePickingTask":                    true,
	"ShipmentWaybill":                            true,
	"IntegrationGifting":                         true,
	"IntegrationWsnAdmin":                        true,
	"OutboundPackagingSimulator":                 true,
	"PackagingRecommenderPackagingConfig":        true,
	"InboundV3BacklogMonitoring":                 true,
	"OutboundWaveConsolidationTask":              true,
	"ShipmentDatabank":                           true,
	"ShipmentLoading":                            true,
	"InboundV3Unload":                            true,
	"OutboundClassManagement":                    true,
	"InventoryProduct":                           true,
	"ShipmentWaypoint":                           true,
	"InboundV3InventoryGroupMonitoring":          true,
	"OutboundAdmin":                              true,
	"ShipmentManifestV2":                         true,
	"InboundSchedule":                            true,
	"ShipmentHandover":                           true,
	"Tenant":                                     true,
	"PackagingRecommendationDecisionTableDetail": true,
	"OutboundPackingTask":                        true,
	"PackagingRecommendationRuleSet":             true,
	"InventoryWarehouseTransfer":                 true,
	"OutboundStationManagement":                  true,
	"RemoteConfig":                               true,
	"ShipmentReceiving":                          true,
	"OutboundCheckPack":                          true,
	"InventoryCounting/__gql_mocks__":            true,
	"InventoryMoveStock/V2/services":             true,
	"InventoryAdmin":                             true,
	"InboundV3Schedule":                          true,
	"InventoryAdjustmentLog":                     true,
	"InventoryReplenishmentTaskManagement":       true,
	"InventoryWarehouseTransfer/components/InventoryWarehouseTransferDetail": true,
	"AdminUser":                         true,
	"OutboundPackerV2":                  true,
	"OutboundStationManagementV2":       true,
	"ShipmentLabeling":                  true,
	"ShipmentManifestDestroy":           true,
	"ShipmentTellMeWhyAllocationFailed": true,
	"IntegrationVirtualBundlingV2":      true,
	"InventoryGroupAdmin":               true,
	"OutboundSchedulerConfig":           true,
	"OutboundWaveConfig":                true,
	"OutboundWavePickingTaskQueue":      true,
	"WarehouseMap":                      true,
	"IntegrationVirtualBundling":        true,
	"InventoryCounting":                 true,
	"InventoryMoveGroup":                true,
	"Location":                          true,
	"OutboundPacker":                    true,
	"OutboundPesoDetail":                true,
	"InboundInstruction":                true,
	"Job":                               true,
	"PackagingRecommenderVisualizer":    true,
	"InventoryMoveStock":                true,
	"OutboundWave":                      true,
	"ShipmentGrouping":                  true,
	"InventoryTaskManagement":           true,
}

var targetHooks map[string]bool = map[string]bool{
	"useCreateUserV2":                                             true,
	"useUserListQuery":                                            true,
	"useEditUserV2":                                               true,
	"useCreateUserRoleV2":                                         true,
	"useEditUserRoleV2":                                           true,
	"useRemoveUserRoleV2":                                         true,
	"useUserDetailQuery":                                          true,
	"useWarehouseListQuery":                                       true,
	"useOptions":                                                  true,
	"useUserRolePageQuery":                                        true,
	"useUserRoleDetailQuery":                                      true,
	"useRoleUserListQuery":                                        true,
	"useRoleGroupListQuery":                                       true,
	"useUserRoleGroupDetailQuery":                                 true,
	"useRoleGroupUserListQuery":                                   true,
	"useInboundInstruction":                                       true,
	"useInboundPageQuery":                                         true,
	"useInboundQuery":                                             true,
	"useInboundV3BacklogMonitoringCounterQuery":                   true,
	"useInboundV3BacklogMonitoringPageQuery":                      true,
	"useUserUtils":                                                true,
	"useInboundV3BacklogMonitoringConfirmationQuery":              true,
	"useReceivingCompleteInboundV3Instruction":                    true,
	"useInboundV3InventoryGroupDetailQuery":                       true,
	"useGetZplInboundV3GroupLabelLazyQuery":                       true,
	"useInboundV3InventoryGroupBacklogListQuery":                  true,
	"useApproveInboundV3Instruction":                              true,
	"useCancelInboundV3Instruction":                               true,
	"useCreateInboundV3Instruction":                               true,
	"useCreateInboundV3InstructionAttribute":                      true,
	"useCreateInboundV3InstructionLine":                           true,
	"useEditInboundV3InstructionAttribute":                        true,
	"useEditInboundV3InstructionLine":                             true,
	"useRejectInboundV3Instruction":                               true,
	"useRemoveInboundV3InstructionAttribute":                      true,
	"useRemoveInboundV3InstructionLine":                           true,
	"useRescheduleInboundV3Instruction":                           true,
	"useDisputeInboundV3Instruction":                              true,
	"useProductUomListQuery":                                      true,
	"useInboundV3ClassListQuery":                                  true,
	"useInboundV3InstructionPageQuery":                            true,
	"useInboundV3InstructionCounterQuery":                         true,
	"useInboundV3InstructionDetailQuery":                          true,
	"useInboundV3InstructionLineListQuery":                        true,
	"useInboundV3ReceivingListQuery":                              true,
	"useGetInboundV3InstructionLineDetailListViewQuery":           true,
	"useInboundV3LookupListQuery":                                 true,
	"usePutawayCompleteInboundV3Inbound":                          true,
	"useReceivingCompleteInboundV3Inbound":                        true,
	"useInboundV3ReceivingUserListQuery":                          true,
	"useUserPageQuery":                                            true,
	"useInboundV3InboundCounterQuery":                             true,
	"useInboundV3ListQuery":                                       true,
	"useInboundV3Query":                                           true,
	"useCancelInboundV3Inbound":                                   true,
	"useUnloadInboundV3Inbound":                                   true,
	"useTenantConfigListQuery":                                    true,
	"useCreateInboundV3Inbound":                                   true,
	"useEditInboundV3Inbound":                                     true,
	"useInboundV3InstructionListOptionsQuery":                     true,
	"useQueryOptions":                                             true,
	"useInboundV3InstructionDetailUnloadQuery":                    true,
	"useInboundV3InstructionListQuery":                            true,
	"useInboundV3InboundStatusHistoryQuery":                       true,
	"useUpdateGRNUnloadFilesInboundV3Inbound":                     true,
	"useOutboundListGiftingQuery":                                 true,
	"useEditOutboundAttributeList":                                true,
	"useIntegrationBundleStockMovementListQuery":                  true,
	"useGetTokopediaStockLazyQuery":                               true,
	"useVirtualBundlingListQuery":                                 true,
	"useGetTokopediaStockQuery":                                   true,
	"useVirtualBundlingDetailQuery":                               true,
	"useCreateVirtualBundlingOne":                                 true,
	"useUpdateVirtualBundlingOne":                                 true,
	"useGetVirtualBundleQuery":                                    true,
	"useGetVirtualBundlingDetailQuery":                            true,
	"useOutboundListQuery":                                        true,
	"useOutboundQuery":                                            true,
	"useGetWsnVehicleSuggestionQuery":                             true,
	"useWsnProductListQuery":                                      true,
	"useWsnReadyForPickupUpdate":                                  true,
	"useWsnPickupUpdate":                                          true,
	"useWsnFinishPacking":                                         true,
	"useStockWorkAdjustmentListQuery":                             true,
	"useWarehouseLocationDetailQuery":                             true,
	"useProductDetailQuery":                                       true,
	"useStockInventoryAdminListQuery":                             true,
	"useStockInventoryAdminDetailQuery":                           true,
	"useEditInventoryAdminSerialNumber":                           true,
	"useEditInventoryAdminExpiryDate":                             true,
	"useEditInventoryAdminLotNumber":                              true,
	"useCreateCountingCountList":                                  true,
	"useUpdateCountingCountUser":                                  true,
	"useUpdateCountingRequestCanceled":                            true,
	"useCountingRequestListQuery":                                 true,
	"useCountingRequestQuery":                                     true,
	"useExecuteAction":                                            true,
	"useCountingReconcileStockInventoryAllocationDetailListQuery": true,
	"useCountingActionAttributeListQuery":                         true,
	"useCountingReconcileResultListQuery":                         true,
	"useCountingResultAttributeListQuery":                         true,
	"useCreateCountingRequest":                                    true,
	"useCountingTaskListDesktopQuery":                             true,
	"useUpdateCountingTaskCanceled":                               true,
	"useGetZplInvGroupLabelLazyQuery":                             true,
	"useStockInventoryGroupReferenceListQuery":                    true,
	"useCreateInventoryGroup":                                     true,
	"useEditInventoryGroupLifeCycle":                              true,
	"useStockInventoryGroupListAdminQuery":                        true,
	"useStockInventoryListQuery":                                  true,
	"useStockInventoryGroupListQuery":                             true,
	"useStockInventoryGroupListLazyQuery":                         true,
	"useCreateWork":                                               true,
	"useWarehouseLocationListLazyQuery":                           true,
	"useStockInventoryMoveStockListLazyQuery":                     true,
	"useStockInventoryMoveStockListQuery":                         true,
	"useStockInventoryGroupMoveStockListLazyQuery":                true,
	"useWarehouseLocationClassListLazyQuery":                      true,
	"useWarehouseConfigPageLazyQuery":                             true,
	"useStockWorkflowReasonListLazyQuery":                         true,
	"OptionType":                                                  true,
	"useOptionReturnType":                                         true,
	"parseSerialNumberBarcode":                                    true,
	"parseOldBarcodeWithSKU":                                      true,
	"useCreateProductAttribute":                                   true,
	"useEditProductAttribute":                                     true,
	"useProductAttributeListQuery":                                true,
	"useRemoveProductAttribute":                                   true,
	"useProductBundleListQuery":                                   true,
	"useRemoveProductBundle":                                      true,
	"useEditProductBundle":                                        true,
	"useCreateProductBundle":                                      true,
	"useEditProductCategory":                                      true,
	"useProductCategoryDetailQuery":                               true,
	"useProductPageQuery":                                         true,
	"useProductCategoryRuleListQuery":                             true,
	"useProductClassConfigurationListQuery":                       true,
	"useEditProductMutation":                                      true,
	"useActivateProduct":                                          true,
	"useDeactivateProduct":                                        true,
	"useUploadImage":                                              true,
	"useProductFormOptionsQuery":                                  true,
	"useCreateProduct":                                            true,
	"useProductStandardUomListQuery":                              true,
	"useRemoveProductUOM":                                         true,
	"useRemoveProductIdentifier":                                  true,
	"useCreateUOMMutation":                                        true,
	"useEditUOMMutation":                                          true,
	"useAssignReplenishment":                                      true,
	"useGetReplenishmentListLazyQuery":                            true,
	"useGetReplenishmentListQuery":                                true,
	"useWarehouseLocationListQuery":                               true,
	"ForkliftDriverTaskType":                                      true,
	"useProductListQuery":                                         true,
	"ReplenishmentOperatorTaskType":                               true,
	"useTenantListQuery":                                          true,
	"useAssignTaskReplenish":                                      true,
	"useGetTaskManagementListQuery":                               true,
	"useWarehouseTransferListQuery":                               true,
	"useCreateWarehouseTransfer":                                  true,
	"useWarehouseTransferDetailQuery":                             true,
	"useApproveWarehouseTransfer":                                 true,
	"useCancelWarehouseTransfer":                                  true,
	"useRevertWarehouseTransfer":                                  true,
	"useSubmitWarehouseTransfer":                                  true,
	"useUpdateWarehouseTransfer":                                  true,
	"useOutboundClassListWhTransferQuery":                         true,
	"useWarehouseTransferClassConfigListQuery":                    true,
	"useAddWarehouseTransferLines":                                true,
	"useDeleteWarehouseTransferLines":                             true,
	"useUpdateWarehouseTransferLines":                             true,
	"useWarehouseTransferLineListQuery":                           true,
	"useInboundV3InstructionListWarehouseTransferQuery":           true,
	"useOutboundListWarehouseTransferQuery":                       true,
	"useWaybillListWarehouseTransferQuery":                        true,
	"useWarehouseTransferTransactionListQuery":                    true,
	"useRejectWarehouseTransfer":                                  true,
	"useJobListQuery":                                             true,
	"useJobDetailQuery":                                           true,
	"useJobParameterListQuery":                                    true,
	"useEditCancelJob":                                            true,
	"useCreateLocation":                                           true,
	"useWarehouseLocationListAdminQuery":                          true,
	"useEditLocation":                                             true,
	"useGetZplItemListBoxLabelLazyQuery":                          true,
	"useEditOutboundStatusCancelled":                              true,
	"useOutboundAdminListQuery":                                   true,
	"useOutboundAdminListQueryV2":                                 true,
	"useWaybillListLazyQuery":                                     true,
	"useOutboundV2Query":                                          true,
	"useGetOutboundIsCancelableQuery":                             true,
	"useEditOutboundStatusApproved":                               true,
	"useInvokeOutboundAllocation":                                 true,
	"useGetZplGiftingLabelLazyQuery":                              true,
	"useEditOutboundCheckerMaxBox":                                true,
	"useGetZplItemListLabelLazyQuery":                             true,
	"useEditOutboundPartialProcessPolicy":                         true,
	"useEditOutboundStatusRejected":                               true,
	"useEditShipmentWaybillShipmentList":                          true,
	"useCourierRoutePlanListQuery":                                true,
	"useCheckerStockInventoryListLazyQuery":                       true,
	"useOutboundCheckerUomListLazyQuery":                          true,
	"useOutboundWorkListLazyQuery":                                true,
	"useOutboundCreateCheckerBox":                                 true,
	"useEditOutboundCheckerFinish":                                true,
	"useCreateOutboundCheckerBoxAndFinish":                        true,
	"useEditOutboundCheckerBoxItem":                               true,
	"useEditOutboundStartChecking":                                true,
	"useGetZplShippingLabelLazyQuery":                             true,
	"useOutboundWaybillShipmentListQuery":                         true,
	"useCreateOutboundClassPriority":                              true,
	"useGetOutboundClassPriorityById":                             true,
	"useGetOutboundClassManagementListLazyQuery":                  true,
	"useGetOutboundClassManagementPriorityListLazyQuery":          true,
	"useUpdateOutboundClass":                                      true,
	"useCreateJob":                                                true,
	"useUserUtilsStore":                                           true,
	"useOutboundPackagingSimulatorListQuery":                      true,
	"useJobListLazyQuery":                                         true,
	"useCancelOutboundPackagingSimulator":                         true,
	"useEditOutboundStartPacking":                                 true,
	"useEditOutboundPackingNeedPeso":                              true,
	"useEditOutboundPackingProcessItem":                           true,
	"useStockInventoryGroupListPackerLazyQuery":                   true,
	"useUserListLazyQuery":                                        true,
	"useWaybillListOutboundPackerQuery":                           true,
	"useEditOutboundPackingV2Complete":                            true,
	"useEditOutboundPackingV2Start":                               true,
	"useEditOutboundPackingV2NeedPeso":                            true,
	"useEditOutboundPackingV2ProcessItem":                         true,
	"useOutboundScheduleListQuery":                                true,
	"useOutboundWaveListQuery":                                    true,
	"useWarehouseListLazyQuery":                                   true,
	"useGetOutboundStationStats":                                  true,
	"useGetOutboundStationStatsDetail":                            true,
	"useOutboundStation":                                          true,
	"useEditOutboundStationStatus":                                true,
	"useGetOutboundStationWaveList":                               true,
	"useGetOutboundClassManagementPriorityListQuery":              true,
	"useOutboundWaveConsolidationListQuery":                       true,
	"useWarehouseLocationListZoneDetailQuery":                     true,
	"useWarehouseZoneQuery":                                       true,
	"useOutboundWaveSortingTaskOutboundListQuery":                 true,
	"useOutboundWavePickingTaskPickerAssignment":                  true,
	"useOutboundClassListLazyQuery":                               true,
	"useOutboundClassPriorityListLazyQuery":                       true,
	"useOutboundWaveRuleSetListQuery":                             true,
	"useOutboundWaveRuleSetQuery":                                 true,
	"useCreateOutboundWave":                                       true,
	"useTenantListLazyQuery":                                      true,
	"useOutboundWaveQuery":                                        true,
	"useOutboundWaveSortingListQuery":                             true,
	"useOutboundWavePickingListQuery":                             true,
	"useCancelOutboundWave":                                       true,
	"useOutboundWaveRuleSnapshot":                                 true,
	"useCreateOutboundWaveRuleSet":                                true,
	"useCreateOutboundWaveSchedule":                               true,
	"useOutboundWaveRuleSetListLazyQuery":                         true,
	"useOutboundWaveScheduleQuery":                                true,
	"useEditOutboundWaveRuleSet":                                  true,
	"useEditOutboundWaveSchedule":                                 true,
	"useOutboundWaveScheduleListQuery":                            true,
	"useEditOutboundWaveRuleSetMap":                               true,
	"useOutboundWavePickingQueueQuery":                            true,
	"usePackagingRecommendationDecisionTableQuery":                true,
	"usePackagingRecommendationRecommendationConfigListQuery":     true,
	"usePackagingRecommendationRuleSetListQuery":                  true,
	"usePackagingRecommenderLibSync":                              true,
	"useCreatePackage":                                            true,
	"usePackagingRecommendationPackageTypeListQuery":              true,
	"useCreatePackageGroup":                                       true,
	"useCreatePackageType":                                        true,
	"useEditPackage":                                              true,
	"useEditPackageGroup":                                         true,
	"useEditPackageType":                                          true,
	"useCreatePackageGroupType":                                   true,
	"useDeletePackageGroupType":                                   true,
	"useEditPackageGroupType":                                     true,
	"usePackagingRecommendationPackageGroupTypeListQuery":         true,
	"usePackagingRecommendationPackageDetailQuery":                true,
	"usePackagingRecommendationPackageListQuery":                  true,
	"useDeletePackage":                                            true,
	"usePackagingRecommendationPackageGroupListQuery":             true,
	"useDeletePackageGroup":                                       true,
	"useDeletePackageType":                                        true,
	"usePackagingSimulationV2LazyQuery":                           true,
	"usePackagingSimulationV2ByOumListLazyQuery":                  true,
	"usePackagingSimulationV2ByOutboundLazyQuery":                 true,
	"type RemoteConfigItemData":                                   true,
	"type RemoteConfigScopeData":                                  true,
	"useCreateConfigRule":                                         true,
	"useUpdateConfigRule":                                         true,
	"ConfigRuleTypeEnum":                                          true,
	"useUpdateRemoteConfig":                                       true,
	"useRemoteConfigListQuery":                                    true,
	"useCompanyConfigListLazyQuery":                               true,
	"useDeleteConfigRule":                                         true,
	"useDatabankGroupListQuery":                                   true,
	"useDatabankListQuery":                                        true,
	"useCreateShipmentWaybillGroupMutation":                       true,
	"useGetZplGroupingLabelLazyQuery":                             true,
	"useWaybillListAdminQuery":                                    true,
	"useWaybillGroupingDetailQuery":                               true,
	"useHandoverDetailLazyQuery":                                  true,
	"useHandoverGetManifestLazyQuery":                             true,
	"useCreateShipmentHandoverDetailMutation":                     true,
	"useCreateShipmentHandoverMutation":                           true,
	"useEditShipmentHandoverStatusClosedMutation":                 true,
	"useRemoveShipmentHandoverDetailMutation":                     true,
	"useHandoverListQuery":                                        true,
	"useHandoverDetailQuery":                                      true,
	"useEditShipmentWaybillStatusLabelingMutation":                true,
	"useWaybillListLoadingLazyQuery":                              true,
	"useEditShipmentWaybillMutation":                              true,
	"useStockInventoryGroupListLoadingLazyQuery":                  true,
	"useManifestListQuery":                                        true,
	"useManifestDetailQuery":                                      true,
	"useWaybillListManifestLazyQuery":                             true,
	"useManifestWaybillChildrenListQuery":                         true,
	"useCreateShipmentManifestMutation":                           true,
	"useCreateShipmentManifestDetailMutation":                     true,
	"useEditShipmentManifestStatusClosedMutation":                 true,
	"useRemoveShipmentManifestDetailMutation":                     true,
	"useCreateShipmentManifestTruckMutation":                      true,
	"useEditShipmentManifestTruckMutation":                        true,
	"useCreateShipmentManifestDestroyMutation":                    true,
	"useEditShipmentWaybillStatusReceivedMutation":                true,
	"useCourierServiceListLazyQuery":                              true,
	"useWaybillTellMeWhyAllocationFailedLazyQuery":                true,
	"useEditShipmentWaybillStatusMutation":                        true,
	"useEditShipmentStatusMutation":                               true,
	"useEditShipmentWaybillWeightAndDimensionMutation":            true,
	"useWaybillDetailQuery":                                       true,
	"useDuplicateShipmentWaybillMutation":                         true,
	"useWaypointListQuery":                                        true,
	"useResolveWaypointAdminQuery":                                true,
	"useCreateShipmentWaypointMutation":                           true,
	"useEditShipmentWaypointMutation":                             true,
	"useRegionEdgeListLazyQuery":                                  true,
	"useRegionListLazyQuery":                                      true,
	"useRegionListQuery":                                          true,
	"useCreateTenantGroupMember":                                  true,
	"useRemoveTenantGroupMember":                                  true,
	"useTenantConfigDetailQuery":                                  true,
	"useCreateTenantConfig":                                       true,
	"useEditTenantConfig":                                         true,
	"useTenantConfigPageQuery":                                    true,
	"useRemoveTenantConfig":                                       true,
	"useTenantPageQuery":                                          true,
	"useCreateTenant":                                             true,
	"useEditTenant":                                               true,
	"useTenantDetailQuery":                                        true,
	"useTenantGroupPageQuery":                                     true,
	"useCreateTenantGroup":                                        true,
	"useEditTenantGroup":                                          true,
	"useTenantGroupDetailQuery":                                   true,
	"useTenantGroupMemberPageQuery":                               true,
	"useTenantInvoiceServiceDetailQuery":                          true,
	"useTenantRateDetailQuery":                                    true,
	"useCreateTenantInvoiceRate":                                  true,
	"useEditTenantInvoiceRate":                                    true,
	"useTenantRatePageQuery":                                      true,
}

func GetGQLImport(hooksParentFolderPath string) []string {
	// Target hook functions to look for

	// Store unique GQL imports
	gqlImports := make(map[string]bool)

	// Walk through the directory
	filepath.Walk(hooksParentFolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a TypeScript file
		if !strings.HasSuffix(info.Name(), ".ts") {
			return nil
		}

		// Check if file name matches any of our target hooks
		fileName := strings.TrimSuffix(info.Name(), ".ts")
		if !targetHooks[fileName] {
			return nil
		}

		// Read file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			// Look for import statements containing @wms/gqls
			if strings.Contains(line, "import") && strings.Contains(line, "@wms/gqls") {
				// Extract the import path
				if start := strings.Index(line, "@wms/gqls"); start != -1 {
					importPath := line[start:]
					if end := strings.Index(importPath, "'"); end != -1 {
						gqlImports[importPath[:end]] = true
					} else if end := strings.Index(importPath, "\""); end != -1 {
						gqlImports[importPath[:end]] = true
					}
				}
			}
		}

		return scanner.Err()
	})

	// Convert map to slice
	result := make([]string, 0, len(gqlImports))
	for imp := range gqlImports {
		result = append(result, imp)
	}

	return result
}
