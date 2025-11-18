package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Schema structures matching OCSF JSON format
type EventClass struct {
	UID         int                        `json:"uid"`
	Caption     string                     `json:"caption"`
	Description string                     `json:"description"`
	Extends     string                     `json:"extends"`
	Name        string                     `json:"name"`
	Attributes  map[string]json.RawMessage `json:"attributes"`
	Category    string                     `json:"category"`
}

type AttributeRef struct {
	Group       string               `json:"group"`
	Requirement string               `json:"requirement"`
	Description string               `json:"description"`
	Enum        map[string]EnumValue `json:"enum"`
	Sibling     string               `json:"sibling"`
}

type EnumValue struct {
	Caption     string `json:"caption"`
	Description string `json:"description"`
}

type Dictionary struct {
	Attributes map[string]DictAttribute `json:"attributes"`
}

type DictAttribute struct {
	Type        string `json:"type"`
	Caption     string `json:"caption"`
	Description string `json:"description"`
	IsArray     bool   `json:"is_array"`
}

type Categories struct {
	Attributes map[string]Category `json:"attributes"`
}

type Category struct {
	UID         int    `json:"uid"`
	Caption     string `json:"caption"`
	Description string `json:"description"`
}

var (
	schemaDir = flag.String("schema", "../../ocsf-schema", "Path to OCSF schema directory")
	outputDir = flag.String("output", "../../core/pkg/ocsf", "Output directory for generated code")
	verbose   = flag.Bool("v", false, "Verbose output")

	// Global dictionary loaded once and used throughout generation
	globalDictionary Dictionary
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load categories
	categories, err := loadCategories(filepath.Join(*schemaDir, "categories.json"))
	if err != nil {
		return fmt.Errorf("load categories: %w", err)
	}

	// Load dictionary
	dictionary, err := loadDictionary(filepath.Join(*schemaDir, "dictionary.json"))
	if err != nil {
		return fmt.Errorf("load dictionary: %w", err)
	}

	// Store globally for type inference
	globalDictionary = dictionary

	// Create output directories
	eventsDir := filepath.Join(*outputDir, "events")
	objectsDir := filepath.Join(*outputDir, "objects")
	if err := os.MkdirAll(eventsDir, 0755); err != nil {
		return fmt.Errorf("create events dir: %w", err)
	}
	if err := os.MkdirAll(objectsDir, 0755); err != nil {
		return fmt.Errorf("create objects dir: %w", err)
	}

	// Track generated event class names to avoid duplicate object types
	generatedEventTypes := make(map[string]bool)

	// Collect all object types needed
	objectTypes := make(map[string]bool)

	// Process each category
	schemaEventsDir := filepath.Join(*schemaDir, "events")
	categoryDirs, err := os.ReadDir(schemaEventsDir)
	if err != nil {
		return fmt.Errorf("read events dir: %w", err)
	}

	totalClasses := 0
	for _, categoryDir := range categoryDirs {
		if !categoryDir.IsDir() {
			continue
		}

		categoryName := categoryDir.Name()
		categoryPath := filepath.Join(schemaEventsDir, categoryName)

		// Create category subdirectory
		categoryOutputDir := filepath.Join(eventsDir, categoryName)
		if err := os.MkdirAll(categoryOutputDir, 0755); err != nil {
			return fmt.Errorf("create category dir %s: %w", categoryName, err)
		}

		eventFiles, err := os.ReadDir(categoryPath)
		if err != nil {
			if *verbose {
				fmt.Printf("Warning: cannot read category %s: %v\n", categoryName, err)
			}
			continue
		}

		for _, eventFile := range eventFiles {
			if !strings.HasSuffix(eventFile.Name(), ".json") {
				continue
			}

			// Skip category base files (e.g., iam.json)
			if eventFile.Name() == categoryName+".json" {
				continue
			}

			eventPath := filepath.Join(categoryPath, eventFile.Name())
			class, err := loadEventClass(eventPath)
			if err != nil {
				if *verbose {
					fmt.Printf("Warning: cannot load %s: %v\n", eventPath, err)
				}
				continue
			}

			// Set category name from directory
			class.Category = categoryName

			// Compute full class UID (category_uid * 1000 + class_uid)
			categoryUID := getCategoryUID(categoryName, categories)
			class.UID = categoryUID*1000 + class.UID

			// Track this as a generated event type
			generatedEventTypes[toGoStructName(class.Name)] = true

			// Collect object types from this class
			collectObjectTypes(class, dictionary, objectTypes)

			// Generate class file in category subdirectory
			if err := generateEventClassFile(class, categoryUID, categoryOutputDir, categoryName); err != nil {
				return fmt.Errorf("generate class %s: %w", class.Name, err)
			}

			totalClasses++
			if *verbose {
				fmt.Printf("Generated: %s/%s (UID %d)\n", categoryName, class.Name, class.UID)
			}
		}
	}

	// Generate placeholder object types (excluding event types)
	for eventType := range generatedEventTypes {
		delete(objectTypes, eventType)
	}

	// Generate actual object types from schema
	if err := generateObjects(objectsDir); err != nil {
		return fmt.Errorf("generate objects: %w", err)
	}

	// Generate mappings file for category, class, and activity names
	if err := generateMappings(categories, *outputDir); err != nil {
		return fmt.Errorf("generate mappings: %w", err)
	}

	fmt.Printf("✅ Generated %d OCSF event classes in %s/events/\n", totalClasses, *outputDir)
	fmt.Printf("✅ Generated OCSF object types in %s/objects/\n", *outputDir)
	fmt.Printf("✅ Generated OCSF mappings in %s/mappings.go\n", *outputDir)

	// Validate no generic JSON types were generated
	if err := validateNoGenericTypes(*outputDir); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	fmt.Printf("✅ Type safety validation passed\n")

	return nil
}

