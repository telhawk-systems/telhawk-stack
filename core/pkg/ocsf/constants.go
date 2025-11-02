package ocsf

// Category UIDs as defined by OCSF schema
const (
	CategoryOther              = 0
	CategorySystemActivity     = 1
	CategoryFindings           = 2
	CategoryIAM                = 3
	CategoryNetworkActivity    = 4
	CategoryDiscovery          = 5
	CategoryApplicationActivity = 6
	CategoryRemediation        = 7
	CategoryUnmannedSystems    = 8
)

// Class UIDs for System Activity (Category 1)
const (
	ClassFileActivity          = 1001
	ClassKernelExtension       = 1002
	ClassKernelActivity        = 1003
	ClassMemoryActivity        = 1004
	ClassModuleActivity        = 1005
	ClassScheduledJobActivity  = 1006
	ClassProcessActivity       = 1007
	ClassEventLogActivity      = 1008
	ClassScriptActivity        = 1009
)

// Class UIDs for Findings (Category 2)
const (
	ClassVulnerabilityFinding             = 2002
	ClassComplianceFinding                = 2003
	ClassDetectionFinding                 = 2004
	ClassIncidentFinding                  = 2005
	ClassDataSecurityFinding              = 2006
	ClassApplicationSecurityPostureFinding = 2007
	ClassIAMAnalysisFinding               = 2008
)

// Class UIDs for IAM (Category 3)
const (
	ClassAccountChange      = 3001
	ClassAuthentication     = 3002
	ClassAuthorizeSession   = 3003
	ClassEntityManagement   = 3004
	ClassUserAccess         = 3005
	ClassGroupManagement    = 3006
)

// Class UIDs for Network Activity (Category 4)
const (
	ClassNetworkActivity = 4001
	ClassHTTPActivity    = 4002
	ClassDNSActivity     = 4003
	ClassDHCPActivity    = 4004
	ClassRDPActivity     = 4005
	ClassSMBActivity     = 4006
	ClassSSHActivity     = 4007
	ClassFTPActivity     = 4008
	ClassEmailActivity   = 4009
	ClassNTPActivity     = 4013
	ClassTunnelActivity  = 4014
)

// Class UIDs for Discovery (Category 5)
const (
	ClassInventoryInfo              = 5001
	ClassUserInventory              = 5003
	ClassPatchState                 = 5004
	ClassDeviceConfigStateChange    = 5019
	ClassSoftwareInfo               = 5020
	ClassOSINTInventoryInfo         = 5021
	ClassCloudResourcesInventoryInfo = 5023
	ClassEvidenceInfo               = 5040
)

// Class UIDs for Application Activity (Category 6)
const (
	ClassWebResourcesActivity = 6001
	ClassApplicationLifecycle = 6002
	ClassAPIActivity          = 6003
	ClassDatastoreActivity    = 6005
	ClassFileHostingActivity  = 6006
	ClassScanActivity         = 6007
	ClassApplicationError     = 6008
)

// Class UIDs for Remediation (Category 7)
const (
	ClassRemediationActivity        = 7001
	ClassFileRemediationActivity    = 7002
	ClassProcessRemediationActivity = 7003
	ClassNetworkRemediationActivity = 7004
)

// Class UIDs for Unmanned Systems (Category 8)
const (
	ClassDroneFlightsActivity     = 8001
	ClassAirborneBroadcastActivity = 8002
)

// Severity IDs as defined by OCSF
const (
	SeverityUnknown      = 0
	SeverityInformational = 1
	SeverityLow          = 2
	SeverityMedium       = 3
	SeverityHigh         = 4
	SeverityCritical     = 5
	SeverityFatal        = 6
)

// Status IDs as defined by OCSF
const (
	StatusUnknown = 0
	StatusSuccess = 1
	StatusFailure = 2
	StatusOther   = 99
)

// Authentication Activity IDs (for Class 3002)
const (
	AuthActivityUnknown  = 0
	AuthActivityLogon    = 1
	AuthActivityLogoff   = 2
	AuthActivityAuthTicket = 3
	AuthActivityServiceTicket = 4
	AuthActivityPreAuth  = 5
	AuthActivityOther    = 99
)

// Process Activity IDs (for Class 1007)
const (
	ProcessActivityUnknown  = 0
	ProcessActivityLaunch   = 1
	ProcessActivityTerminate = 2
	ProcessActivityInject   = 3
	ProcessActivityOpen     = 4
	ProcessActivityOther    = 99
)

// File Activity IDs (for Class 1001)
const (
	FileActivityUnknown     = 0
	FileActivityCreate      = 1
	FileActivityRead        = 2
	FileActivityUpdate      = 3
	FileActivityDelete      = 4
	FileActivityRename      = 5
	FileActivitySetAttributes = 6
	FileActivitySetSecurity = 7
	FileActivityEncrypt     = 8
	FileActivityDecrypt     = 9
	FileActivityMount       = 10
	FileActivityUnmount     = 11
	FileActivityOther       = 99
)

// Network Activity IDs (for Class 4001)
const (
	NetworkActivityUnknown      = 0
	NetworkActivityOpen         = 1
	NetworkActivityClose        = 2
	NetworkActivityConnect      = 3
	NetworkActivityRefuse       = 4
	NetworkActivityTraffic      = 5
	NetworkActivityReconnect    = 6
	NetworkActivityRenew        = 7
	NetworkActivityOther        = 99
)

// Detection Finding Activity IDs (for Class 2004)
const (
	DetectionActivityUnknown = 0
	DetectionActivityCreate  = 1
	DetectionActivityUpdate  = 2
	DetectionActivityClose   = 3
	DetectionActivityOther   = 99
)

// ComputeTypeUID calculates the OCSF type_uid from category and class.
// Formula: (category_uid * 100) + class_uid_suffix
// For classes under 100: type_uid = (category_uid * 100) + class_uid
// For classes >= 100: type_uid = class_uid * 100 + activity_id
func ComputeTypeUID(categoryUID, classUID, activityID int) int {
	if classUID < 100 {
		// Base event
		return classUID*100 + activityID
	}
	// Standard classes
	baseClass := classUID % 1000
	return (categoryUID * 100 + baseClass) * 100 + activityID
}

// SeverityName returns the human-readable severity name.
func SeverityName(severityID int) string {
	switch severityID {
	case SeverityUnknown:
		return "Unknown"
	case SeverityInformational:
		return "Informational"
	case SeverityLow:
		return "Low"
	case SeverityMedium:
		return "Medium"
	case SeverityHigh:
		return "High"
	case SeverityCritical:
		return "Critical"
	case SeverityFatal:
		return "Fatal"
	default:
		return "Unknown"
	}
}

// StatusName returns the human-readable status name.
func StatusName(statusID int) string {
	switch statusID {
	case StatusUnknown:
		return "Unknown"
	case StatusSuccess:
		return "Success"
	case StatusFailure:
		return "Failure"
	case StatusOther:
		return "Other"
	default:
		return "Unknown"
	}
}
