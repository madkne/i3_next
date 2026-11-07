package sway

import (
	"testing"
)

const (
	dquote   = "\x22"
	backtick = "\x60"
)

func TestValidateWorkspaceName(t *testing.T) {
	valid := []string{"1", "2", "10", "web", "code", "my.ws", "ws-1", "ws_1", "WS.A-1_b"}
	for _, v := range valid {
		if err := ValidateWorkspaceName(v); err != nil {
			t.Errorf("ValidateWorkspaceName(%q) = %v, want nil", v, err)
		}
	}

	invalid := []string{
		"", " ", "; exec foot", ";exec foot", "web; exec foot", "web\nexec foot",
		"workspace 2; exec foot", "web" + dquote + "; exec foot",
		"web" + backtick + "exec foot", "web$exec foot", "web|exec foot", "web&exec foot",
		"web<exec foot", "web>exec foot",
	}
	for _, v := range invalid {
		if err := ValidateWorkspaceName(v); err == nil {
			t.Errorf("ValidateWorkspaceName(%q) = nil, want error", v)
		}
	}
}

func TestValidateAppID(t *testing.T) {
	valid := []string{"foot", "firefox", "org.gnome.Nautilus", "org.kde.dolphin", "my-app_v1"}
	for _, v := range valid {
		if err := ValidateAppID(v); err != nil {
			t.Errorf("ValidateAppID(%q) = %v, want nil", v, err)
		}
	}

	invalid := []string{
		"", " ", "; exec foot", "foot" + dquote + "; exec foot", dquote + "foot" + dquote + "; exec foot",
		"foot\nexec foot", "foot" + backtick + "exec foot", "foot$exec foot",
	}
	for _, v := range invalid {
		if err := ValidateAppID(v); err == nil {
			t.Errorf("ValidateAppID(%q) = nil, want error", v)
		}
	}
}

func TestValidateOutputName(t *testing.T) {
	valid := []string{"eDP-1", "DP-1", "HDMI-A-1", "DP-2"}
	for _, v := range valid {
		if err := ValidateOutputName(v); err != nil {
			t.Errorf("ValidateOutputName(%q) = %v, want nil", v, err)
		}
	}

	if err := ValidateOutputName("; exec foot"); err == nil {
		t.Errorf("ValidateOutputName(%q) = nil, want error", "; exec foot")
	}
	if err := ValidateOutputName(""); err == nil {
		t.Errorf("ValidateOutputName(%q) = nil, want error", "")
	}
}
