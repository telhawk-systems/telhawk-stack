package validator

import (
	"context"
	"fmt"

	"github.com/telhawk-systems/telhawk-stack/common/ocsf"
)

// BasicValidator ensures minimal OCSF required fields exist.
type BasicValidator struct{}

// Supports returns true for all classes.
func (BasicValidator) Supports(class string) bool {
	return class != ""
}

// Validate performs structural validation.
func (BasicValidator) Validate(ctx context.Context, event *ocsf.Event) error {
	_ = ctx
	if event.Class == "" {
		return fmt.Errorf("missing class")
	}
	if event.ClassUID == 0 && event.Class != "base_event" {
		return fmt.Errorf("missing class_uid")
	}
	if event.Category == "" {
		return fmt.Errorf("missing category")
	}
	if event.Time.IsZero() {
		return fmt.Errorf("missing time")
	}
	if event.Metadata.Product.Name == "" {
		return fmt.Errorf("missing metadata.product.name")
	}
	if event.Metadata.Version == "" {
		return fmt.Errorf("missing metadata.version")
	}
	return nil
}
