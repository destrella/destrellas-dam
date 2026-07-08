package metadatos

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"destrellas-dam/internal/modelo"
)

func TestGuardarMetadatosConCirlicoMantieneUTF8EnIPTC(t *testing.T) {
	t.Parallel()

	servicio := NuevoServicio()
	if servicio.rutaExiftool == "" {
		t.Skip("exiftool no está disponible")
	}

	directorioTemporal := t.TempDir()
	rutaImagen := filepath.Join(directorioTemporal, "cirilico.jpg")
	crearJPEGPrueba(t, rutaImagen)

	metadatos := modelo.MetadatosArchivo{
		PalabrasClave: []string{"Привет", "Москва"},
		Sujetos:       []string{"Привет", "Москва"},
	}
	if err := servicio.GuardarMetadatos(context.Background(), rutaImagen, metadatos); err != nil {
		t.Fatalf("GuardarMetadatos devolvió error: %v", err)
	}

	comando := exec.CommandContext(context.Background(), servicio.rutaExiftool,
		"-s",
		"-G1",
		"-a",
		"-Keywords",
		"-Subject",
		"-CodedCharacterSet",
		rutaImagen,
	)
	salida, err := comando.CombinedOutput()
	if err != nil {
		t.Fatalf("no se pudieron leer los metadatos escritos: %v: %s", err, strings.TrimSpace(string(salida)))
	}

	texto := string(salida)
	if !strings.Contains(texto, "[IPTC]          Keywords                        : Привет, Москва") {
		t.Fatalf("IPTC:Keywords no conservó UTF-8; salida:\n%s", texto)
	}
	if !strings.Contains(texto, "[XMP-dc]        Subject                         : Привет, Москва") {
		t.Fatalf("XMP-dc:Subject no contiene los valores esperados; salida:\n%s", texto)
	}
	if !strings.Contains(texto, "[IPTC]          CodedCharacterSet               : UTF8") {
		t.Fatalf("IPTC no quedó marcado como UTF-8; salida:\n%s", texto)
	}
}

func crearJPEGPrueba(t *testing.T, ruta string) {
	t.Helper()

	imagen := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			imagen.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	archivo, err := os.Create(ruta)
	if err != nil {
		t.Fatalf("no se pudo crear la imagen de prueba: %v", err)
	}
	defer archivo.Close()

	if err := jpeg.Encode(archivo, imagen, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("no se pudo codificar el JPEG de prueba: %v", err)
	}
}
