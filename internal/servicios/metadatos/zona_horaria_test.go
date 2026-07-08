package metadatos

import "testing"

func TestInferirZonaHorariaDesdeGPS(t *testing.T) {
	t.Parallel()

	servicio := &Servicio{}
	zona, err := servicio.InferirZonaHorariaDesdeGPS("2026-07-05", "", 39.9254474, 116.3870752)
	if err != nil {
		t.Fatalf("InferirZonaHorariaDesdeGPS devolvió error: %v", err)
	}
	if zona != "+08:00" {
		t.Fatalf("offset inesperado: %q", zona)
	}
}

func TestInferirZonaHorariaDesdeGPSRequiereFechaValida(t *testing.T) {
	t.Parallel()

	servicio := &Servicio{}
	if _, err := servicio.InferirZonaHorariaDesdeGPS("", "", 39.9254474, 116.3870752); err == nil {
		t.Fatal("se esperaba error al inferir zona horaria sin fecha")
	}
}
