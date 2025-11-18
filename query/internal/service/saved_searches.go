package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
	"github.com/telhawk-systems/telhawk-stack/query/internal/translator"
	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

// ListSavedSearches returns all saved searches (not showing hidden by default).
func (s *QueryService) ListSavedSearches(ctx context.Context) ([]models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	// Default: do not show hidden
	return s.repo.ListLatest(ctx, false)
}

// ListSavedSearchesWithOptions returns saved searches with options to show all.
func (s *QueryService) ListSavedSearchesWithOptions(ctx context.Context, showAll bool) ([]models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.ListLatest(ctx, showAll)
}

// ListSavedSearchesPaged returns paginated saved searches.
func (s *QueryService) ListSavedSearchesPaged(ctx context.Context, showAll bool, page, size int) ([]models.SavedSearch, int, error) {
	if s.repo == nil {
		return nil, 0, fmt.Errorf("repository not configured")
	}
	return s.repo.ListLatestPaged(ctx, showAll, page, size)
}

// ListSavedSearchesAfter returns saved searches using cursor-based pagination.
func (s *QueryService) ListSavedSearchesAfter(ctx context.Context, showAll bool, cursorCreatedAt *time.Time, cursorVersionID string, size int) ([]models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.ListLatestAfter(ctx, showAll, cursorCreatedAt, cursorVersionID, size)
}

// CreateSavedSearch creates a new saved search.
func (s *QueryService) CreateSavedSearch(ctx context.Context, req *models.SavedSearchCreateRequest) (*models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	// Validate input
	if err := validateSavedSearchInput(req.Name, req.Query); err != nil {
		return nil, err
	}

	id := generateID()
	versionID := generateID()
	now := time.Now().UTC()
	var owner *string
	if !req.IsGlobal {
		// owner is the creator by default
		owner = &req.CreatedBy
	}
	saved := models.SavedSearch{
		ID:        id,
		VersionID: versionID,
		OwnerID:   owner,
		CreatedBy: req.CreatedBy,
		Name:      req.Name,
		Query:     req.Query,
		Filters:   req.Filters,
		IsGlobal:  req.IsGlobal,
		CreatedAt: now,
	}
	if err := s.repo.InsertVersion(ctx, &saved); err != nil {
		return nil, err
	}
	return &saved, nil
}

// GetSavedSearch retrieves the latest version of a saved search by ID.
func (s *QueryService) GetSavedSearch(ctx context.Context, id string) (*models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetLatest(ctx, id)
}

// UpdateSavedSearch creates a new version of a saved search with updated values.
func (s *QueryService) UpdateSavedSearch(ctx context.Context, id string, req *models.SavedSearchUpdateRequest) (*models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	latest, err := s.repo.GetLatest(ctx, id)
	if err != nil {
		return nil, err
	}

	// Determine the final name and query after update
	finalName := latest.Name
	if req.Name != nil {
		finalName = *req.Name
	}
	finalQuery := latest.Query
	if req.Query != nil {
		finalQuery = req.Query
	}

	// Validate the updated values
	if err := validateSavedSearchInput(finalName, finalQuery); err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		latest.Name = *req.Name
	}
	if req.Query != nil {
		latest.Query = req.Query
	}
	if req.Filters != nil {
		latest.Filters = req.Filters
	}
	latest.VersionID = generateID()
	latest.CreatedBy = req.CreatedBy
	latest.CreatedAt = time.Now().UTC()
	latest.DisabledAt = nil
	// Do not change hidden state on update
	if err := s.repo.InsertVersion(ctx, latest); err != nil {
		return nil, err
	}
	return latest, nil
}

// DeleteSavedSearch marks a saved search as hidden.
func (s *QueryService) DeleteSavedSearch(ctx context.Context, id string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}
	latest, err := s.repo.GetLatest(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	latest.VersionID = generateID()
	latest.HiddenAt = &now
	latest.CreatedAt = now
	if err := s.repo.InsertVersion(ctx, latest); err != nil {
		return err
	}
	return nil
}

