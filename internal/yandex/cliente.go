package yandex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"destrellas-dam/internal/modelo"
)

// ErrNoImplementado deja claro que la integracion esta encapsulada pero aun no completa.
var ErrNoImplementado = errors.New("integracion de Yandex.Disk pendiente de completarse")

const (
	baseURLDefecto = "https://cloud-api.yandex.net/v1/disk"
	rutaRaizYandex = "disk:/"
)

// ElementoRemoto representa el contrato minimo consumido por la UI.
type ElementoRemoto struct {
	Ruta         string
	Nombre       string
	Tamano       int64
	Tipo         modelo.TipoArchivo
	EsDirectorio bool
	HashMD5      string
	HashSHA256   string
	Modificado   time.Time
}

// Cliente describe la integracion futura con Yandex.Disk.
type Cliente interface {
	Configurado() bool
	ListarDirectorios(ctx context.Context, ruta string, limite, desplazamiento int) ([]ElementoRemoto, error)
	ListarElementos(ctx context.Context, ruta string, limite, desplazamiento int) ([]ElementoRemoto, error)
	Descargar(ctx context.Context, ruta string) (io.ReadCloser, error)
	DescargarPreview(ctx context.Context, ruta, tamano string) (io.ReadCloser, error)
	Mover(ctx context.Context, origen, destino string) error
	EnviarAPapelera(ctx context.Context, ruta string) error
}

// ClienteREST implementa el acceso a Yandex.Disk con el token configurado.
type ClienteREST struct {
	clave   string
	baseURL string
	cliente *http.Client
}

// ClienteNulo mantiene la app funcional aunque no haya integracion real todavia.
type ClienteNulo struct {
	clave string
}

// NuevoCliente crea el cliente adecuado segun exista o no un token configurado.
func NuevoCliente(clave string) Cliente {
	clave = strings.TrimSpace(clave)
	if clave == "" {
		return NuevoClienteNulo(clave)
	}
	return &ClienteREST{
		clave:   clave,
		baseURL: baseURLDefecto,
		cliente: &http.Client{Timeout: 30 * time.Second},
	}
}