func loadCategories(path string) (Categories, error) {
	var cats Categories
	data, err := os.ReadFile(path)
	if err != nil {
		return cats, err
	}
	return cats, json.Unmarshal(data, &cats)
}

func loadDictionary(path string) (Dictionary, error) {
	var dict Dictionary
	data, err := os.ReadFile(path)
	if err != nil {
		return dict, err
	}
	return dict, json.Unmarshal(data, &dict)
}

func loadEventClass(path string) (*EventClass, error) {
	var class EventClass
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &class, json.Unmarshal(data, &class)
}

func getCategoryUID(categoryName string, categories Categories) int {
	// Map category directory names to UIDs
	categoryMap := map[string]string{
		"system":           "system",
		"findings":         "findings",
		"iam":              "iam",
		"network":          "network",
		"discovery":        "discovery",
		"application":      "application",
		"remediation":      "remediation",
		"unmanned_systems": "unmanned_systems",
	}

	if mappedName, ok := categoryMap[categoryName]; ok {
		if cat, ok := categories.Attributes[mappedName]; ok {
			return cat.UID
		}
	}
	return 0
}

// generateFileHeader generates the standard header for generated files
func generateFileHeader() string {
	return `// Code generated by TelHawk OCSF Generator. DO NOT EDIT.
//
// Generated from OCSF Schema: https://github.com/ocsf/ocsf-schema
//
// This generated code contains no original authorship and is released into the
// public domain. For convenience, TelHawk Systems provides it under the TelHawk
// Systems Source Available License (TSSAL) v1.0 as-is, without warranty of any kind.

`
}

