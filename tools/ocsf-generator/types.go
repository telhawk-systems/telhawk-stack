package main

import (
	"fmt"
	"strings"
)

// inferGoType determines the Go type for an OCSF attribute.
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

// inferGoTypeWithObjects infers Go types for event class fields (uses objects package).
func inferGoTypeWithObjects(attrName string) string {
	// Use dictionary-based type inference (PRIMARY SOURCE OF TRUTH)
	// inObjectsPackage = false because this is used in event classes
	return inferGoType(attrName, globalDictionary, false)
}

// inferGoTypeSimple infers Go types for object fields (uses local types).
func inferGoTypeSimple(attrName string) string {
	// Use dictionary-based type inference (PRIMARY SOURCE OF TRUTH)
	// inObjectsPackage = true because this is used in object generation
	return inferGoType(attrName, globalDictionary, true)
}

// mapOCSFTypeToGo maps an OCSF type string to a Go type.
func mapOCSFTypeToGo(ocsfType string, attrName string, inObjectsPackage bool) string {
	// Resolve extended types recursively
	resolvedType := resolveExtendedType(ocsfType, 0)

	// Base primitive type mappings
	typeMap := map[string]string{
		"string_t":    "string",
		"integer_t":   "int",
		"long_t":      "int64",
		"float_t":     "float64",
		"boolean_t":   "bool",
		"timestamp_t": "int64", // Unix timestamp in milliseconds
	}

	if goType, ok := typeMap[resolvedType]; ok {
		return goType
	}

	// CRITICAL: NO interface{} or map[string]interface{}
	// These lose type safety and violate our policy
	switch resolvedType {
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
	if !strings.HasSuffix(resolvedType, "_t") && resolvedType != "" {
		typeName := toGoStructName(resolvedType)
		if inObjectsPackage {
			// Within objects package, use local type reference
			return "*" + typeName
		}
		// From outside, use objects. prefix
		return "*objects." + typeName
	}

	return "string"
}

// resolveExtendedType recursively resolves extended types to their base primitive types.
// For example: port_t -> integer_t -> int
// maxDepth prevents infinite recursion.
func resolveExtendedType(ocsfType string, depth int) string {
	const maxDepth = 10
	if depth > maxDepth {
		if *verbose {
			fmt.Printf("WARNING: Max recursion depth reached resolving type %s\n", ocsfType)
		}
		return ocsfType
	}

	// Check if this is an extended type defined in dictionary.types
	if typeDef, ok := globalDictionary.Types.Attributes[ocsfType]; ok {
		// Extended type found, recursively resolve its underlying type
		if typeDef.Type != "" && typeDef.Type != ocsfType {
			return resolveExtendedType(typeDef.Type, depth+1)
		}
	}

	// No further resolution needed
	return ocsfType
}

// collectObjectTypes collects all object types used in an event class.
func collectObjectTypes(class *EventClass, dictionary Dictionary, objectTypes map[string]bool) {
	for attrName := range class.Attributes {
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
