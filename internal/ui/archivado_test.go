package ui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/servicios/archivos"
)

func TestArchivarArchivoConFechaUsaJerarquiaPorDia(t *testing.T) {
	t.Parallel()

	directorioTemporal := t.TempDir()
	carpetaArchivado := filepath.Join(directorioTemporal, "archivado")
	servicio := archivos.NuevoServicio(carpetaArchivado)

	rutaOrigen := filepath.Join(directorioTemporal, "foto.jpg")
	if err := os.WriteFile(rutaOrigen, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("no se pudo preparar el archivo origen: %v", err)
	}

	archivo := modelo.Archivo{
		Ruta: rutaOrigen,
		Metadatos: modelo.MetadatosArchivo{
			Fecha: "2024-05-09",
			Hora:  "12:34:56",
		},
	}

	destino, err := archivarArchivoConFecha(context.Background(), servicio, archivo)
	if err != nil {
		t.Fatalf("archivarArchivoConFecha devolvió error: %v", err)
	}

	esperado := filepath.Join(carpetaArchivado, "2024", "05", "09", "foto.jpg")
	if destino != esperado {
		t.Fatalf("destino inesperado: %q", destino)
	}
	if _, err := os.Stat(destino); err != nil {
		t.Fatalf("no se encontró el archivo archivado: %v", err)
	}
}

func TestArchivarArchivoConFechaRequiereMetadatosArchivables(t *testing.T) {
	t.Parallel()

	servicio := archivos.NuevoServicio(t.TempDir())
	archivo := modelo.Archivo{
		Ruta: "/tmp/foto.jpg",
		Metadatos: modelo.MetadatosArchivo{
			Fecha: "2024-05-09",
		},
	}

	if _, err := archivarArchivoConFecha(context.Background(), servicio, archivo); err == nil {
		t.Fatal("se esperaba error al intentar archivar sin hora válida")
	}
}
