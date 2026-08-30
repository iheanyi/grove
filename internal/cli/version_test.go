package cli

import (
	"runtime/debug"
	"testing"
)

func TestWithBuildInfo(t *testing.T) {
	tests := []struct {
		name                              string
		version, commit, date             string
		info                              *debug.BuildInfo
		wantVersion, wantCommit, wantDate string
	}{
		{
			name:    "ldflags already set",
			version: "v1.2.3",
			commit:  "abcdef0",
			date:    "2026-08-30T20:00:00Z",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v0.10.2"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890abcdef"},
					{Key: "vcs.time", Value: "2026-08-29T20:00:00Z"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVersion: "v1.2.3",
			wantCommit:  "abcdef0",
			wantDate:    "2026-08-30T20:00:00Z",
		},
		{
			name:    "module version with no vcs settings",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v0.10.2"},
			},
			wantVersion: "v0.10.2",
			wantCommit:  "unknown",
			wantDate:    "unknown",
		},
		{
			name:    "devel with vcs revision and time",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890abcdef"},
					{Key: "vcs.time", Value: "2026-08-30T20:00:00Z"},
				},
			},
			wantVersion: "dev",
			wantCommit:  "1234567",
			wantDate:    "2026-08-30T20:00:00Z",
		},
		{
			name:    "dirty vcs revision",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			info: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abcdef1234567890"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVersion: "dev",
			wantCommit:  "abcdef1-dirty",
			wantDate:    "unknown",
		},
		{
			name:        "nil info",
			version:     "dev",
			commit:      "unknown",
			date:        "unknown",
			info:        nil,
			wantVersion: "dev",
			wantCommit:  "unknown",
			wantDate:    "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, gotCommit, gotDate := withBuildInfo(tt.version, tt.commit, tt.date, tt.info)
			if gotVersion != tt.wantVersion {
				t.Fatalf("version = %q, want %q", gotVersion, tt.wantVersion)
			}
			if gotCommit != tt.wantCommit {
				t.Fatalf("commit = %q, want %q", gotCommit, tt.wantCommit)
			}
			if gotDate != tt.wantDate {
				t.Fatalf("date = %q, want %q", gotDate, tt.wantDate)
			}
		})
	}
}
