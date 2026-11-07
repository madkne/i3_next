package sway

import "encoding/json"

type Version struct {
	HumanReadable string `json:"human_readable"`
	Build         string `json:"build"`
	BuildDate     string `json:"build_date"`
	BuildCommit   string `json:"build_commit"`
	ConfigPath    string `json:"config_path"`
}

func UnmarshalVersion(payload []byte) (*Version, error) {
	var v Version
	if err := json.Unmarshal(payload, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
