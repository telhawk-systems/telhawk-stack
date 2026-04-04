package controllers

import (
	pickle "github.com/telhawk-systems/telhawk-stack/findings/app/http"
	"github.com/telhawk-systems/telhawk-stack/findings/app/models"

	"github.com/google/uuid"
)

type FindingController struct {
	pickle.Controller
}

func (c FindingController) Index(ctx *pickle.Context) pickle.Response {
	authID, err := uuid.Parse(ctx.Auth().UserID)
	if err != nil {
		return ctx.Unauthorized("invalid auth")
	}

	// Get scan IDs owned by this user
	scans, err := models.QueryScan().WhereUserID(authID).All()
	if err != nil {
		return ctx.Error(err)
	}
	if len(scans) == 0 {
		return ctx.JSON(200, []models.Finding{})
	}

	scanIDs := make([]uuid.UUID, len(scans))
	for i, s := range scans {
		scanIDs[i] = s.ID
	}

	q := models.QueryFinding().
		WhereScanIDIn(scanIDs).
		Limit(100)

	if tool := ctx.Query("tool"); tool != "" {
		q = q.WhereTool(tool)
	}
	if severity := ctx.Query("severity"); severity != "" {
		q = q.WhereSeverity(&severity)
	}
	if signalType := ctx.Query("signal_type"); signalType != "" {
		q = q.WhereSignalType(signalType)
	}
	if category := ctx.Query("category"); category != "" {
		q = q.WhereCategory(&category)
	}

	findings, err := q.All()
	if err != nil {
		return ctx.Error(err)
	}

	return ctx.JSON(200, findings)
}

func (c FindingController) Show(ctx *pickle.Context) pickle.Response {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return ctx.JSON(400, map[string]string{"error": "invalid id"})
	}

	authID, err := uuid.Parse(ctx.Auth().UserID)
	if err != nil {
		return ctx.Unauthorized("invalid auth")
	}

	finding, err := models.QueryFinding().WhereID(id).First()
	if err != nil {
		return ctx.NotFound("finding not found")
	}

	// Verify ownership via scan
	_, err = models.QueryScan().
		WhereID(finding.ScanID).
		WhereUserID(authID).
		First()
	if err != nil {
		return ctx.NotFound("finding not found")
	}

	return ctx.JSON(200, finding)
}
