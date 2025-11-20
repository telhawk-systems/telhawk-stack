package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAlertingClient(t *testing.T) {
	client := NewAlertingClient("http://localhost:8085")

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8085", client.baseURL)
	assert.NotNil(t, client.client)
	assert.Equal(t, 30*time.Second, client.client.Timeout)
}

func TestListAlerts_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/alerts", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AlertsResponse{
			Alerts: []Alert{
				{
					ID:         "alert-1",
					Index:      "telhawk-alerts-2024.11.20",
					Severity:   "High",
					SeverityID: 4,
					Time:       1700000000000,
					FindingInfo: struct {
						Title       string   `json:"title"`
						Description string   `json:"desc"`
						UID         string   `json:"uid"`
						CreatedTime int64    `json:"created_time"`
						Types       []string `json:"types"`
					}{
						Title:       "Brute Force Attack Detected",
						Description: "Multiple failed login attempts",
						UID:         "finding-123",
						CreatedTime: 1700000000000,
						Types:       []string{"brute_force"},
					},
					DetectionSchemaID:        "rule-1",
					DetectionSchemaVersionID: "v1",
				},
			},
			Pagination: map[string]interface{}{
				"total": 1,
				"page":  1,
			},
		})
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	resp, err := client.ListAlerts(testToken, 1, 10, nil)

	require.NoError(t, err)
	assert.Len(t, resp.Alerts, 1)
	assert.Equal(t, "alert-1", resp.Alerts[0].ID)
	assert.Equal(t, "Brute Force Attack Detected", resp.Alerts[0].FindingInfo.Title)
	assert.Equal(t, "High", resp.Alerts[0].Severity)
}

func TestListAlerts_WithFilters(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "high", r.URL.Query().Get("severity"))
		assert.Equal(t, "rule-123", r.URL.Query().Get("detection_id"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AlertsResponse{
			Alerts:     []Alert{},
			Pagination: map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	filters := map[string]string{
		"severity":     "high",
		"detection_id": "rule-123",
	}

	_, err := client.ListAlerts(testToken, 1, 10, filters)
	assert.NoError(t, err)
}

func TestAlert_HelperMethods(t *testing.T) {
	alert := Alert{
		Time: 1700000000000,
		FindingInfo: struct {
			Title       string   `json:"title"`
			Description string   `json:"desc"`
			UID         string   `json:"uid"`
			CreatedTime int64    `json:"created_time"`
			Types       []string `json:"types"`
		}{
			Title: "Port Scan Detection",
		},
		RawData: struct {
			EventCount    int      `json:"event_count,omitempty"`
			DistinctCount int      `json:"distinct_count,omitempty"`
			GroupKey      string   `json:"group_key,omitempty"`
			GroupBy       []string `json:"group_by,omitempty"`
			Threshold     int      `json:"threshold,omitempty"`
			TimeWindow    string   `json:"time_window,omitempty"`
		}{
			EventCount:    150,
			DistinctCount: 50,
		},
	}

	assert.Equal(t, "Port Scan Detection", alert.DetectionName())
	assert.Equal(t, 150, alert.EventCount())
	assert.Equal(t, 50, alert.DistinctCount())

	triggeredAt := alert.TriggeredAt()
	assert.Equal(t, int64(1700000000), triggeredAt.Unix())
}

func TestGetAlert_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/alerts/alert-456", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Alert{
			ID:         "alert-456",
			Severity:   "Critical",
			SeverityID: 5,
		})
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	alert, err := client.GetAlert(testToken, "alert-456")

	require.NoError(t, err)
	assert.Equal(t, "alert-456", alert.ID)
	assert.Equal(t, "Critical", alert.Severity)
}

