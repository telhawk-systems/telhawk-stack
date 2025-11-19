package seeder

import (
	"log"
	"math/rand"
	"time"
)

// ocsfEnricher handles OCSF-specific field enrichment
type ocsfEnricher struct {
	fieldGen *fieldGenerator
}

// newOCSFEnricher creates a new OCSF enricher
func newOCSFEnricher(fieldGen *fieldGenerator) *ocsfEnricher {
	return &ocsfEnricher{
		fieldGen: fieldGen,
	}
}

// enrichEvent adds all required OCSF fields to an event
func (oe *ocsfEnricher) enrichEvent(event map[string]interface{}) {
	oe.addRequiredOCSFFields(event)
}

// enrichNetworkEvent adds network-specific OCSF fields
func (oe *ocsfEnricher) enrichNetworkEvent(event map[string]interface{}, groupByValues map[string]interface{}, index int) {
	classUID, ok := event["class_uid"].(float64)
	if !ok || int(classUID) != 4001 { // Network Activity
		return
	}

	log.Printf("  DEBUG: Enriching network endpoints for OCSF compliance")

	// Ensure src_endpoint has all required fields (ip, port, name, uid)
	if srcEp, ok := event["src_endpoint"].(map[string]interface{}); ok {
		if _, hasIP := srcEp["ip"]; !hasIP {
			// For port scanning, all events should have the same source IP
			if index == 0 {
				// Generate once and store in groupByValues for reuse
				groupByValues[".src_endpoint.ip"] = oe.fieldGen.generateValueForField(".src_endpoint.ip")
			}
			srcEp["ip"] = groupByValues[".src_endpoint.ip"]
		}
		if _, hasPort := srcEp["port"]; !hasPort {
			srcEp["port"] = oe.fieldGen.generateValueForField(".src_endpoint.port")
		}
		if _, hasName := srcEp["name"]; !hasName {
			srcEp["name"] = ""
		}
		if _, hasUID := srcEp["uid"]; !hasUID {
			srcEp["uid"] = ""
		}
	}

	// Ensure dst_endpoint has all required fields (ip, port, name, uid)
	if dstEp, ok := event["dst_endpoint"].(map[string]interface{}); ok {
		if _, hasIP := dstEp["ip"]; !hasIP {
			// For port scanning, all events should have the same destination IP (the target)
			if index == 0 {
				// Generate once and store in groupByValues for reuse
				groupByValues[".dst_endpoint.ip"] = oe.fieldGen.generateValueForField(".dst_endpoint.ip")
			}
			dstEp["ip"] = groupByValues[".dst_endpoint.ip"]
		}
		if _, hasPort := dstEp["port"]; !hasPort {
			dstEp["port"] = oe.fieldGen.generateValueForField(".dst_endpoint.port")
		}
		if _, hasName := dstEp["name"]; !hasName {
			dstEp["name"] = ""
		}
		if _, hasUID := dstEp["uid"]; !hasUID {
			dstEp["uid"] = ""
		}
	}
}

