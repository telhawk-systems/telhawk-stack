package service

import (
	crypto_rand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/validator"
	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

// generateID creates a random UUID v4.
func generateID() string {
	buf := make([]byte, 16)
	if _, err := crypto_rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return formatUUID(buf)
}

// formatUUID formats a 16-byte buffer as a UUID string.
func formatUUID(b []byte) string {
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexBytes := make([]byte, 36)
	hex.Encode(hexBytes[0:8], b[0:4])
	hexBytes[8] = '-'
	hex.Encode(hexBytes[9:13], b[4:6])
	hexBytes[13] = '-'
	hex.Encode(hexBytes[14:18], b[6:8])
	hexBytes[18] = '-'
	hex.Encode(hexBytes[19:23], b[8:10])
	hexBytes[23] = '-'
	hex.Encode(hexBytes[24:], b[10:16])
	return string(hexBytes)
}

// validateSavedSearchInput validates the name and query for a saved search.
func validateSavedSearchInput(name string, queryMap map[string]interface{}) error {
	// Validate name is not empty or whitespace
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return fmt.Errorf("%w: saved search name cannot be empty or whitespace", ErrValidationFailed)
	}

	// Validate query structure by unmarshaling into the Query model
	queryJSON, err := json.Marshal(queryMap)
	if err != nil {
		return fmt.Errorf("%w: invalid query format: %w", ErrValidationFailed, err)
	}

	var query model.Query
	if err := json.Unmarshal(queryJSON, &query); err != nil {
		return fmt.Errorf("%w: invalid query structure: %w", ErrValidationFailed, err)
	}

	// Validate query using the query validator
	v := validator.NewQueryValidator()
	if err := v.Validate(&query); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	return nil
}
