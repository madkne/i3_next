package sway

import "encoding/json"

type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Workspace struct {
	Num            int    `json:"num"`
	Name           string `json:"name"`
	Visible        bool   `json:"visible"`
	Focused        bool   `json:"focused"`
	Urgent         bool   `json:"urgent"`
	Rect           Rect   `json:"rect"`
	Output         string `json:"output"`
	Type           string `json:"type"`
	Representation string `json:"representation"`
}

func UnmarshalWorkspaces(payload []byte) ([]Workspace, error) {
	var ws []Workspace
	if err := json.Unmarshal(payload, &ws); err != nil {
		return nil, err
	}
	return ws, nil
}

type Output struct {
	ID               int     `json:"id"`
	Name             string  `json:"name"`
	Active           bool    `json:"active"`
	CurrentWorkspace string  `json:"current_workspace"`
	Rect             Rect    `json:"rect"`
	Scale            float64 `json:"scale"`
	Transform        string  `json:"transform"`
	Make             string  `json:"make"`
	Model            string  `json:"model"`
	Serial           string  `json:"serial"`
}

func UnmarshalOutputs(payload []byte) ([]Output, error) {
	var os []Output
	if err := json.Unmarshal(payload, &os); err != nil {
		return nil, err
	}
	return os, nil
}

type Node struct {
	ID                 int            `json:"id"`
	Name               string         `json:"name"`
	Type               string         `json:"type"`
	Border             string         `json:"border"`
	CurrentBorderWidth int            `json:"current_border_width"`
	Layout             string         `json:"layout"`
	Orientation        string         `json:"orientation"`
	Percent            float64        `json:"percent"`
	Rect               Rect           `json:"rect"`
	WindowRect         Rect           `json:"window_rect"`
	DecoRect           Rect           `json:"deco_rect"`
	Geometry           Rect           `json:"geometry"`
	Window             int            `json:"window"`
	Urgent             bool           `json:"urgent"`
	Focused            bool           `json:"focused"`
	Focus              []int          `json:"focus"`
	Floating           string         `json:"floating"`
	Sticky             bool           `json:"sticky"`
	PID                int            `json:"pid"`
	AppID              string         `json:"app_id"`
	WindowProperties   map[string]any `json:"window_properties"`
	Nodes              []Node         `json:"nodes"`
	FloatingNodes      []Node         `json:"floating_nodes"`
	Marks              []string       `json:"marks"`
	FullscreenMode     int            `json:"fullscreen_mode"`
	Active             bool           `json:"active"`
	CurrentWorkspace   string         `json:"current_workspace"`
	Scale              float64        `json:"scale"`
	Transform          string         `json:"transform"`
	Make               string         `json:"make"`
	Model              string         `json:"model"`
	Serial             string         `json:"serial"`
}

func UnmarshalTree(payload []byte) (*Node, error) {
	var n Node
	if err := json.Unmarshal(payload, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

type Input struct {
	Identifier       string         `json:"identifier"`
	Name             string         `json:"name"`
	Vendor           int            `json:"vendor"`
	Product          int            `json:"product"`
	Type             string         `json:"type"`
	XKBLayouts       []string       `json:"xkb_layouts"`
	XKBCurrentLayout string         `json:"xkb_current_layout_index"`
	Libinput         map[string]any `json:"libinput"`
}

func UnmarshalInputs(payload []byte) ([]Input, error) {
	var ins []Input
	if err := json.Unmarshal(payload, &ins); err != nil {
		return nil, err
	}
	return ins, nil
}

type BarConfig struct {
	ID            string `json:"id"`
	Mode          string `json:"mode"`
	Position      string `json:"position"`
	StatusCommand string `json:"status_command"`
}

func UnmarshalBarConfig(payload []byte) (*BarConfig, error) {
	var b BarConfig
	if err := json.Unmarshal(payload, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

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
