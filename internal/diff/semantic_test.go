package diff

import (
	"testing"

	"github.com/rayancheca/callblast/internal/ast"
)

func TestDiffFunctions_Unchanged(t *testing.T) {
	base := []ast.FunctionDef{
		{Name: "Foo", File: "/f.go", Signature: "func Foo()", BodyHash: "abc123"},
	}
	changes := DiffFunctions(base, base)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for identical functions, got %d", len(changes))
	}
}

func TestDiffFunctions_BodyChanged(t *testing.T) {
	base := []ast.FunctionDef{
		{Name: "Foo", File: "/f.go", Signature: "func Foo()", BodyHash: "abc"},
	}
	head := []ast.FunctionDef{
		{Name: "Foo", File: "/f.go", Signature: "func Foo()", BodyHash: "xyz"},
	}
	changes := DiffFunctions(base, head)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != ast.ChangeBody {
		t.Errorf("expected body change, got %s", changes[0].Type)
	}
}

func TestDiffFunctions_SignatureChanged(t *testing.T) {
	base := []ast.FunctionDef{
		{Name: "Foo", File: "/f.go", Signature: "func Foo(a int)", BodyHash: "abc"},
	}
	head := []ast.FunctionDef{
		{Name: "Foo", File: "/f.go", Signature: "func Foo(a int, b string)", BodyHash: "abc"},
	}
	changes := DiffFunctions(base, head)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != ast.ChangeSignature {
		t.Errorf("expected signature change, got %s", changes[0].Type)
	}
}

func TestDiffFunctions_Added(t *testing.T) {
	base := []ast.FunctionDef{}
	head := []ast.FunctionDef{
		{Name: "New", File: "/f.go", Signature: "func New()", BodyHash: "abc"},
	}
	changes := DiffFunctions(base, head)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != ast.ChangeAdded {
		t.Errorf("expected added change, got %s", changes[0].Type)
	}
}

func TestDiffFunctions_Removed(t *testing.T) {
	base := []ast.FunctionDef{
		{Name: "Old", File: "/f.go", Signature: "func Old()", BodyHash: "abc"},
	}
	head := []ast.FunctionDef{}
	changes := DiffFunctions(base, head)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != ast.ChangeRemoved {
		t.Errorf("expected removed change, got %s", changes[0].Type)
	}
}