func generateEventClassFile(class *EventClass, categoryUID int, outputDir, packageName string) error {
	// Generate Go code
	var buf strings.Builder

	// Check if we need objects import
	needsObjects := false
	for _, rawAttr := range class.Attributes {
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err == nil {
			// Check if any field uses objects package
			for attrName := range class.Attributes {
				if !isBaseField(attrName) {
					goType := inferGoTypeWithObjects(attrName)
					if strings.Contains(goType, "objects.") {
						needsObjects = true
						break
					}
				}
			}
		}
		if needsObjects {
			break
		}
	}

	// Package and imports
	buf.WriteString(generateFileHeader())
	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	buf.WriteString("import (\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"time\"\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf\"\n")
	if needsObjects {
		buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf/objects\"\n")
	}
	buf.WriteString(")\n\n")

	buf.WriteString(fmt.Sprintf("type %s struct {\n", toGoStructName(class.Name)))
	buf.WriteString("\t// Embed base OCSF event\n")
	buf.WriteString("\tocsf.Event\n\n")

	// Add class-specific fields
	buf.WriteString("\t// Class-specific attributes\n")

	// Sort attributes for consistent output
	attrs := make([]string, 0, len(class.Attributes))
	for name := range class.Attributes {
		attrs = append(attrs, name)
	}
	sort.Strings(attrs)

	for _, attrName := range attrs {
		rawAttr := class.Attributes[attrName]

		// Try to parse as AttributeRef
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err != nil {
			// Skip if we can't parse (might be array or other format)
			if *verbose {
				fmt.Printf("  Warning: skipping attribute %s in %s: %v\n", attrName, class.Name, err)
			}
			continue
		}

		// Skip base event fields we already have
		if isBaseField(attrName) {
			continue
		}

		fieldName := toGoFieldName(attrName)
		goType := inferGoTypeWithObjects(attrName)
		jsonTag := attrName

		omit := ""
		if attr.Requirement == "optional" || attr.Requirement == "recommended" {
			omit = ",omitempty"
		}

		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s%s\"`\n", fieldName, goType, jsonTag, omit))
	}

	buf.WriteString("}\n\n")

	// Generate activity enum if present
	if rawActivityAttr, hasEnum := class.Attributes["activity_id"]; hasEnum {
		var activityAttr AttributeRef
		if err := json.Unmarshal(rawActivityAttr, &activityAttr); err == nil && len(activityAttr.Enum) > 0 {
			buf.WriteString(generateActivityEnum(class, activityAttr))
		}
	}

	// Generate constructor helper
	buf.WriteString(generateConstructor(class, categoryUID))

	// Generate Validate method
	buf.WriteString(generateValidator(class))

	// Write to file
	outputPath := filepath.Join(outputDir, toGoFileName(class.Name))
	return os.WriteFile(outputPath, []byte(buf.String()), 0644)
}

func generateActivityEnum(class *EventClass, activityAttr AttributeRef) string {
	var buf strings.Builder

	structName := toGoStructName(class.Name)
	buf.WriteString("const (\n")

	// Sort enum values
	keys := make([]string, 0, len(activityAttr.Enum))
	for k := range activityAttr.Enum {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		enum := activityAttr.Enum[key]
		constName := fmt.Sprintf("%sActivity%s", structName, toGoConstName(enum.Caption))
		buf.WriteString(fmt.Sprintf("\t%s = %s\n", constName, key))
	}

	buf.WriteString(")\n\n")
	return buf.String()
}

