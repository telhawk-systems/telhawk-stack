package requests

type CreateScanRequest struct {
	Tool        string   `json:"tool" validate:"required,max=50"`
	Project     string   `json:"project" validate:"required,max=255"`
	CommitHash  string   `json:"commit_hash" validate:"max=64"`
	ToolVersion string   `json:"tool_version" validate:"max=50"`
	Signals     []Signal `json:"signals"`
}

type Signal struct {
	Fingerprint string         `json:"fingerprint" validate:"required,max=64"`
	SignalType  string         `json:"signal_type" validate:"required,max=50"`
	Severity    string         `json:"severity" validate:"omitempty,oneof=critical high medium low info"`
	Category    string         `json:"category" validate:"max=100"`
	Route       string         `json:"route" validate:"max=500"`
	FilePath    string         `json:"file_path" validate:"max=500"`
	Line        int            `json:"line"`
	Data        map[string]any `json:"data" validate:"required"`
}