// NuevoClienteNulo crea un cliente seguro para los casos sin token configurado.
func NuevoClienteNulo(clave string) *ClienteNulo {
	return &ClienteNulo{clave: strings.TrimSpace(clave)}
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

// DescargarPreview responde con un error explicito.
func (c *ClienteNulo) DescargarPreview(_ context.Context, _ string, _ string) (io.ReadCloser, error) {
	return nil, ErrNoImplementado
}

// Mover responde con un error explicito.
func (c *ClienteNulo) Mover(_ context.Context, _ string, _ string) error {
	return ErrNoImplementado
}

// EnviarAPapelera responde con un error explicito.
func (c *ClienteNulo) EnviarAPapelera(_ context.Context, _ string) error {
	return ErrNoImplementado
}

// Configurado informa si el cliente REST dispone de token usable.
func (c *ClienteREST) Configurado() bool {
	return c != nil && strings.TrimSpace(c.clave) != ""
}

// ListarDirectorios devuelve directorios hijos de la ruta pedida.
func (c *ClienteREST) ListarDirectorios(ctx context.Context, ruta string, limite, desplazamiento int) ([]ElementoRemoto, error) {
	return c.listarRecursosFiltrados(ctx, ruta, limite, desplazamiento, func(elemento ElementoRemoto) bool {
		return elemento.EsDirectorio
	})
}

// ListarElementos devuelve archivos y directorios hijos inmediatos de la ruta pedida.
func (c *ClienteREST) ListarElementos(ctx context.Context, ruta string, limite, desplazamiento int) ([]ElementoRemoto, error) {
	return c.listarRecursosFiltrados(ctx, ruta, limite, desplazamiento, nil)
}

// Descargar abre un flujo de lectura al contenido remoto solicitado.
func (c *ClienteREST) Descargar(ctx context.Context, ruta string) (io.ReadCloser, error) {
	if !c.Configurado() {
		return nil, ErrNoImplementado
	}

	href, err := c.obtenerURLDescarga(ctx, ruta)
	if err != nil {
		return nil, err
	}
	return c.descargarDesdeURL(ctx, href, "no se pudo descargar el archivo remoto desde Yandex.Disk")
}

// DescargarPreview obtiene la miniatura remota en el tamaño solicitado.
func (c *ClienteREST) DescargarPreview(ctx context.Context, ruta, tamano string) (io.ReadCloser, error) {
	if !c.Configurado() {
		return nil, ErrNoImplementado
	}

	href, err := c.obtenerURLPreview(ctx, ruta, tamano)
	if err != nil {
		return nil, err
	}
	return c.descargarDesdeURL(ctx, href, "no se pudo descargar la vista previa remota desde Yandex.Disk")
}

// Mover traslada un archivo o carpeta remota a otra carpeta de Yandex.Disk.
func (c *ClienteREST) Mover(ctx context.Context, origen, destino string) error {
	if !c.Configurado() {
		return ErrNoImplementado
	}

	parametros := url.Values{}
	parametros.Set("from", normalizarRutaYandex(origen))
	parametros.Set("path", normalizarRutaYandex(destino))
	parametros.Set("overwrite", "false")

	endpoint := c.baseURL + "/resources/move?" + parametros.Encode()
	return c.ejecutarPeticionEstado(ctx, http.MethodPost, endpoint, http.StatusCreated, http.StatusAccepted)
}

// EnviarAPapelera mueve el recurso remoto a la papelera de Yandex.Disk.
func (c *ClienteREST) EnviarAPapelera(ctx context.Context, ruta string) error {
	if !c.Configurado() {
		return ErrNoImplementado
	}

	parametros := url.Values{}
	parametros.Set("path", normalizarRutaYandex(ruta))
	parametros.Set("permanent", "false")

	endpoint := c.baseURL + "/resources?" + parametros.Encode()
	return c.ejecutarPeticionEstado(ctx, http.MethodDelete, endpoint, http.StatusNoContent, http.StatusAccepted)
}

func (c *ClienteREST) listarRecursosFiltrados(ctx context.Context, ruta string, limite, desplazamiento int, aceptar func(ElementoRemoto) bool) ([]ElementoRemoto, error) {
	if !c.Configurado() {
		return nil, ErrNoImplementado
	}
	if limite < 1 {
		limite = 40
	}
	if desplazamiento < 0 {
		desplazamiento = 0
	}

	tamanoPagina := limite
	if tamanoPagina < 100 {
		tamanoPagina = 100
	}
	if tamanoPagina > 500 {
		tamanoPagina = 500
	}

	var (
		resultados []ElementoRemoto
		saltados   int
		offsetRaw  int
	)
	if aceptar == nil {
		offsetRaw = desplazamiento
	}

	for len(resultados) < limite {
		pagina, err := c.listarPaginaRecursos(ctx, ruta, tamanoPagina, offsetRaw)
		if err != nil {
			return nil, err
		}
		if len(pagina) == 0 {
			break
		}
		offsetRaw += len(pagina)

		for _, elemento := range pagina {
			if aceptar != nil && !aceptar(elemento) {
				continue
			}
			if saltados < desplazamiento {
				saltados++
				continue
			}
			resultados = append(resultados, elemento)
			if len(resultados) >= limite {
				break
			}
		}

		if len(pagina) < tamanoPagina {
			break
		}
	}

	return resultados, nil
}

func (c *ClienteREST) obtenerURLDescarga(ctx context.Context, ruta string) (string, error) {
	endpoint := c.baseURL + "/resources/download?path=" + url.QueryEscape(normalizarRutaYandex(ruta))
	return c.obtenerCampoHref(ctx, endpoint, "no se pudo solicitar el enlace de descarga a Yandex.Disk")
}

func (c *ClienteREST) obtenerURLPreview(ctx context.Context, ruta, tamano string) (string, error) {
	parametros := url.Values{}
	parametros.Set("path", normalizarRutaYandex(ruta))
	parametros.Set("fields", "preview")
	parametros.Set("preview_size", normalizarTamanoPreview(tamano))

	endpoint := c.baseURL + "/resources?" + parametros.Encode()
	req, err := c.nuevaSolicitudAutenticada(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("no se pudo preparar la solicitud de vista previa remota: %w", err)
	}

	resp, err := c.cliente.Do(req)
	if err != nil {
		return "", fmt.Errorf("no se pudo solicitar la vista previa remota a Yandex.Disk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", describirErrorRespuesta(resp)
	}

	var payload struct {
		Preview string `json:"preview"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("no se pudo interpretar la respuesta de vista previa de Yandex.Disk: %w", err)
	}
	if strings.TrimSpace(payload.Preview) == "" {
		return "", errors.New("Yandex.Disk no devolvió una vista previa para este recurso")
	}
	return strings.TrimSpace(payload.Preview), nil
}

func (c *ClienteREST) obtenerCampoHref(ctx context.Context, endpoint, mensaje string) (string, error) {
	req, err := c.nuevaSolicitudAutenticada(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("no se pudo preparar la solicitud remota: %w", err)
	}

	resp, err := c.cliente.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", mensaje, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", describirErrorRespuesta(resp)
	}

	var payload struct {
		Href string `json:"href"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("no se pudo interpretar la respuesta remota de Yandex.Disk: %w", err)
	}
	if strings.TrimSpace(payload.Href) == "" {
		return "", errors.New("Yandex.Disk no devolvió una URL de descarga")
	}
	return strings.TrimSpace(payload.Href), nil
}

func (c *ClienteREST) descargarDesdeURL(ctx context.Context, href, mensaje string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, href, nil)
	if err != nil {
		return nil, fmt.Errorf("no se pudo preparar la descarga remota: %w", err)
	}
	if c.Configurado() {
		req.Header.Set("Authorization", "OAuth "+strings.TrimSpace(c.clave))
	}

	resp, err := c.cliente.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", mensaje, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, describirErrorRespuesta(resp)
	}
	return resp.Body, nil
}