func generateConstructor(class *EventClass, categoryUID int) string {
	var buf strings.Builder

	structName := toGoStructName(class.Name)
	funcName := "New" + structName

	buf.WriteString(fmt.Sprintf("func %s(activityID int) *%s {\n", funcName, structName))
	buf.WriteString(fmt.Sprintf("\treturn &%s{\n", structName))
	buf.WriteString("\t\tEvent: ocsf.Event{\n")

	buf.WriteString(fmt.Sprintf("\t\t\tCategoryUID: %d,\n", categoryUID))
	buf.WriteString(fmt.Sprintf("\t\t\tClassUID:    %d,\n", class.UID))
	buf.WriteString("\t\t\tActivityID:  activityID,\n")
	buf.WriteString(fmt.Sprintf("\t\t\tTypeUID:     ocsf.ComputeTypeUID(%d, %d, activityID),\n", categoryUID, class.UID))
	buf.WriteString("\t\t\tTime:        time.Now(),\n")
	buf.WriteString("\t\t\tSeverityID:  ocsf.SeverityUnknown,\n")
	buf.WriteString(fmt.Sprintf("\t\t\tCategory:    %q,\n", class.Category))
	buf.WriteString(fmt.Sprintf("\t\t\tClass:       %q,\n", class.Name))
	buf.WriteString("\t\t\tSeverity:    \"Unknown\",\n")
	buf.WriteString("\t\t\tMetadata: ocsf.Metadata{\n")
	buf.WriteString("\t\t\t\tProduct: ocsf.Product{\n")
	buf.WriteString("\t\t\t\t\tName:   \"TelHawk Stack\",\n")
	buf.WriteString("\t\t\t\t\tVendor: \"TelHawk Systems\",\n")
	buf.WriteString("\t\t\t\t},\n")
	buf.WriteString("\t\t\t\tVersion: \"1.1.0\",\n")
	buf.WriteString("\t\t\t},\n")
	buf.WriteString("\t\t},\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	return buf.String()
}

func generateValidator(class *EventClass) string {
	var buf strings.Builder
	structName := toGoStructName(class.Name)

	buf.WriteString(fmt.Sprintf("// Validate checks that all required fields are properly set\n"))
	buf.WriteString(fmt.Sprintf("func (e *%s) Validate() error {\n", structName))

	// Track if we need to check any fields
	hasValidation := false

	// Check each attribute for required string fields
	for attrName, rawAttr := range class.Attributes {
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err != nil {
			continue
		}

		// Skip optional and recommended fields
		if attr.Requirement == "optional" || attr.Requirement == "recommended" {
			continue
		}

		// Skip base event fields
		if isBaseField(attrName) {
			continue
		}

		// Check if it's a string type that needs validation
		goType := inferGoTypeWithObjects(attrName)
		if goType == "string" {
			hasValidation = true
			fieldName := toGoFieldName(attrName)
			buf.WriteString(fmt.Sprintf("\tif e.%s == \"\" {\n", fieldName))
			buf.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"required field %s is empty\")\n", attrName))
			buf.WriteString("\t}\n")
		}
	}

	if !hasValidation {
		buf.WriteString("\t// No required string fields to validate\n")
	}

	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")

	return buf.String()
}

func generateObjectValidator(object *ObjectSchema) string {
	var buf strings.Builder
	structName := toGoStructName(object.Name)

	buf.WriteString(fmt.Sprintf("// Validate checks that all required fields are properly set\n"))
	buf.WriteString(fmt.Sprintf("func (o *%s) Validate() error {\n", structName))

	// Track if we need to check any fields
	hasValidation := false

	// Check each attribute for required string fields
	for attrName, rawAttr := range object.Attributes {
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err != nil {
			continue
		}

		// Skip optional and recommended fields
		if attr.Requirement == "optional" || attr.Requirement == "recommended" {
			continue
		}

		// Check if it's a string type that needs validation
		goType := inferGoTypeSimple(attrName)
		if goType == "string" {
			hasValidation = true
			fieldName := toGoFieldName(attrName)
			buf.WriteString(fmt.Sprintf("\tif o.%s == \"\" {\n", fieldName))
			buf.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"required field %s is empty\")\n", attrName))
			buf.WriteString("\t}\n")
		}
	}

	if !hasValidation {
		buf.WriteString("\t// No required string fields to validate\n")
	}

	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")

	return buf.String()
}

// Helper functions
func toGoStructName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, "")
}

func toGoFileName(name string) string {
	return name + ".go"
}

func toGoFieldName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, "")
}

