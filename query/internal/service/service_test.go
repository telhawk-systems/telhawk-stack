package service

import (
"testing"
"time"

"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

func TestBuildOpenSearchQuery(t *testing.T) {
svc := &QueryService{}

tests := []struct {
name     string
req      *models.SearchRequest
wantType string
}{
{
name: "match_all query",
req: &models.SearchRequest{
Query: "*",
},
wantType: "match_all",
},
{
name: "query_string with text",
req: &models.SearchRequest{
Query: "severity:high",
},
wantType: "bool",
},
{
name: "query with time range",
req: &models.SearchRequest{
Query: "severity:high",
TimeRange: &models.TimeRange{
From: time.Unix(1698796800, 0),
To:   time.Unix(1698883200, 0),
},
},
wantType: "bool",
},
{
name: "query with search_after",
req: &models.SearchRequest{
Query: "severity:high",
SearchAfter: []interface{}{1698883200, "doc123"},
Sort: &models.SortOptions{
Field: "time",
Order: "desc",
},
},
wantType: "bool",
},
{
name: "query with aggregations",
req: &models.SearchRequest{
Query: "*",
Aggregations: map[string]models.AggregationRequest{
"severity_count": {
Type:  "terms",
Field: "severity",
Size:  5,
},
"events_over_time": {
Type:  "date_histogram",
Field: "time",
Opts: map[string]interface{}{
"interval": "1h",
},
},
},
},
wantType: "match_all",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
query := svc.buildOpenSearchQuery(tt.req)
if query == nil {
t.Fatal("expected query to be non-nil")
}
if _, ok := query["query"]; !ok {
t.Error("expected query to contain 'query' key")
}
if tt.req.SearchAfter != nil {
if sa, ok := query["search_after"]; !ok {
t.Error("expected query to contain 'search_after'")
} else if len(sa.([]interface{})) != len(tt.req.SearchAfter) {
t.Error("search_after length mismatch")
}
}
if tt.req.Aggregations != nil {
if _, ok := query["aggs"]; !ok {
t.Error("expected query to contain 'aggs'")
}
}
})
}
}

func TestGenerateID(t *testing.T) {
id1 := generateID()
id2 := generateID()

if id1 == id2 {
t.Error("expected unique IDs")
}

if len(id1) != 36 {
t.Errorf("expected UUID length 36, got %d", len(id1))
}
}

func TestAggregationBuilding(t *testing.T) {
svc := &QueryService{}
	
tests := []struct {
name    string
aggReq  models.AggregationRequest
aggName string
}{
{
name: "terms aggregation",
aggName: "top_users",
aggReq: models.AggregationRequest{
Type:  "terms",
Field: "user.name",
Size:  20,
},
},
{
name: "date histogram",
aggName: "timeline",
aggReq: models.AggregationRequest{
Type:  "date_histogram",
Field: "time",
Opts: map[string]interface{}{
"interval": "5m",
},
},
},
{
name: "metric aggregation",
aggName: "avg_latency",
aggReq: models.AggregationRequest{
Type:  "avg",
Field: "duration_ms",
},
},
}
	
for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
req := &models.SearchRequest{
Query: "*",
Aggregations: map[string]models.AggregationRequest{
tt.aggName: tt.aggReq,
},
}
query := svc.buildOpenSearchQuery(req)
			
if aggs, ok := query["aggs"]; !ok {
t.Error("expected query to contain 'aggs'")
} else {
aggsMap := aggs.(map[string]interface{})
if _, ok := aggsMap[tt.aggName]; !ok {
t.Errorf("expected aggregation '%s' to exist", tt.aggName)
}
}
})
}
}
