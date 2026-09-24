package testutil

import "github.com/Zxilly/cjv/internal/progress"

// ProgressRecorder is a progress.Sink that keeps the kind of every event it
// receives, in order, so a test can assert what an operation reported.
type ProgressRecorder struct {
	Kinds []progress.Kind
}

// Report implements progress.Sink.
func (r *ProgressRecorder) Report(e progress.Event) {
	r.Kinds = append(r.Kinds, e.Kind)
}