func toGoConstName(caption string) string {
	// Remove special characters and convert to CamelCase
	caption = strings.ReplaceAll(caption, " ", "")
	caption = strings.ReplaceAll(caption, "-", "")
	caption = strings.ReplaceAll(caption, "(", "")
	caption = strings.ReplaceAll(caption, ")", "")
	return caption
}

func isBaseField(name string) bool {
	baseFields := map[string]bool{
		"category_uid": true, "class_uid": true, "activity_id": true,
		"type_uid": true, "time": true, "severity_id": true,
		"category": true, "class": true, "activity": true,
		"severity": true, "status": true, "status_id": true,
		"observed_time": true, "metadata": true, "raw": true,
		"enrichments": true, "properties": true,
	}
	return baseFields[name]
}

func inferGoType(attrName string, dict Dictionary, inObjectsPackage bool) string {
	// Check dictionary for type information (PRIMARY SOURCE OF TRUTH)
	if attr, ok := dict.Attributes[attrName]; ok {
		baseType := mapOCSFTypeToGo(attr.Type, attrName, inObjectsPackage)
		if attr.IsArray {
			return "[]" + baseType
		}
		return baseType
	}

	// Fallback to heuristics based on name only if not in dictionary
	if strings.HasSuffix(attrName, "_id") || strings.HasSuffix(attrName, "_uid") {
		return "int"
	}
	if strings.HasSuffix(attrName, "_time") {
		return "int64"
	}
	if strings.Contains(attrName, "count") {
		return "int"
	}

	return "string"
}

func inferGoTypeWithObjects(attrName string) string {
	// Use dictionary-based type inference (PRIMARY SOURCE OF TRUTH)
	// inObjectsPackage = false because this is used in event classes
	return inferGoType(attrName, globalDictionary, false)
}

func mapOCSFTypeToGo(ocsfType string, attrName string, inObjectsPackage bool) string {
	typeMap := map[string]string{
		"string_t":    "string",
		"integer_t":   "int",
		"long_t":      "int64",
		"float_t":     "float64",
		"boolean_t":   "bool",
		"timestamp_t": "int64", // Unix timestamp in milliseconds
	}

	if goType, ok := typeMap[ocsfType]; ok {
		return goType
	}

	// CRITICAL: NO interface{} or map[string]interface{}
	// These lose type safety and violate our policy
	switch ocsfType {
	case "json_t":
		// Log warning - this needs a proper type definition
		if *verbose {
			fmt.Printf("WARNING: Field %s uses json_t - needs proper type (defaulting to string)\n", attrName)
		}
		return "string" // Safe fallback until proper type defined

	case "object_t":
		// Generic object - should be typed
		if *verbose {
			fmt.Printf("WARNING: Field %s uses object_t - needs proper type (defaulting to map[string]string)\n", attrName)
		}
		return "map[string]string" // Safer than interface{}
	}

	// If it's an object type (no _t suffix), use pointer to objects package
	if !strings.HasSuffix(ocsfType, "_t") && ocsfType != "" {
		typeName := toGoStructName(ocsfType)
		if inObjectsPackage {
			// Within objects package, use local type reference
			return "*" + typeName
		}
		// From outside, use objects. prefix
		return "*objects." + typeName
	}

	return "string"
}

func cleanDescription(desc string) string {
	// Remove HTML tags and clean up description
	desc = strings.ReplaceAll(desc, "<code>", "`")
	desc = strings.ReplaceAll(desc, "</code>", "`")
	desc = strings.ReplaceAll(desc, "\n", " ")
	desc = strings.TrimSpace(desc)
	return desc
}

func collectObjectTypes(class *EventClass, dictionary Dictionary, objectTypes map[string]bool) {
	for attrName, rawAttr := range class.Attributes {
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err != nil {
			continue
		}

		// Check if this is an object type from dictionary
		if dictAttr, ok := dictionary.Attributes[attrName]; ok {
			typeName := dictAttr.Type
			// If it's an object type (not a primitive), collect it
			if !strings.HasSuffix(typeName, "_t") && typeName != "" {
				objectTypes[toGoStructName(typeName)] = true
			}
		}
	}
}

