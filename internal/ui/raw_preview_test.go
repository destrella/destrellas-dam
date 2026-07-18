package ui

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/yandex"
)

type clienteYandexPreviewPrueba struct {
	llamoPreviewRuta bool
	llamoPreviewURL  bool
	png              []byte
}

func (c *clienteYandexPreviewPrueba) Configurado() bool { return true }

func (c *clienteYandexPreviewPrueba) ListarDirectorios(context.Context, string, int, int) ([]yandex.ElementoRemoto, error) {
	return nil, yandex.ErrNoImplementado
}

func (c *clienteYandexPreviewPrueba) ListarElementos(context.Context, string, int, int) ([]yandex.ElementoRemoto, error) {
	return nil, yandex.ErrNoImplementado
}

func (c *clienteYandexPreviewPrueba) Descargar(context.Context, string) (io.ReadCloser, error) {
	return nil, yandex.ErrNoImplementado
}

func (c *clienteYandexPreviewPrueba) DescargarPreview(context.Context, string, string) (io.ReadCloser, error) {
	c.llamoPreviewRuta = true
	return io.NopCloser(bytes.NewReader(c.png)), nil
}

func (c *clienteYandexPreviewPrueba) DescargarPreviewURL(context.Context, string) (io.ReadCloser, error) {
	c.llamoPreviewURL = true
	return io.NopCloser(bytes.NewReader(c.png)), nil
}

func (c *clienteYandexPreviewPrueba) Mover(context.Context, string, string) error {
	return yandex.ErrNoImplementado
}

func (c *clienteYandexPreviewPrueba) EnviarAPapelera(context.Context, string) error {
	return yandex.ErrNoImplementado
}

func TestDecodificarPreviewRemotaUsaURLListadaParaRaw(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	imagen := image.NewRGBA(image.Rect(0, 0, 4, 4))
	imagen.Set(0, 0, color.NRGBA{R: 255, A: 255})
	if err := png.Encode(&buffer, imagen); err != nil {
		t.Fatalf("no se pudo preparar el PNG de prueba: %v", err)
	}

	cliente := &clienteYandexPreviewPrueba{png: buffer.Bytes()}
	app := &Aplicacion{
		clienteYandex: cliente,
	}

	preview, err := app.decodificarPreview(modelo.Archivo{
		Origen:     modelo.OrigenYandex,
		Ruta:       "disk:/RAW/captura.cr3",
		PreviewURL: "https://preview/listada",
		Tipo:       modelo.TipoImagen,
	}, 360)
	if err != nil {
		t.Fatalf("no se pudo decodificar la preview remota RAW: %v", err)
	}
	if preview == nil {
		t.Fatal("se esperaba una imagen de preview")
	}
	if !cliente.llamoPreviewURL {
		t.Fatal("la preview remota debería descargarse desde la URL listada")
	}
	if cliente.llamoPreviewRuta {
		t.Fatal("no debería usar la solicitud secundaria por ruta cuando ya existe preview listada")
	}
}
