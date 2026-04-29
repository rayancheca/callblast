package ast

import (
	"strings"
	"testing"
)

func TestExtractGoFile_Functions(t *testing.T) {
	src := []byte(`package main

func Add(a, b int) int {
	return a + b
}

func Multiply(x, y float64) float64 {
	return x * y
}
`)
	funcs, _, err := ExtractGoFile("/tmp/test.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(funcs) != 2 {
		t.Errorf("expected 2 functions, got %d", len(funcs))
	}
	if funcs[0].Name != "Add" {
		t.Errorf("expected first function to be Add, got %s", funcs[0].Name)
	}
	if funcs[1].Name != "Multiply" {
		t.Errorf("expected second function to be Multiply, got %s", funcs[1].Name)
	}
}

func TestExtractGoFile_CallSites(t *testing.T) {
	src := []byte(`package main

func helper() string { return "x" }

func caller() {
	result := helper()
	_ = result
}
`)
	_, calls, err := ExtractGoFile("/tmp/test.go", src)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, c := range calls {
		if c.CalleeFunc == "helper" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected call to helper, got: %v", calls)
	}
}

func TestExtractGoFile_MethodReceiver(t *testing.T) {
	src := []byte(`package main

type Server struct{}

func (s *Server) Handle() error {
	return nil
}
`)
	funcs, _, err := ExtractGoFile("/tmp/test.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(funcs))
	}
	if funcs[0].Receiver == "" {
		t.Error("expected non-empty receiver")
	}
	if !strings.Contains(funcs[0].QualifiedName(), "Server") {
		t.Errorf("expected qualified name to contain receiver, got %s", funcs[0].QualifiedName())
	}
}

func TestExtractTSFile_SelectorCalls(t *testing.T) {
	src := []byte(`
export function processOrder(repo) {
  const order = repo.findById(id);
  repo.save(order);
  notify(order);
}
`)
	_, calls, err := ExtractTSFile("/tmp/test.ts", src)
	if err != nil {
		t.Fatal(err)
	}

	qualified := map[string]bool{}
	plain := map[string]bool{}
	for _, c := range calls {
		if strings.Contains(c.CalleeFunc, ".") {
			qualified[c.CalleeFunc] = true
		} else {
			plain[c.CalleeFunc] = true
		}
	}

	if !qualified["repo.findById"] {
		t.Errorf("expected qualified call repo.findById, got: %v", calls)
	}
	if !qualified["repo.save"] {
		t.Errorf("expected qualified call repo.save, got: %v", calls)
	}
	// notify() is a plain call — should still be captured
	if !plain["notify"] {
		t.Errorf("expected plain call notify, got: %v", calls)
	}
	// method names should NOT appear as plain calls when already emitted as qualified
	if plain["findById"] {
		t.Errorf("findById should not appear as a plain call (already in qualified form)")
	}
	if plain["save"] {
		t.Errorf("save should not appear as a plain call (already in qualified form)")
	}
}

func TestExtractGoFile_BodyHash(t *testing.T) {
	src1 := []byte(`package main

func Foo() {
	x := 1
	_ = x
}
`)
	src2 := []byte(`package main

func Foo() {
	x := 2
	_ = x
}
`)
	funcs1, _, _ := ExtractGoFile("/tmp/test.go", src1)
	funcs2, _, _ := ExtractGoFile("/tmp/test.go", src2)

	if len(funcs1) == 0 || len(funcs2) == 0 {
		t.Fatal("no functions extracted")
	}
	if funcs1[0].BodyHash == funcs2[0].BodyHash {
		t.Error("expected different body hashes for different bodies")
	}
}
