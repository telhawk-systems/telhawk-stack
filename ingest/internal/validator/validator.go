package validator

import (
	"context"

	"github.com/telhawk-systems/telhawk-stack/common/ocsf"
)

// Validator defines contract for OCSF schema validation units.
type Validator interface {
	Validate(ctx context.Context, event *ocsf.Event) error
	Supports(class string) bool
}

// Chain applies a list of validators sequentially.
type Chain struct {
	validators []Validator
}

// NewChain constructs a validator chain.
func NewChain(validators ...Validator) *Chain {
	return &Chain{validators: validators}
}

// Validate executes validators in order until an error occurs.
func (c *Chain) Validate(ctx context.Context, event *ocsf.Event) error {
	if c == nil {
		return nil
	}
	for _, v := range c.validators {
		if v.Supports(event.Class) {
			if err := v.Validate(ctx, event); err != nil {
				return err
			}
		}
	}
	return nil
}