// DisableSavedSearch marks a saved search as disabled.
func (s *QueryService) DisableSavedSearch(ctx context.Context, id, userID string) (*models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	latest, err := s.repo.GetLatest(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	latest.VersionID = generateID()
	latest.CreatedBy = userID
	latest.CreatedAt = now
	latest.DisabledAt = &now
	if err := s.repo.InsertVersion(ctx, latest); err != nil {
		return nil, err
	}
	return latest, nil
}

// EnableSavedSearch re-enables a disabled saved search.
func (s *QueryService) EnableSavedSearch(ctx context.Context, id, userID string) (*models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	latest, err := s.repo.GetLatest(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	latest.VersionID = generateID()
	latest.CreatedBy = userID
	latest.CreatedAt = now
	latest.DisabledAt = nil
	if err := s.repo.InsertVersion(ctx, latest); err != nil {
		return nil, err
	}
	return latest, nil
}

// HideSavedSearch marks a saved search as hidden.
func (s *QueryService) HideSavedSearch(ctx context.Context, id, userID string) (*models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	latest, err := s.repo.GetLatest(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	latest.VersionID = generateID()
	latest.CreatedBy = userID
	latest.CreatedAt = now
	latest.HiddenAt = &now
	if err := s.repo.InsertVersion(ctx, latest); err != nil {
		return nil, err
	}
	return latest, nil
}

// UnhideSavedSearch makes a hidden saved search visible again.
func (s *QueryService) UnhideSavedSearch(ctx context.Context, id, userID string) (*models.SavedSearch, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	latest, err := s.repo.GetLatest(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	latest.VersionID = generateID()
	latest.CreatedBy = userID
	latest.CreatedAt = now
	latest.HiddenAt = nil
	if err := s.repo.InsertVersion(ctx, latest); err != nil {
		return nil, err
	}
	return latest, nil
}

// RunSavedSearch executes a saved search and returns results.
func (s *QueryService) RunSavedSearch(ctx context.Context, id string) (*models.SearchResponse, error) {
	saved, err := s.GetSavedSearch(ctx, id)
	if err != nil {
		return nil, err
	}
	if saved.DisabledAt != nil {
		return nil, ErrSearchDisabled
	}

	// Unmarshal saved query into Query model
	queryJSON, err := json.Marshal(saved.Query)
	if err != nil {
		return nil, fmt.Errorf("marshal saved query: %w", err)
	}
	var query model.Query
	if err := json.Unmarshal(queryJSON, &query); err != nil {
		return nil, fmt.Errorf("unmarshal saved query: %w", err)
	}

	// Translate Phase 2 query to OpenSearch DSL
	translator := translator.NewOpenSearchTranslator()
	osQuery, err := translator.Translate(&query)
	if err != nil {
		return nil, fmt.Errorf("translate saved query: %w", err)
	}

	// Execute using OpenSearch client
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(osQuery); err != nil {
		return nil, fmt.Errorf("encode opensearch query: %w", err)
	}
	res, err := s.osClient.Client().Search(
		s.osClient.Client().Search.WithContext(ctx),
		s.osClient.Client().Search.WithIndex(s.osClient.Index()+"*"),
		s.osClient.Client().Search.WithBody(&buf),
		s.osClient.Client().Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("search error: %s", res.String())
	}
	var searchResult struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	}
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	results := make([]map[string]interface{}, 0, len(searchResult.Hits.Hits))
	for _, hit := range searchResult.Hits.Hits {
		results = append(results, hit.Source)
	}
	return &models.SearchResponse{
		RequestID:    generateID(),
		LatencyMS:    0,
		ResultCount:  len(results),
		TotalMatches: searchResult.Hits.Total.Value,
		Results:      results,
		Aggregations: searchResult.Aggregations,
	}, nil
}
