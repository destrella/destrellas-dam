package archivos

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestArchivarConFechaCreaJerarquia(t *testing.T) {
	t.Parallel()

	directorioTemporal := t.TempDir()
	rutaArchivado := filepath.Join(directorioTemporal, "Archivado DAM")
	rutaOrigen := filepath.Join(directorioTemporal, "entrada", "foto.jpg")
	if err := os.MkdirAll(filepath.Dir(rutaOrigen), 0o755); err != nil {
		t.Fatalf("no se pudo preparar el origen: %v", err)
	}
	if err := os.WriteFile(rutaOrigen, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("no se pudo crear el archivo origen: %v", err)
	}

	servicio := NuevoServicio(rutaArchivado)
	destino, err := servicio.ArchivarConFecha(context.Background(), rutaOrigen, "2024-05-09", "12:34:56")
	if err != nil {
		t.Fatalf("ArchivarConFecha devolvió error: %v", err)
	}

	rutaEsperada := filepath.Join(rutaArchivado, "2024", "05", "09", "foto.jpg")
	if destino != rutaEsperada {
		t.Fatalf("ruta inesperada: %q, esperada %q", destino, rutaEsperada)
	}
	if _, err := os.Stat(destino); err != nil {
		t.Fatalf("el archivo archivado no existe en destino: %v", err)
	}
	if _, err := os.Stat(rutaOrigen); !os.IsNotExist(err) {
		t.Fatalf("el archivo origen debió moverse, error actual: %v", err)
	}
}

func TestArchivarConFechaRequiereFechaYHora(t *testing.T) {
	t.Parallel()

	directorioTemporal := t.TempDir()
	rutaOrigen := filepath.Join(directorioTemporal, "video.mp4")
	if err := os.WriteFile(rutaOrigen, []byte("video"), 0o644); err != nil {
		t.Fatalf("no se pudo crear el archivo origen: %v", err)
	}

	servicio := NuevoServicio(filepath.Join(directorioTemporal, "Archivado DAM"))
	if _, err := servicio.ArchivarConFecha(context.Background(), rutaOrigen, "2024-05-09", ""); err == nil {
		t.Fatal("se esperaba error al archivar sin hora")
	}
	if _, err := servicio.ArchivarConFecha(context.Background(), rutaOrigen, "2024-99-09", "10:00:00"); err == nil {
		t.Fatal("se esperaba error al archivar con fecha inválida")
	}
}