// addRequiredOCSFFields adds mandatory OCSF fields based on the event's class_uid
func (oe *ocsfEnricher) addRequiredOCSFFields(event map[string]interface{}) {
	classUIDRaw, ok := event["class_uid"]
	if !ok {
		return
	}

	// Convert to int for comparison
	var classUID int
	switch v := classUIDRaw.(type) {
	case int:
		classUID = v
	case float64:
		classUID = int(v)
	default:
		return
	}

	// Add baseline device fields for all events (realistic production data)
	if _, exists := event["device"]; !exists {
		event["device"] = map[string]interface{}{
			"hostname": oe.fieldGen.generateValueForField("device.hostname"),
			"ip":       oe.fieldGen.generateValueForField("device.ip"),
			"type":     "Server",
			"type_id":  1, // Server
			"os": map[string]interface{}{
				"name":    "Linux",
				"type":    "Linux",
				"type_id": 200,
			},
		}
	}

	// Add category_uid based on class_uid
	switch classUID {
	case 3002: // Authentication
		event["category_uid"] = 3
		event["category_name"] = "Identity & Access Management"
		event["class_name"] = "Authentication"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 1 // Logon
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "Logon"
		}
		if _, exists := event["severity_id"]; !exists {
			if statusID, ok := event["status_id"].(int); ok && statusID == 2 {
				event["severity_id"] = 3 // Medium for failures
			} else {
				event["severity_id"] = 1 // Informational
			}
		}
		if _, exists := event["status"]; !exists {
			if statusID, ok := event["status_id"].(int); ok && statusID == 2 {
				event["status"] = "Failure"
			} else {
				event["status"] = "Success"
			}
		}
		// Add baseline actor.session if not present
		if actor, ok := event["actor"].(map[string]interface{}); ok {
			if _, exists := actor["session"]; !exists {
				actor["session"] = map[string]interface{}{
					"uid":        oe.fieldGen.generateValueForField("actor.session.uid"),
					"issuer":     "TelHawk Auth",
					"created_at": time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second).Unix(),
				}
			}
		}
	case 4001: // Network Activity
		event["category_uid"] = 4
		event["category_name"] = "Network Activity"
		event["class_name"] = "Network Activity"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 5 // Traffic
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "Traffic"
		}
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1 // Informational
		}
		// Add connection_info if not exists (required for network events)
		if _, exists := event["connection_info"]; !exists {
			event["connection_info"] = map[string]interface{}{
				"protocol_name": "TCP",
				"direction":     "outbound",
				"direction_id":  2, // Outbound
			}
		}
	case 1007: // Process Activity
		event["category_uid"] = 1
		event["category_name"] = "System Activity"
		event["class_name"] = "Process Activity"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 1 // Launch
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "Launch"
		}
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1
		}
		// Add baseline process fields if not present
		if _, exists := event["process"]; !exists {
			event["process"] = map[string]interface{}{
				"name":     oe.fieldGen.generateValueForField("process.name"),
				"pid":      oe.fieldGen.generateValueForField("process.pid"),
				"cmd_line": oe.fieldGen.generateValueForField("process.cmd_line"),
				"file": map[string]interface{}{
					"path": oe.fieldGen.generateValueForField("file.path"),
					"name": oe.fieldGen.generateValueForField("file.name"),
				},
			}
		}
		// Add actor if not present (process activities have actors)
		if _, exists := event["actor"]; !exists {
			event["actor"] = map[string]interface{}{
				"user": map[string]interface{}{
					"name": oe.fieldGen.generateValueForField("actor.user.name"),
					"uid":  oe.fieldGen.generateValueForField("actor.user.uid"),
				},
			}
		}
	case 3001: // Account Change
		event["category_uid"] = 3
		event["category_name"] = "Identity & Access Management"
		event["class_name"] = "Account Change"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 1 // Create
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "Create"
		}
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 2 // Low
		}
		// Add user (target of account change) if not present
		if _, exists := event["user"]; !exists {
			event["user"] = map[string]interface{}{
				"name": oe.fieldGen.generateValueForField("user.name"),
				"uid":  oe.fieldGen.generateValueForField("user.uid"),
			}
		}
		// Add actor (who made the change) if not present
		if _, exists := event["actor"]; !exists {
			event["actor"] = map[string]interface{}{
				"user": map[string]interface{}{
					"name": oe.fieldGen.generateValueForField("actor.user.name"),
					"uid":  oe.fieldGen.generateValueForField("actor.user.uid"),
				},
			}
		}
	case 4003: // DNS Activity
		event["category_uid"] = 4
		event["category_name"] = "Network Activity"
		event["class_name"] = "DNS Activity"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 1 // Query
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "Query"
		}
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1 // Informational
		}
		// Ensure query object exists with hostname
		if _, exists := event["query"]; !exists {
			event["query"] = map[string]interface{}{
				"hostname": oe.fieldGen.generateValueForField("query.hostname"),
				"type":     "A",
				"class":    "IN",
			}
		}
	case 4002: // HTTP Activity
		event["category_uid"] = 4
		event["category_name"] = "Network Activity"
		event["class_name"] = "HTTP Activity"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 1 // HTTP Request
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "HTTP Request"
		}
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1 // Informational
		}
		// Ensure http_request object exists
		if _, exists := event["http_request"]; !exists {
			event["http_request"] = map[string]interface{}{
				"method": "GET",
				"url": map[string]interface{}{
					"path":     oe.fieldGen.generateValueForField("http_request.url.path"),
					"scheme":   "https",
					"hostname": oe.fieldGen.generateValueForField("http_request.url.hostname"),
				},
				"user_agent": oe.fieldGen.generateValueForField("http_request.user_agent"),
			}
		}
	case 4006: // File Activity
		event["category_uid"] = 4
		event["category_name"] = "Network Activity"
		event["class_name"] = "File Activity"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 1 // Create
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "Create"
		}
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1 // Informational
		}
		// Ensure file object exists
		if _, exists := event["file"]; !exists {
			event["file"] = map[string]interface{}{
				"path": oe.fieldGen.generateValueForField("file.path"),
				"name": oe.fieldGen.generateValueForField("file.name"),
				"size": oe.fieldGen.generateValueForField("file.size"),
			}
		}
		// Add actor if not present (file activities have actors)
		if _, exists := event["actor"]; !exists {
			event["actor"] = map[string]interface{}{
				"user": map[string]interface{}{
					"name": oe.fieldGen.generateValueForField("actor.user.name"),
					"uid":  oe.fieldGen.generateValueForField("actor.user.uid"),
				},
			}
		}
	default:
		// Generic defaults
		event["category_uid"] = 0
		event["category_name"] = "Unknown"
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1
		}
	}

	// Add metadata with product info
	if _, exists := event["metadata"]; !exists {
		event["metadata"] = map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
				"version":     "2.0.0-rule-based",
			},
			"version": "1.1.0", // OCSF schema version
		}
	}

	// Add time field if missing (use current time)
	if _, exists := event["time"]; !exists {
		event["time"] = time.Now().Unix()
	}
}
