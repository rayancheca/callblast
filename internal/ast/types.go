package ast

// ChangeType classifies how a function changed between base and head.
type ChangeType string

const (
	ChangeAdded           ChangeType = "added"
	ChangeRemoved         ChangeType = "removed"
	ChangeSignature       ChangeType = "signature_changed"
	ChangeBody            ChangeType = "body_changed"
	ChangeRenamed         ChangeType = "renamed"
)

// FunctionDef represents a parsed function from source.
type FunctionDef struct {
	Name       string
	File       string
	Line       int
	Signature  string // full signature as text
	BodyHash   string // SHA1 of body text for change detection
	Receiver   string // Go method receiver type (empty for functions)
	Language   string // "go" | "typescript"
}

// QualifiedName returns a unique identifier: "file::funcName" or "file::Receiver.funcName".
func (f *FunctionDef) QualifiedName() string {
	if f.Receiver != "" {
		return f.File + "::" + f.Receiver + "." + f.Name
	}
	return f.File + "::" + f.Name
}

// CallSite represents a function call found inside a function body.
type CallSite struct {
	CallerFunc string // qualified name of the caller
	CalleeFunc string // best-effort name of the callee (may be partial)
	File       string
	Line       int
}

// SemanticChange represents a detected change between two versions of a function.
type SemanticChange struct {
	Func       FunctionDef
	Type       ChangeType
	OldFunc    *FunctionDef // non-nil for renames and modifications
}
