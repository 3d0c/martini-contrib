package linker

import (
	"github.com/3d0c/martini-contrib/config"
	"testing"
)

func TestLinker(t *testing.T) {
	config.Init("./linker_test.json")

	link := Get()

	if !link.Default {
		t.Fatalf("Expected link.Default = true, got %v\n", link.Default)
	}
}
