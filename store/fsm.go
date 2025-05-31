package store

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/raft"
)

type OpType string

const (
	OpTypeSet    OpType = "set"
	OpTypeDelete OpType = "del"
)

var validate = validator.New()

// kvFsm implements the raft.FSM interface for a key-value store.
type kvFsm struct {
	data *sync.Map
}

// fsmPayload is the structure for data in Raft logs for data apply operations.
type fsmPayload struct {
	Op    OpType `validate:"required,oneof=set del"`
	Key   string `validate:"required"`
	Value string `validate:"required_if=Op set"`
}

// Apply applies a Raft log entry to the FSM.
func (kf *kvFsm) Apply(log *raft.Log) any {
	switch log.Type {
	case raft.LogCommand:
		var p fsmPayload
		if err := json.Unmarshal(log.Data, &p); err != nil {
			return fmt.Errorf("could not parse command payload: %w", err)
		}

		if err := validate.Struct(p); err != nil {
			return fmt.Errorf("invalid command payload: %w", err)
		}

		switch p.Op {
		case OpTypeSet:
			kf.data.Store(p.Key, p.Value)
		case OpTypeDelete:
			kf.data.Delete(p.Key)
		}

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
		var p fsmPayload
		if err := decoder.Decode(&p); err != nil {
			// Original code did not check for io.EOF specifically here.
			return fmt.Errorf("could not decode payload from snapshot: %w", err)
		}

		switch p.Op {
		case OpTypeSet:
			kf.data.Store(p.Key, p.Value)
		case OpTypeDelete:
			// don't expect delete op in snapshot,
			// but leave this here for completeness anw
			kf.data.Delete(p.Key)
		}
	}
	return rc.Close()
}
