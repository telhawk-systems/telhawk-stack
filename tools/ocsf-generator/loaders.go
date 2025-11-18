package main

import (
	"encoding/json"
	"os"
)

// loadCategories loads the OCSF categories from the specified file.
func loadCategories(path string) (Categories, error) {
	var cats Categories
	data, err := os.ReadFile(path)
	if err != nil {
		return cats, err
	}
	return cats, json.Unmarshal(data, &cats)
}

// loadDictionary loads the OCSF dictionary from the specified file.
func loadDictionary(path string) (Dictionary, error) {
	var dict Dictionary
	data, err := os.ReadFile(path)
	if err != nil {
		return dict, err
	}
	return dict, json.Unmarshal(data, &dict)
}

// loadEventClass loads an OCSF event class from the specified file.
func loadEventClass(path string) (*EventClass, error) {
	var class EventClass
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &class, json.Unmarshal(data, &class)
}

// loadObjectSchema loads an OCSF object schema from the specified file.
func loadObjectSchema(path string) (*ObjectSchema, error) {
	var obj ObjectSchema
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &obj, json.Unmarshal(data, &obj)
}

// getCategoryUID returns the UID for the given category name.
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
