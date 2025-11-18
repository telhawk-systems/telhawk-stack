package main

import "encoding/json"

// Schema structures matching OCSF JSON format

// EventClass represents an OCSF event class definition.
type EventClass struct {
	UID         int                        `json:"uid"`
	Caption     string                     `json:"caption"`
	Description string                     `json:"description"`
	Extends     string                     `json:"extends"`
	Name        string                     `json:"name"`
	Attributes  map[string]json.RawMessage `json:"attributes"`
	Category    string                     `json:"category"`
}

// AttributeRef represents an attribute reference in OCSF.
type AttributeRef struct {
	Group       string               `json:"group"`
	Requirement string               `json:"requirement"`
	Description string               `json:"description"`
	Enum        map[string]EnumValue `json:"enum"`
	Sibling     string               `json:"sibling"`
}

// EnumValue represents an enumeration value in OCSF.
type EnumValue struct {
	Caption     string `json:"caption"`
	Description string `json:"description"`
}

// Dictionary represents the OCSF dictionary with attributes and types.
type Dictionary struct {
	Attributes map[string]DictAttribute `json:"attributes"`
	Types      TypesSection             `json:"types"`
}

// TypesSection contains type definitions from the dictionary.
type TypesSection struct {
	Attributes map[string]TypeDefinition `json:"attributes"`
}

// TypeDefinition represents a type definition in the dictionary.
type TypeDefinition struct {
	Type        string `json:"type"`
	Caption     string `json:"caption"`
	Description string `json:"description"`
	TypeName    string `json:"type_name"`
}

// DictAttribute represents an attribute definition in the dictionary.
type DictAttribute struct {
	Type        string `json:"type"`
	Caption     string `json:"caption"`
	Description string `json:"description"`
	IsArray     bool   `json:"is_array"`
}

// Categories represents the collection of OCSF categories.
type Categories struct {
	Attributes map[string]Category `json:"attributes"`
}

// Category represents an OCSF event category.
type Category struct {
	UID         int    `json:"uid"`
	Caption     string `json:"caption"`
	Description string `json:"description"`
}

// ObjectSchema represents an OCSF object definition.
type ObjectSchema struct {
	Name        string                     `json:"name"`
	Caption     string                     `json:"caption"`
	Description string                     `json:"description"`
	Extends     string                     `json:"extends"`
	Attributes  map[string]json.RawMessage `json:"attributes"`
}
