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
	"time"

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

func TestGuardarMetadatosReemplazaPalabrasClaveYSujetosPrevios(t *testing.T) {
	t.Parallel()

	servicio := NuevoServicio()
	if servicio.rutaExiftool == "" {
		t.Skip("exiftool no está disponible")
	}

	directorioTemporal := t.TempDir()
	rutaImagen := filepath.Join(directorioTemporal, "reemplazo.jpg")
	crearJPEGPrueba(t, rutaImagen)

	comandoInicial := exec.CommandContext(context.Background(), servicio.rutaExiftool,
		"-overwrite_original_in_place",
		"-P",
		"-m",
		"-charset", "IPTC=UTF8",
		"-codedcharacterset=UTF8",
		"-Keywords=старое",
		"-Keywords=selfie",
		"-Subject=Мария",
		rutaImagen,
	)
	if salida, err := comandoInicial.CombinedOutput(); err != nil {
		t.Fatalf("no se pudo preparar el estado inicial del archivo: %v: %s", err, strings.TrimSpace(string(salida)))
	}

	metadatos := modelo.MetadatosArchivo{
		PalabrasClave: []string{"Марія Тагаєва", "selfie", "teléfono"},
		Sujetos:       []string{"Марія Тагаєва", "selfie", "teléfono"},
	}
	if err := servicio.GuardarMetadatos(context.Background(), rutaImagen, metadatos); err != nil {
		t.Fatalf("GuardarMetadatos devolvió error: %v", err)
	}

	comandoLectura := exec.CommandContext(context.Background(), servicio.rutaExiftool,
		"-j",
		"-G1",
		"-a",
		"-Keywords",
		"-Subject",
		rutaImagen,
	)
	salida, err := comandoLectura.CombinedOutput()
	if err != nil {
		t.Fatalf("no se pudieron leer los metadatos escritos: %v: %s", err, strings.TrimSpace(string(salida)))
	}

	texto := string(salida)
	if strings.Contains(texto, "старое") || strings.Contains(texto, "Мария") {
		t.Fatalf("persistieron valores previos que debieron reemplazarse; salida:\n%s", texto)
	}
	if !strings.Contains(texto, "\"IPTC:Keywords\": [\"Марія Тагаєва\",\"selfie\",\"teléfono\"]") {
		t.Fatalf("IPTC:Keywords no quedó reemplazado correctamente; salida:\n%s", texto)
	}
	if !strings.Contains(texto, "\"XMP-dc:Subject\": [\"Марія Тагаєва\",\"selfie\",\"teléfono\"]") {
		t.Fatalf("XMP-dc:Subject no quedó reemplazado correctamente; salida:\n%s", texto)
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

func TestConstruirRutaSalidaFrameIncluyeNumeroYFormato(t *testing.T) {
	t.Parallel()

	ruta := construirRutaSalidaFrame("/tmp/video original.mov", "jpg", 42)
	esperada := "/tmp/video original-frame-000042.jpg"
	if ruta != esperada {
		t.Fatalf("ruta inesperada: %q", ruta)
	}
}

func TestParsearTasaFotogramas(t *testing.T) {
	t.Parallel()

	valor, err := parsearTasaFotogramas("30000/1001")
	if err != nil {
		t.Fatalf("parsearTasaFotogramas devolvió error: %v", err)
	}
	if valor < 29.96 || valor > 29.98 {
		t.Fatalf("fps inesperado: %.6f", valor)
	}
}

func TestNumeroFotogramaAproximado(t *testing.T) {
	t.Parallel()

	numero := numeroFotogramaAproximado(2*time.Second, 29.97)
	if numero != 60 {
		t.Fatalf("número de frame inesperado: %d", numero)
	}
}