func generateObjectPlaceholders(objectTypes map[string]bool, outputDir string) error {
	var buf strings.Builder

	buf.WriteString("// Code generated by ocsf-generator. DO NOT EDIT.\n")
	buf.WriteString("// These are placeholder types for OCSF objects.\n")
	buf.WriteString("// For full OCSF compliance, replace with complete implementations\n")
	buf.WriteString("// from ocsf-schema/objects/ directory.\n\n")
	buf.WriteString("package objects\n\n")

	// Sort for consistent output
	types := make([]string, 0, len(objectTypes))
	for t := range objectTypes {
		types = append(types, t)
	}
	sort.Strings(types)

	buf.WriteString("// Placeholder object types - implement these from OCSF schema\n\n")
	for _, typeName := range types {
		buf.WriteString(fmt.Sprintf("// %s is a placeholder for the OCSF %s object\n", typeName, typeName))
		buf.WriteString(fmt.Sprintf("type %s struct{}\n\n", typeName))
	}

	outputPath := filepath.Join(outputDir, "placeholders.go")
	return os.WriteFile(outputPath, []byte(buf.String()), 0644)
}

func generateObjects(outputDir string) error {
	objectsSchemaDir := filepath.Join(*schemaDir, "objects")
	objectFiles, err := os.ReadDir(objectsSchemaDir)
	if err != nil {
		return fmt.Errorf("read objects dir: %w", err)
	}

	totalObjects := 0
	for _, objectFile := range objectFiles {
		if !strings.HasSuffix(objectFile.Name(), ".json") {
			continue
		}

		objectPath := filepath.Join(objectsSchemaDir, objectFile.Name())
		object, err := loadObjectSchema(objectPath)
		if err != nil {
			if *verbose {
				fmt.Printf("Warning: cannot load object %s: %v\n", objectFile.Name(), err)
			}
			continue
		}

		// Skip base/internal objects (starting with _)
		if strings.HasPrefix(object.Name, "_") {
			continue
		}

		if err := generateObjectFile(object, outputDir); err != nil {
			if *verbose {
				fmt.Printf("Warning: cannot generate object %s: %v\n", object.Name, err)
			}
			continue
		}

		totalObjects++
		if *verbose {
			fmt.Printf("Generated object: %s\n", object.Name)
		}
	}

	return nil
}

type ObjectSchema struct {
	Name        string                     `json:"name"`
	Caption     string                     `json:"caption"`
	Description string                     `json:"description"`
	Extends     string                     `json:"extends"`
	Attributes  map[string]json.RawMessage `json:"attributes"`
}

func loadObjectSchema(path string) (*ObjectSchema, error) {
	var obj ObjectSchema
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &obj, json.Unmarshal(data, &obj)
}

func generateObjectFile(object *ObjectSchema, outputDir string) error {
	var buf strings.Builder

	// Package and imports
	buf.WriteString(generateFileHeader())
	buf.WriteString("package objects\n\n")

	// Check if we need fmt import for validation (required string fields only)
	needsFmt := false
	for attrName, rawAttr := range object.Attributes {
		var attr AttributeRef
		if json.Unmarshal(rawAttr, &attr) == nil {
			if attr.Requirement != "optional" && attr.Requirement != "recommended" {
				goType := inferGoTypeSimple(attrName)
				if goType == "string" {
					needsFmt = true
					break
				}
			}
		}
	}
	if needsFmt {
		buf.WriteString("import \"fmt\"\n\n")
	}

	structName := toGoStructName(object.Name)
	buf.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	// Add embedded parent type if extends is specified
	// Skip abstract types (those starting with underscore)
	if object.Extends != "" && !strings.HasPrefix(object.Extends, "_") {
		parentName := toGoStructName(object.Extends)
		buf.WriteString(fmt.Sprintf("\t%s\n", parentName))
	}

	// Sort attributes for consistent output
	attrs := make([]string, 0, len(object.Attributes))
	for name := range object.Attributes {
		attrs = append(attrs, name)
	}
	sort.Strings(attrs)

	for _, attrName := range attrs {
		rawAttr := object.Attributes[attrName]

		// Try to parse as AttributeRef
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err != nil {
			continue
		}

		fieldName := toGoFieldName(attrName)
		goType := inferGoTypeSimple(attrName)
		jsonTag := attrName

		omit := ""
		if attr.Requirement == "optional" || attr.Requirement == "recommended" {
			omit = ",omitempty"
		}

		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s%s\"`\n", fieldName, goType, jsonTag, omit))
	}

	buf.WriteString("}\n\n")

	// Generate Validate method for objects
	buf.WriteString(generateObjectValidator(object))

	// Write to file
	outputPath := filepath.Join(outputDir, toGoFileName(object.Name))
	return os.WriteFile(outputPath, []byte(buf.String()), 0644)
}

