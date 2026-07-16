package runner

// Kind identifies the type of output/error event delivered by the Runner.
type Kind string

const (
	// KindReplace carries the full new content of an interval-mode run.
	KindReplace Kind = "replace"
	// KindAppend carries one new line from a stream-mode section.
	KindAppend Kind = "append"
	// KindError carries a human-readable message for in-section display.
	KindError Kind = "error"
)

// Event is a single output or error notification for one section,
// delivered on the Runner's event channel.
type Event struct {
	SectionKey string
	Kind       Kind
	Text       string
}
