package main

import "strings"

// toGoStructName converts an OCSF name to a Go struct name (PascalCase).
func toGoStructName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, "")
}

// toGoFileName converts an OCSF name to a Go filename.
func toGoFileName(name string) string {
	return name + ".go"
}

// toGoFieldName converts an OCSF attribute name to a Go field name (PascalCase).
func toGoFieldName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, "")
}

// toGoConstName converts a caption to a Go constant name.
func toGoConstName(caption string) string {
	// Remove special characters and convert to CamelCase
	caption = strings.ReplaceAll(caption, " ", "")
	caption = strings.ReplaceAll(caption, "-", "")
	caption = strings.ReplaceAll(caption, "(", "")
	caption = strings.ReplaceAll(caption, ")", "")
	return caption
}

// isBaseField returns true if the field name is a base OCSF event field.
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

// cleanDescription cleans up OCSF descriptions for Go comments.
func cleanDescription(desc string) string {
	// Remove HTML tags and clean up description
	desc = strings.ReplaceAll(desc, "<code>", "`")
	desc = strings.ReplaceAll(desc, "</code>", "`")
	desc = strings.ReplaceAll(desc, "\n", " ")
	desc = strings.TrimSpace(desc)
	return desc
}
