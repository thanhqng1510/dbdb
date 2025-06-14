package conf

import (
	"testing"
)

func TestGetConfig_ValidArgs(t *testing.T) {
	args := []string{
		"--node-id", "node1",
		"--raft-port", "9000",
		"--http-port", "8000",
	}
	cfg, _ := GetConfig(args)
	if cfg.Id != "node1" {
		t.Errorf("expected Id 'node1', got '%s'", cfg.Id)
	}
	if cfg.RaftPort != "9000" {
		t.Errorf("expected RaftPort '9000', got '%s'", cfg.RaftPort)
	}
	if cfg.HttpPort != "8000" {
		t.Errorf("expected HttpPort '8000', got '%s'", cfg.HttpPort)
	}
	if cfg.Bootstrap {
		t.Errorf("expected Bootstrap false, got true")
	}
	if cfg.JoinAddr != "" {
		t.Errorf("expected JoinAddr '', got '%s'", cfg.JoinAddr)
	}
}

func TestGetConfig_BootstrapAndJoinConflict(t *testing.T) {
	args := []string{
		"--node-id", "node1",
		"--raft-port", "9000",
		"--http-port", "8000",
		"--bootstrap",
		"--join", "localhost:8000",
	}
	cfg, err := GetConfig(args)
	if err == nil {
		t.Fatalf("expected error due to --bootstrap and --join conflict, got nil")
	}
	if cfg != (Config{}) {
		t.Errorf("expected zero Config on error, got %+v", cfg)
	}
	if err.Error() != "error: --bootstrap cannot be used with --join" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetConfig_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing node-id",
			args:    []string{"--raft-port", "9000", "--http-port", "8000"},
			wantErr: "error: --node-id is required",
		},
		{
			name:    "missing raft-port",
			args:    []string{"--node-id", "node1", "--http-port", "8000"},
			wantErr: "error: --raft-port is required",
		},
		{
			name:    "missing http-port",
			args:    []string{"--node-id", "node1", "--raft-port", "9000"},
			wantErr: "error: --http-port is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := GetConfig(tc.args)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if cfg != (Config{}) {
				t.Errorf("expected zero Config on error, got %+v", cfg)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("expected error %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}
