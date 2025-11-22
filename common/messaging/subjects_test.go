package messaging

import (
	"strings"
	"testing"
)

func TestSubjectConstants_Defined(t *testing.T) {
	// Verify all subject constants are non-empty
	subjects := map[string]string{
		"SubjectSearchJobsQuery":        SubjectSearchJobsQuery,
		"SubjectSearchJobsCorrelate":    SubjectSearchJobsCorrelate,
		"SubjectSearchResultsQuery":     SubjectSearchResultsQuery,
		"SubjectSearchResultsCorrelate": SubjectSearchResultsCorrelate,
		"SubjectRespondAlertsCreated":   SubjectRespondAlertsCreated,
		"SubjectRespondAlertsUpdated":   SubjectRespondAlertsUpdated,
		"SubjectRespondCasesCreated":    SubjectRespondCasesCreated,
		"SubjectRespondCasesUpdated":    SubjectRespondCasesUpdated,
		"SubjectRespondCasesAssigned":   SubjectRespondCasesAssigned,
	}

	for name, value := range subjects {
		if value == "" {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestSubjectConstants_FollowNamingConvention(t *testing.T) {
	// Subjects should follow the pattern: {domain}.{action}.{resource}
	subjects := []string{
		SubjectSearchJobsQuery,
		SubjectSearchJobsCorrelate,
		SubjectSearchResultsQuery,
		SubjectSearchResultsCorrelate,
		SubjectRespondAlertsCreated,
		SubjectRespondAlertsUpdated,
		SubjectRespondCasesCreated,
		SubjectRespondCasesUpdated,
		SubjectRespondCasesAssigned,
	}

	for _, subject := range subjects {
		parts := strings.Split(subject, ".")
		if len(parts) < 3 {
			t.Errorf("subject %q does not follow {domain}.{action}.{resource} pattern", subject)
		}
	}
}

func TestSubjectConstants_SearchDomain(t *testing.T) {
	// Verify search subjects start with "search."
	searchSubjects := []string{
		SubjectSearchJobsQuery,
		SubjectSearchJobsCorrelate,
		SubjectSearchResultsQuery,
		SubjectSearchResultsCorrelate,
	}

	for _, subject := range searchSubjects {
		if !strings.HasPrefix(subject, "search.") {
			t.Errorf("search subject %q should start with 'search.'", subject)
		}
	}
}

func TestSubjectConstants_RespondDomain(t *testing.T) {
	// Verify respond subjects start with "respond."
	respondSubjects := []string{
		SubjectRespondAlertsCreated,
		SubjectRespondAlertsUpdated,
		SubjectRespondCasesCreated,
		SubjectRespondCasesUpdated,
		SubjectRespondCasesAssigned,
	}

	for _, subject := range respondSubjects {
		if !strings.HasPrefix(subject, "respond.") {
			t.Errorf("respond subject %q should start with 'respond.'", subject)
		}
	}
}

func TestQueueConstants_Defined(t *testing.T) {
	// Verify all queue group constants are non-empty
	queues := map[string]string{
		"QueueSearchWorkers":  QueueSearchWorkers,
		"QueueRespondWorkers": QueueRespondWorkers,
		"QueueWebWorkers":     QueueWebWorkers,
	}

	for name, value := range queues {
		if value == "" {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestQueueConstants_NoDots(t *testing.T) {
	// Queue names should not contain dots (they're not subjects)
	queues := []string{
		QueueSearchWorkers,
		QueueRespondWorkers,
		QueueWebWorkers,
	}

	for _, queue := range queues {
		if strings.Contains(queue, ".") {
			t.Errorf("queue name %q should not contain dots", queue)
		}
	}
}

func TestSearchQueryResultSubject(t *testing.T) {
	tests := []struct {
		name     string
		queryID  string
		expected string
	}{
		{
			name:     "simple query ID",
			queryID:  "abc123",
			expected: "search.results.query.abc123",
		},
		{
			name:     "UUID-style query ID",
			queryID:  "550e8400-e29b-41d4-a716-446655440000",
			expected: "search.results.query.550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "empty query ID",
			queryID:  "",
			expected: "search.results.query.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SearchQueryResultSubject(tt.queryID)
			if result != tt.expected {
				t.Errorf("SearchQueryResultSubject(%q) = %q, want %q", tt.queryID, result, tt.expected)
			}
		})
	}
}

func TestSearchQueryResultSubject_StartsWithBase(t *testing.T) {
	// The result should always start with the base subject
	queryID := "test-query"
	result := SearchQueryResultSubject(queryID)

	if !strings.HasPrefix(result, SubjectSearchResultsQuery) {
		t.Errorf("SearchQueryResultSubject result %q should start with %q", result, SubjectSearchResultsQuery)
	}
}

func TestSearchQueryResultSubject_ContainsQueryID(t *testing.T) {
	// The result should contain the query ID
	queryID := "unique-query-12345"
	result := SearchQueryResultSubject(queryID)

	if !strings.Contains(result, queryID) {
		t.Errorf("SearchQueryResultSubject result %q should contain query ID %q", result, queryID)
	}
}