func inferGoTypeSimple(attrName string) string {
	// Use dictionary-based type inference (PRIMARY SOURCE OF TRUTH)
	// inObjectsPackage = true because this is used in object generation
	return inferGoType(attrName, globalDictionary, true)
}

// validateNoGenericTypes ensures no interface{} or map[string]interface{} in generated code
func validateNoGenericTypes(outputDir string) error {
	var violations []string

	// Check events directory
	eventsDir := filepath.Join(outputDir, "events")
	err := filepath.Walk(eventsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Check for forbidden patterns
			if strings.Contains(string(content), "interface{}") {
				violations = append(violations, fmt.Sprintf("%s contains interface{}", path))
			}
			if strings.Contains(string(content), "map[string]interface{}") {
				violations = append(violations, fmt.Sprintf("%s contains map[string]interface{}", path))
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Check objects directory
	objectsDir := filepath.Join(outputDir, "objects")
	err = filepath.Walk(objectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Check for forbidden patterns
			if strings.Contains(string(content), "interface{}") {
				violations = append(violations, fmt.Sprintf("%s contains interface{}", path))
			}
			if strings.Contains(string(content), "map[string]interface{}") {
				violations = append(violations, fmt.Sprintf("%s contains map[string]interface{}", path))
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(violations) > 0 {
		return fmt.Errorf("type safety violations found:\n%s", strings.Join(violations, "\n"))
	}

	return nil
}

// generateMappings creates a comprehensive mappings file for OCSF UIDs to names
func generateMappings(categories Categories, outputDir string) error {
	var buf strings.Builder

	buf.WriteString("// Code generated by ocsf-generator. DO NOT EDIT.\n\n")
	buf.WriteString("package ocsf\n\n")

	// Generate CategoryName function
	buf.WriteString("// CategoryName returns the human-readable category name from category_uid.\n")
	buf.WriteString("func CategoryName(categoryUID int) string {\n")
	buf.WriteString("\tswitch categoryUID {\n")

	// Sort categories for consistent output
	var catKeys []string
	for key := range categories.Attributes {
		catKeys = append(catKeys, key)
	}
	sort.Strings(catKeys)

	for _, key := range catKeys {
		cat := categories.Attributes[key]
		buf.WriteString(fmt.Sprintf("\tcase %d:\n\t\treturn %q\n", cat.UID, cat.Caption))
	}

	buf.WriteString("\tdefault:\n\t\treturn \"Unknown\"\n")
	buf.WriteString("\t}\n}\n\n")

	// Generate ClassName function
	buf.WriteString("// ClassName returns the human-readable class name from class_uid.\n")
	buf.WriteString("func ClassName(classUID int) string {\n")
	buf.WriteString("\tswitch classUID {\n")

	// Collect all class mappings from event schema
	classMap := make(map[int]string)
	schemaEventsDir := filepath.Join(*schemaDir, "events")
	categoryDirs, err := os.ReadDir(schemaEventsDir)
	if err != nil {
		return fmt.Errorf("read events dir: %w", err)
	}

	for _, categoryDir := range categoryDirs {
		if !categoryDir.IsDir() {
			continue
		}

		categoryName := categoryDir.Name()
		categoryPath := filepath.Join(schemaEventsDir, categoryName)
		categoryUID := getCategoryUID(categoryName, categories)

		eventFiles, err := os.ReadDir(categoryPath)
		if err != nil {
			continue
		}

		for _, eventFile := range eventFiles {
			if !strings.HasSuffix(eventFile.Name(), ".json") {
				continue
			}

			if eventFile.Name() == categoryName+".json" {
				continue
			}

			eventPath := filepath.Join(categoryPath, eventFile.Name())
			class, err := loadEventClass(eventPath)
			if err != nil {
				continue
			}

			fullUID := categoryUID*1000 + class.UID
			classMap[fullUID] = class.Caption
		}
	}

	// Sort class UIDs for consistent output
	var classUIDs []int
	for uid := range classMap {
		classUIDs = append(classUIDs, uid)
	}
	sort.Ints(classUIDs)

	for _, uid := range classUIDs {
		buf.WriteString(fmt.Sprintf("\tcase %d:\n\t\treturn %q\n", uid, classMap[uid]))
	}

	buf.WriteString("\tdefault:\n\t\treturn \"Unknown\"\n")
	buf.WriteString("\t}\n}\n\n")

	// Generate ActivityName function
	buf.WriteString("// ActivityName returns the human-readable activity name from class_uid and activity_id.\n")
	buf.WriteString("func ActivityName(classUID int, activityID int) string {\n")
	buf.WriteString("\t// Format: class_uid * 100 + activity_id for lookup key\n")
	buf.WriteString("\tkey := classUID*100 + activityID\n")
	buf.WriteString("\tswitch key {\n")

	// Collect activity mappings from event schemas
	activityMap := make(map[int]string)

	for _, categoryDir := range categoryDirs {
		if !categoryDir.IsDir() {
			continue
		}

		categoryName := categoryDir.Name()
		categoryPath := filepath.Join(schemaEventsDir, categoryName)
		categoryUID := getCategoryUID(categoryName, categories)

		eventFiles, err := os.ReadDir(categoryPath)
		if err != nil {
			continue
		}

		for _, eventFile := range eventFiles {
			if !strings.HasSuffix(eventFile.Name(), ".json") {
				continue
			}

			if eventFile.Name() == categoryName+".json" {
				continue
			}

			eventPath := filepath.Join(categoryPath, eventFile.Name())
			class, err := loadEventClass(eventPath)
			if err != nil {
				continue
			}

			fullClassUID := categoryUID*1000 + class.UID

			// Extract activity_id enum if present
			if rawActivityAttr, ok := class.Attributes["activity_id"]; ok {
				var activityAttr AttributeRef
				if err := json.Unmarshal(rawActivityAttr, &activityAttr); err == nil {
					for actIDStr, enumVal := range activityAttr.Enum {
						var actID int
						fmt.Sscanf(actIDStr, "%d", &actID)
						key := fullClassUID*100 + actID
						activityMap[key] = enumVal.Caption
					}
				}
			}
		}
	}

	// Sort activity keys for consistent output
	var activityKeys []int
	for key := range activityMap {
		activityKeys = append(activityKeys, key)
	}
	sort.Ints(activityKeys)

	for _, key := range activityKeys {
		buf.WriteString(fmt.Sprintf("\tcase %d:\n\t\treturn %q\n", key, activityMap[key]))
	}

	buf.WriteString("\tdefault:\n\t\treturn \"\"\n")
	buf.WriteString("\t}\n}\n")

	outputPath := filepath.Join(outputDir, "mappings.go")
	return os.WriteFile(outputPath, []byte(buf.String()), 0644)
}
