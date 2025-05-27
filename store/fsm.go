package store

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
)

// kvFsm implements the raft.FSM interface for a key-value store.
type kvFsm struct {
	data *sync.Map
}

// setPayload is the structure for data in Raft logs for set operations.
type setPayload struct {
	Key   string
	Value string
}

// Apply applies a Raft log entry to the FSM.
func (kf *kvFsm) Apply(log *raft.Log) any {
	switch log.Type {
	case raft.LogCommand:
		var sp setPayload
		if err := json.Unmarshal(log.Data, &sp); err != nil {
			return fmt.Errorf("could not parse command payload: %w", err)
		}
		kf.data.Store(sp.Key, sp.Value)
		return nil // Return nil for success, or an error object for FSM-level errors
	default:
		return fmt.Errorf("unknown raft log type: %#v", log.Type)
	}
}

// snapshotNoop is a no-op FSMSnapshot implementation.
type snapshotNoop struct{}

func (sn snapshotNoop) Persist(_ raft.SnapshotSink) error { return nil }
func (sn snapshotNoop) Release() {}

// Snapshot returns a snapshot of the FSM state.
func (kf *kvFsm) Snapshot() (raft.FSMSnapshot, error) {
	// A real implementation would iterate over kf.db and write to the sink.
	return snapshotNoop{}, nil
}

// Restore restores the FSM state from a snapshot.
func (kf *kvFsm) Restore(rc io.ReadCloser) error {
	decoder := json.NewDecoder(rc)
	for decoder.More() {
		var sp setPayload
		if err := decoder.Decode(&sp); err != nil {
			// Original code did not check for io.EOF specifically here.
			return fmt.Errorf("could not decode payload from snapshot: %w", err)
		}
		kf.data.Store(sp.Key, sp.Value)
	}
	return rc.Close()
}