func TestGetAlert_NotFound(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"alert not found"}`))
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	alert, err := client.GetAlert(testToken, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, alert)
	assert.Contains(t, err.Error(), "failed to get alert")
}

func TestListCases_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/cases", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "20", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CasesResponse{
			Cases: []Case{
				{
					ID:          "case-1",
					Title:       "Investigating brute force",
					Description: "Multiple failed logins from 192.168.1.100",
					Status:      "open",
					Severity:    "high",
					CreatedBy:   "analyst-1",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					AlertCount:  5,
				},
			},
			Pagination: map[string]interface{}{
				"total": 1,
			},
		})
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	resp, err := client.ListCases(testToken, 1, 20, "")

	require.NoError(t, err)
	assert.Len(t, resp.Cases, 1)
	assert.Equal(t, "case-1", resp.Cases[0].ID)
	assert.Equal(t, "Investigating brute force", resp.Cases[0].Title)
	assert.Equal(t, "open", resp.Cases[0].Status)
	assert.Equal(t, 5, resp.Cases[0].AlertCount)
}

func TestListCases_WithStatusFilter(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "closed", r.URL.Query().Get("status"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CasesResponse{
			Cases:      []Case{},
			Pagination: map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	_, err := client.ListCases(testToken, 1, 10, "closed")

	assert.NoError(t, err)
}

func TestGetCase_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/cases/case-789", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Case{
			ID:          "case-789",
			Title:       "Security Incident",
			Description: "Detailed investigation",
			Status:      "in_progress",
			Severity:    "critical",
		})
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	caseData, err := client.GetCase(testToken, "case-789")

	require.NoError(t, err)
	assert.Equal(t, "case-789", caseData.ID)
	assert.Equal(t, "Security Incident", caseData.Title)
	assert.Equal(t, "in_progress", caseData.Status)
}

func TestCreateCase_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/cases", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		assert.Equal(t, "New Investigation", payload["title"])
		assert.Equal(t, "Investigating suspicious activity", payload["description"])
		assert.Equal(t, "high", payload["severity"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Case{
			ID:          "case-new",
			Title:       "New Investigation",
			Description: "Investigating suspicious activity",
			Status:      "open",
			Severity:    "high",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	caseData, err := client.CreateCase(testToken, "New Investigation", "Investigating suspicious activity", "high")

	require.NoError(t, err)
	assert.Equal(t, "case-new", caseData.ID)
	assert.Equal(t, "New Investigation", caseData.Title)
	assert.Equal(t, "open", caseData.Status)
}

func TestCreateCase_ValidationError(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"title is required"}`))
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	caseData, err := client.CreateCase(testToken, "", "", "")

	assert.Error(t, err)
	assert.Nil(t, caseData)
	assert.Contains(t, err.Error(), "failed to create case")
}

func TestUpdateCase_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/cases/case-update", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		assert.Equal(t, "Updated Title", payload["title"])
		assert.Equal(t, "analyst-2", payload["assigned_to"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Case{
			ID:         "case-update",
			Title:      "Updated Title",
			AssignedTo: "analyst-2",
		})
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	caseData, err := client.UpdateCase(testToken, "case-update", "Updated Title", "", "analyst-2")

	require.NoError(t, err)
	assert.Equal(t, "case-update", caseData.ID)
	assert.Equal(t, "Updated Title", caseData.Title)
	assert.Equal(t, "analyst-2", caseData.AssignedTo)
}

func TestUpdateCase_PartialUpdate(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)

		// Only description should be in payload
		assert.NotContains(t, payload, "title")
		assert.Contains(t, payload, "description")
		assert.NotContains(t, payload, "assigned_to")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Case{ID: "case-1"})
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	_, err := client.UpdateCase(testToken, "case-1", "", "Updated description", "")

	assert.NoError(t, err)
}

func TestCloseCase_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/cases/case-close/close", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	err := client.CloseCase(testToken, "case-close")

	assert.NoError(t, err)
}

func TestCloseCase_NotFound(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"case not found"}`))
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	err := client.CloseCase(testToken, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to close case")
}

func TestReopenCase_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/cases/case-reopen/reopen", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	err := client.ReopenCase(testToken, "case-reopen")

	assert.NoError(t, err)
}

func TestReopenCase_AlreadyOpen(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"case is already open"}`))
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)
	err := client.ReopenCase(testToken, "open-case")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reopen case")
}

func TestAlertingClient_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	client := NewAlertingClient(server.URL)

	// Test all methods with bad token
	_, err := client.ListAlerts("bad-token", 1, 10, nil)
	assert.Error(t, err)

	_, err = client.GetAlert("bad-token", "alert-1")
	assert.Error(t, err)

	_, err = client.ListCases("bad-token", 1, 10, "")
	assert.Error(t, err)

	_, err = client.GetCase("bad-token", "case-1")
	assert.Error(t, err)

	_, err = client.CreateCase("bad-token", "title", "desc", "high")
	assert.Error(t, err)

	_, err = client.UpdateCase("bad-token", "case-1", "title", "desc", "")
	assert.Error(t, err)

	err = client.CloseCase("bad-token", "case-1")
	assert.Error(t, err)

	err = client.ReopenCase("bad-token", "case-1")
	assert.Error(t, err)
}

func TestAlertingClient_NetworkError(t *testing.T) {
	client := NewAlertingClient("http://invalid-host-does-not-exist.local:99999")

	_, err := client.ListAlerts("token", 1, 10, nil)
	assert.Error(t, err)

	_, err = client.GetAlert("token", "alert-1")
	assert.Error(t, err)
}
