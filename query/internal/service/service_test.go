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
