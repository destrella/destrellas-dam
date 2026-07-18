package ui

import (
	"strings"
	"testing"
)

func TestNuevaTemaConfiguraFallbackEmoji(t *testing.T) {
	t.Parallel()

	tema := nuevaTema(nuevaPaleta())
	if tema == nil {
		t.Fatal("se esperaba un tema válido")
	}
	if tema.Shaper == nil {
		t.Fatal("el tema debería crear un shaper de texto")
	}
	if !strings.Contains(strings.ToLower(string(tema.Face)), "emoji") {
		t.Fatalf("la familia tipográfica debería incluir fallback de emoji, se obtuvo %q", tema.Face)
	}
}
