package validator

import (
	"context"
	"fmt"

	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
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
	if event.Schema.Namespace == "" {
		return fmt.Errorf("missing schema namespace")
	}
	if event.Schema.Version == "" {
		return fmt.Errorf("missing schema version")
	}
	if event.Category == "" {
		return fmt.Errorf("missing category")
	}
	if event.ObservedTime.IsZero() {
		return fmt.Errorf("missing observed_time")
	}
	return nil
}
