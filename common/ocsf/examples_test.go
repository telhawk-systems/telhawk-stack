package ocsf_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/common/ocsf"
)

// TestAuthentication demonstrates creating an OCSF Authentication event
func TestAuthentication(t *testing.T) {
	event := &ocsf.Event{
		// Required OCSF base fields
		CategoryUID: ocsf.CategoryIAM,
		ClassUID:    ocsf.ClassAuthentication,
		ActivityID:  ocsf.AuthActivityLogon,
		TypeUID:     ocsf.ComputeTypeUID(ocsf.CategoryIAM, ocsf.ClassAuthentication, ocsf.AuthActivityLogon),
		Time:        time.Now(),
		SeverityID:  ocsf.SeverityInformational,

		// Human-readable equivalents
		Category: "iam",
		Class:    "authentication",
		Activity: "Logon",
		Severity: "Informational",

		// Status
		StatusID: ocsf.StatusSuccess,
		Status:   "Success",

		// Timing
		ObservedTime: time.Now(),

		// Metadata
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "TelHawk Stack",
				Vendor: "TelHawk Systems",
			},
			Version:  "1.1.0",
			Profiles: []string{"host", "datetime"},
		},

		// Raw data preservation
		Raw: ocsf.RawDescriptor{
			Format: "json",
			Data: map[string]interface{}{
				"user":      "alice",
				"source_ip": "192.168.1.100",
			},
		},
	}

	data, _ := json.MarshalIndent(event, "", "  ")
	fmt.Println(string(data))
}

// TestNetworkActivity demonstrates creating an OCSF Network Activity event
func TestNetworkActivity(t *testing.T) {
	event := &ocsf.Event{
		// Required OCSF base fields
		CategoryUID: ocsf.CategoryNetworkActivity,
		ClassUID:    ocsf.ClassNetworkActivity,
		ActivityID:  ocsf.NetworkActivityConnect,
		TypeUID:     ocsf.ComputeTypeUID(ocsf.CategoryNetworkActivity, ocsf.ClassNetworkActivity, ocsf.NetworkActivityConnect),
		Time:        time.Now(),
		SeverityID:  ocsf.SeverityLow,

		// Human-readable equivalents
		Category: "network",
		Class:    "network_activity",
		Activity: "Connect",
		Severity: "Low",

		// Status
		StatusID: ocsf.StatusSuccess,
		Status:   "Success",

		// Timing
		ObservedTime: time.Now(),

		// Metadata
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "TelHawk Stack",
				Vendor: "TelHawk Systems",
			},
			Version: "1.1.0",
		},

		// Raw data
		Raw: ocsf.RawDescriptor{
			Format: "json",
			Data: map[string]interface{}{
				"src_ip":   "10.0.1.5",
				"dst_ip":   "93.184.216.34",
				"dst_port": 443,
			},
		},
	}

	data, _ := json.MarshalIndent(event, "", "  ")
	fmt.Println(string(data))
}

// TestComputeTypeUID demonstrates calculating the OCSF type_uid
func TestComputeTypeUID(t *testing.T) {
	// Authentication event
	authTypeUID := ocsf.ComputeTypeUID(
		ocsf.CategoryIAM,
		ocsf.ClassAuthentication,
		ocsf.AuthActivityLogon,
	)
	fmt.Printf("Authentication Logon type_uid: %d\n", authTypeUID)

	// Network Activity event
	networkTypeUID := ocsf.ComputeTypeUID(
		ocsf.CategoryNetworkActivity,
		ocsf.ClassNetworkActivity,
		ocsf.NetworkActivityConnect,
	)
	fmt.Printf("Network Connect type_uid: %d\n", networkTypeUID)

	// Process Activity event
	processTypeUID := ocsf.ComputeTypeUID(
		ocsf.CategorySystemActivity,
		ocsf.ClassProcessActivity,
		ocsf.ProcessActivityLaunch,
	)
	fmt.Printf("Process Launch type_uid: %d\n", processTypeUID)
}

// TestSeverityName demonstrates severity ID to name mapping
func TestSeverityName(t *testing.T) {
	for i := 0; i <= 6; i++ {
		fmt.Printf("Severity %d: %s\n", i, ocsf.SeverityName(i))
	}
}
