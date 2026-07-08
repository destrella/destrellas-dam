package yandex

import (
	"context"
	"errors"
	"io"

	"destrellas-dam/internal/modelo"
)

// ErrNoImplementado deja claro que la integracion esta encapsulada pero aun no completa.
var ErrNoImplementado = errors.New("integracion de Yandex.Disk pendiente de completarse")

// ElementoRemoto representa el contrato minimo consumido por la UI.
type ElementoRemoto struct {
	Ruta         string
	Nombre       string
	Tamano       int64
	Tipo         modelo.TipoArchivo
	EsDirectorio bool
	HashMD5      string
	HashSHA256   string
}

// Cliente describe la integracion futura con Yandex.Disk.
type Cliente interface {
	Configurado() bool
	ListarDirectorios(ctx context.Context, ruta string, limite, desplazamiento int) ([]ElementoRemoto, error)
	ListarElementos(ctx context.Context, ruta string, limite, desplazamiento int) ([]ElementoRemoto, error)
	Descargar(ctx context.Context, ruta string) (io.ReadCloser, error)
}

// ClienteNulo mantiene la app funcional aunque no haya integracion real todavia.
type ClienteNulo struct {
	clave string
}

// NuevoClienteNulo crea un cliente seguro para la primera version.
func NuevoClienteNulo(clave string) *ClienteNulo {
	return &ClienteNulo{clave: clave}
}

// Configurado informa si al menos existe una clave configurada.
func (c *ClienteNulo) Configurado() bool {
	return c != nil && c.clave != ""
}

// ListarDirectorios responde con un error explicito.
func (c *ClienteNulo) ListarDirectorios(_ context.Context, _ string, _ int, _ int) ([]ElementoRemoto, error) {
	return nil, ErrNoImplementado
}

// ListarElementos responde con un error explicito.
func (c *ClienteNulo) ListarElementos(_ context.Context, _ string, _ int, _ int) ([]ElementoRemoto, error) {
	return nil, ErrNoImplementado
}

// Descargar responde con un error explicito.
func (c *ClienteNulo) Descargar(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, ErrNoImplementado
}
