package metadatos

import "testing"

func TestExtraerFechaHoraEditableIgnoraFileModifyDate(t *testing.T) {
	t.Parallel()

	documento := map[string]any{
		"FileModifyDate": "2026:07:05 09:15:00-06:00",
	}

	fecha, hora, zona := extraerFechaHoraEditable(documento)
	if fecha != "" {
		t.Fatalf("no debería inferirse fecha desde FileModifyDate, se obtuvo %q", fecha)
	}
	if hora != "" {
		t.Fatalf("no debería inferirse hora desde FileModifyDate, se obtuvo %q", hora)
	}
	if zona != "" {
		t.Fatalf("no debería inferirse zona horaria desde FileModifyDate, se obtuvo %q", zona)
	}
}

func TestExtraerFechaHoraEditableConservaMetadatosReales(t *testing.T) {
	t.Parallel()

	documento := map[string]any{
		"DateTimeOriginal": "2024:03:01 10:20:30",
		"FileModifyDate":   "2026:07:05 09:15:00-06:00",
	}

	fecha, hora, zona := extraerFechaHoraEditable(documento)
	if fecha != "2024-03-01" {
		t.Fatalf("fecha inesperada: %q", fecha)
	}
	if hora != "10:20:30" {
		t.Fatalf("hora inesperada: %q", hora)
	}
	if zona != "" {
		t.Fatalf("la zona horaria no debería provenir de FileModifyDate, se obtuvo %q", zona)
	}
}