func (c *ClienteREST) listarPaginaRecursos(ctx context.Context, ruta string, limite, offset int) ([]ElementoRemoto, error) {
	ruta = normalizarRutaYandex(ruta)
	endpoint := fmt.Sprintf(
		"%s/resources?path=%s&limit=%d&offset=%d&fields=_embedded.items.name,_embedded.items.path,_embedded.items.type,_embedded.items.size,_embedded.items.md5,_embedded.items.sha256,_embedded.items.modified",
		c.baseURL,
		url.QueryEscape(ruta),
		limite,
		offset,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("no se pudo preparar la solicitud a Yandex.Disk: %w", err)
	}
	req.Header.Set("Authorization", "OAuth "+strings.TrimSpace(c.clave))

	resp, err := c.cliente.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no se pudo consultar Yandex.Disk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, describirErrorRespuesta(resp)
	}

	var payload respuestaRecursosYandex
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("no se pudo interpretar la respuesta de Yandex.Disk: %w", err)
	}

	elementos := make([]ElementoRemoto, 0, len(payload.Embedded.Items))
	for _, item := range payload.Embedded.Items {
		elementos = append(elementos, convertirItemRecurso(item))
	}
	return elementos, nil
}

func (c *ClienteREST) ejecutarPeticionEstado(ctx context.Context, metodo, endpoint string, codigosExito ...int) error {
	req, err := c.nuevaSolicitudAutenticada(ctx, metodo, endpoint, nil)
	if err != nil {
		return fmt.Errorf("no se pudo preparar la operación remota de Yandex.Disk: %w", err)
	}

	resp, err := c.cliente.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo ejecutar la operación remota en Yandex.Disk: %w", err)
	}
	defer resp.Body.Close()

	if coincideCodigoExito(resp.StatusCode, codigosExito) {
		return nil
	}
	return describirErrorRespuesta(resp)
}

func coincideCodigoExito(codigo int, codigosExito []int) bool {
	for _, candidato := range codigosExito {
		if codigo == candidato {
			return true
		}
	}
	return false
}

type respuestaRecursosYandex struct {
	Embedded struct {
		Items []itemRecursoYandex `json:"items"`
	} `json:"_embedded"`
}

type itemRecursoYandex struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	Modified string `json:"modified"`
}

func convertirItemRecurso(item itemRecursoYandex) ElementoRemoto {
	esDirectorio := strings.EqualFold(strings.TrimSpace(item.Type), "dir")
	ruta := normalizarRutaYandex(item.Path)
	modificado, _ := time.Parse(time.RFC3339, strings.TrimSpace(item.Modified))
	return ElementoRemoto{
		Ruta:         ruta,
		Nombre:       strings.TrimSpace(item.Name),
		Tamano:       item.Size,
		EsDirectorio: esDirectorio,
		Tipo:         modelo.TipoDesdeRuta(ruta, esDirectorio),
		HashMD5:      strings.TrimSpace(item.MD5),
		HashSHA256:   strings.TrimSpace(item.SHA256),
		Modificado:   modificado,
	}
}

func normalizarRutaYandex(ruta string) string {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" || ruta == "/" || ruta == "disk:" || ruta == "disk:/" {
		return rutaRaizYandex
	}
	if strings.HasPrefix(ruta, "disk:/") {
		return "disk:/" + strings.TrimPrefix(strings.TrimPrefix(ruta, "disk:/"), "/")
	}
	if strings.HasPrefix(ruta, "/") {
		return "disk:" + ruta
	}
	return "disk:/" + strings.TrimPrefix(ruta, "/")
}

func normalizarTamanoPreview(tamano string) string {
	tamano = strings.ToUpper(strings.TrimSpace(tamano))
	switch tamano {
	case "S", "M", "L", "XL", "XXL", "XXXL":
		return tamano
	default:
		return "M"
	}
}

func (c *ClienteREST) nuevaSolicitudAutenticada(ctx context.Context, metodo, endpoint string, cuerpo io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, metodo, endpoint, cuerpo)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+strings.TrimSpace(c.clave))
	return req, nil
}

func describirErrorRespuesta(resp *http.Response) error {
	var payload struct {
		Message     string `json:"message"`
		Description string `json:"description"`
		Error       string `json:"error"`
	}
	cuerpo, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	_ = json.Unmarshal(cuerpo, &payload)

	var detalles []string
	if texto := strings.TrimSpace(payload.Message); texto != "" {
		detalles = append(detalles, texto)
	}
	if texto := strings.TrimSpace(payload.Description); texto != "" && !strings.EqualFold(texto, payload.Message) {
		detalles = append(detalles, texto)
	}
	if texto := strings.TrimSpace(payload.Error); texto != "" && !strings.EqualFold(texto, payload.Message) {
		detalles = append(detalles, texto)
	}
	if len(detalles) == 0 {
		texto := strings.TrimSpace(string(cuerpo))
		if texto != "" {
			detalles = append(detalles, texto)
		}
	}
	if len(detalles) == 0 {
		return fmt.Errorf("Yandex.Disk devolvió %s", resp.Status)
	}
	return fmt.Errorf("Yandex.Disk devolvió %s: %s", resp.Status, strings.Join(detalles, " | "))
}
