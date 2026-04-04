package controllers

import (
	"encoding/json"

	pickle "github.com/telhawk-systems/telhawk-stack/findings/app/http"
	"github.com/telhawk-systems/telhawk-stack/findings/app/http/requests"
	"github.com/telhawk-systems/telhawk-stack/findings/app/models"

	"github.com/google/uuid"
)

type ScanController struct {
	pickle.Controller
}

func (c ScanController) Index(ctx *pickle.Context) pickle.Response {
	authID, err := uuid.Parse(ctx.Auth().UserID)
	if err != nil {
		return ctx.Unauthorized("invalid auth")
	}

	scans, err := models.QueryScan().
		WhereUserID(authID).
		Limit(100).
		All()
	if err != nil {
		return ctx.Error(err)
	}

	return ctx.JSON(200, scans)
}

func (c ScanController) Show(ctx *pickle.Context) pickle.Response {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return ctx.JSON(400, map[string]string{"error": "invalid id"})
	}

	authID, err := uuid.Parse(ctx.Auth().UserID)
	if err != nil {
		return ctx.Unauthorized("invalid auth")
	}

	scan, err := models.QueryScan().
		WhereID(id).
		WhereUserID(authID).
		First()
	if err != nil {
		return ctx.NotFound("scan not found")
	}

	return ctx.JSON(200, scan)
}

func (c ScanController) Store(ctx *pickle.Context) pickle.Response {
	authID, err := uuid.Parse(ctx.Auth().UserID)
	if err != nil {
		return ctx.Unauthorized("invalid auth")
	}

	req, bindErr := requests.BindCreateScanRequest(ctx.Request())
	if bindErr != nil {
		return ctx.JSON(bindErr.Status, bindErr)
	}

	scan := &models.Scan{
		UserID:      authID,
		Tool:        req.Tool,
		Project:     req.Project,
		CommitHash:  &req.CommitHash,
		ToolVersion: &req.ToolVersion,
		SignalCount: len(req.Signals),
	}
	if err := models.QueryScan().Create(scan); err != nil {
		return ctx.Error(err)
	}

	for _, sig := range req.Signals {
		dataBytes, err := json.Marshal(sig.Data)
		if err != nil {
			return ctx.Error(err)
		}

		finding := &models.Finding{
			Fingerprint: sig.Fingerprint,
			ScanID:      scan.ID,
			Tool:        req.Tool,
			SignalType:  sig.SignalType,
			Severity:    &sig.Severity,
			Category:    &sig.Category,
			Route:       &sig.Route,
			FilePath:    &sig.FilePath,
			Line:        &sig.Line,
			Data:        json.RawMessage(dataBytes),
		}
		if err := models.QueryFinding().Create(finding); err != nil {
			return ctx.Error(err)
		}
	}

	return ctx.JSON(201, scan)
}

func (c ScanController) Destroy(ctx *pickle.Context) pickle.Response {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return ctx.JSON(400, map[string]string{"error": "invalid id"})
	}

	authID, err := uuid.Parse(ctx.Auth().UserID)
	if err != nil {
		return ctx.Unauthorized("invalid auth")
	}

	scan, err := models.QueryScan().
		WhereID(id).
		WhereUserID(authID).
		First()
	if err != nil {
		return ctx.NotFound("scan not found")
	}

	// Delete findings for this scan first, then the scan
	findings, err := models.QueryFinding().WhereScanID(scan.ID).All()
	if err != nil {
		return ctx.Error(err)
	}
	for _, f := range findings {
		if err := models.QueryFinding().Delete(&f); err != nil {
			return ctx.Error(err)
		}
	}

	if err := models.QueryScan().Delete(scan); err != nil {
		return ctx.Error(err)
	}

	return ctx.NoContent()
}
