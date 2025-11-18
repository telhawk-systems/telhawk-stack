package seeder

import (
	"log"
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
	default:
		// Generic defaults
		event["category_uid"] = 0
		event["category_name"] = "Unknown"
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1
		}
	}

	// Add metadata
	if _, exists := event["metadata"]; !exists {
		event["metadata"] = map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
				"version":     "2.0.0-rule-based",
			},
		}
	}

	// Add time field if missing (use current time)
	if _, exists := event["time"]; !exists {
		event["time"] = time.Now().Unix()
	}
}
