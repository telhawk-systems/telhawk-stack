// Package messaging defines standard subject names for TelHawk message bus.
package messaging

// Subject constants for TelHawk message bus.
// Follow the pattern: {domain}.{action}.{resource}
const (
	// Search job subjects - requests for query/correlation execution
	SubjectSearchJobsQuery     = "search.jobs.query"     // Ad-hoc search requests
	SubjectSearchJobsCorrelate = "search.jobs.correlate" // Correlation evaluation requests

	// Search result subjects - responses from search service
	SubjectSearchResultsQuery     = "search.results.query"     // Ad-hoc query results (append .{id} for specific query)
	SubjectSearchResultsCorrelate = "search.results.correlate" // Correlation match results

	// Alert lifecycle subjects - published by respond service
	SubjectRespondAlertsCreated = "respond.alerts.created" // New alert created
	SubjectRespondAlertsUpdated = "respond.alerts.updated" // Alert status changed

	// Case lifecycle subjects - published by respond service
	SubjectRespondCasesCreated  = "respond.cases.created"  // New case opened
	SubjectRespondCasesUpdated  = "respond.cases.updated"  // Case status changed
	SubjectRespondCasesAssigned = "respond.cases.assigned" // Case assigned to analyst
)

// Queue group names for load-balanced consumers.
// Workers in the same queue group share messages (each message processed once).
const (
	QueueSearchWorkers  = "search-workers"  // Pool of search/correlation workers
	QueueRespondWorkers = "respond-workers" // Pool of alert/case processors
	QueueWebWorkers     = "web-workers"     // Pool of web notification handlers
)

// SearchQueryResultSubject returns the subject for a specific query's results.
// Example: search.results.query.abc123
func SearchQueryResultSubject(queryID string) string {
	return SubjectSearchResultsQuery + "." + queryID
}
