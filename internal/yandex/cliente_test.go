package yandex

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"destrellas-dam/internal/modelo"
)

func TestClienteRESTListaElementos(t *testing.T) {
	t.Parallel()

	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/resources" {
			t.Fatalf("ruta inesperada: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "OAuth token-demo" {
			t.Fatalf("cabecera de autorización inesperada: %q", r.Header.Get("Authorization"))
		}
		if got := r.URL.Query().Get("path"); got != "disk:/" {
			t.Fatalf("ruta remota inesperada: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"_embedded": {
				"items": [
					{
						"name": "Fotos",
						"path": "disk:/Fotos",
						"type": "dir",
						"size": 0,
						"modified": "2026-07-12T15:00:00+00:00"
					},
					{
						"name": "clip.mp4",
						"path": "disk:/Videos/clip.mp4",
						"type": "file",
						"size": 1048576,
						"md5": "md5-demo",
						"sha256": "sha-demo",
						"modified": "2026-07-12T16:30:00+00:00"
					}
				]
			}
		}`)
	}))
	defer servidor.Close()

	cliente := &ClienteREST{
		clave:   "token-demo",
		baseURL: servidor.URL,
		cliente: servidor.Client(),
	}

	elementos, err := cliente.ListarElementos(context.Background(), "disk:/", 10, 0)
	if err != nil {
		t.Fatalf("ListarElementos devolvió error: %v", err)
	}
	if len(elementos) != 2 {
		t.Fatalf("cantidad inesperada de elementos: %d", len(elementos))
	}
	if !elementos[0].EsDirectorio || elementos[0].Tipo != modelo.TipoDirectorio {
		t.Fatalf("el primer elemento debería ser un directorio, se obtuvo %+v", elementos[0])
	}
	if elementos[1].Tipo != modelo.TipoVideo || elementos[1].HashMD5 != "md5-demo" || elementos[1].HashSHA256 != "sha-demo" {
		t.Fatalf("el segundo elemento no se interpretó correctamente: %+v", elementos[1])
	}
	if !elementos[1].Modificado.Equal(time.Date(2026, 7, 12, 16, 30, 0, 0, time.FixedZone("", 0))) {
		t.Fatalf("fecha de modificación inesperada: %v", elementos[1].Modificado)
	}
}

func TestClienteRESTListaDirectoriosFiltrandoArchivos(t *testing.T) {
	t.Parallel()

	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"_embedded": {
				"items": [
					{"name":"uno.jpg","path":"disk:/uno.jpg","type":"file","size":100},
					{"name":"Carpeta A","path":"disk:/Carpeta A","type":"dir","size":0},
					{"name":"dos.jpg","path":"disk:/dos.jpg","type":"file","size":120},
					{"name":"Carpeta B","path":"disk:/Carpeta B","type":"dir","size":0}
				]
			}
		}`)
	}))
	defer servidor.Close()

	cliente := &ClienteREST{
		clave:   "token-demo",
		baseURL: servidor.URL,
		cliente: servidor.Client(),
	}

	elementos, err := cliente.ListarDirectorios(context.Background(), "disk:/", 10, 0)
	if err != nil {
		t.Fatalf("ListarDirectorios devolvió error: %v", err)
	}
	if len(elementos) != 2 {
		t.Fatalf("se esperaban 2 directorios, se obtuvieron %d", len(elementos))
	}
	if elementos[0].Nombre != "Carpeta A" || elementos[1].Nombre != "Carpeta B" {
		t.Fatalf("directorios inesperados: %+v", elementos)
	}
}

func TestClienteRESTDescargaContenidoRemoto(t *testing.T) {
	t.Parallel()

	var servidor *httptest.Server
	servidor = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/resources/download":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"href":"`+servidor.URL+`/contenido/demo"}`)
		case "/contenido/demo":
			_, _ = io.WriteString(w, "contenido remoto")
		default:
			http.NotFound(w, r)
		}
	}))
	defer servidor.Close()

	cliente := &ClienteREST{
		clave:   "token-demo",
		baseURL: servidor.URL,
		cliente: servidor.Client(),
	}

	lector, err := cliente.Descargar(context.Background(), "disk:/demo.txt")
	if err != nil {
		t.Fatalf("Descargar devolvió error: %v", err)
	}
	defer lector.Close()

	contenido, err := io.ReadAll(lector)
	if err != nil {
		t.Fatalf("no se pudo leer el contenido descargado: %v", err)
	}
	if strings.TrimSpace(string(contenido)) != "contenido remoto" {
		t.Fatalf("contenido descargado inesperado: %q", string(contenido))
	}
}

func TestClienteRESTDescargaPreviewRemoto(t *testing.T) {
	t.Parallel()

	var servidor *httptest.Server
	servidor = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/resources":
			if got := r.URL.Query().Get("preview_size"); got != "XXXL" {
				t.Fatalf("tamaño de preview inesperado: %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"preview":"`+servidor.URL+`/preview/demo"}`)
		case "/preview/demo":
			w.Header().Set("Content-Type", "image/png")
			_, _ = io.WriteString(w, "preview remoto")
		default:
			http.NotFound(w, r)
		}
	}))
	defer servidor.Close()

	cliente := &ClienteREST{
		clave:   "token-demo",
		baseURL: servidor.URL,
		cliente: servidor.Client(),
	}

	lector, err := cliente.DescargarPreview(context.Background(), "disk:/demo.jpg", "XXXL")
	if err != nil {
		t.Fatalf("DescargarPreview devolvió error: %v", err)
	}
	defer lector.Close()

	contenido, err := io.ReadAll(lector)
	if err != nil {
		t.Fatalf("no se pudo leer la preview descargada: %v", err)
	}
	if strings.TrimSpace(string(contenido)) != "preview remoto" {
		t.Fatalf("preview descargada inesperada: %q", string(contenido))
	}
}

func TestClienteRESTMoverRecursoRemoto(t *testing.T) {
	t.Parallel()

	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/resources/move" {
			t.Fatalf("ruta inesperada: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("método inesperado: %s", r.Method)
		}
		if got := r.URL.Query().Get("from"); got != "disk:/Fotos/demo.jpg" {
			t.Fatalf("origen inesperado: %q", got)
		}
		if got := r.URL.Query().Get("path"); got != "disk:/Archivado/demo.jpg" {
			t.Fatalf("destino inesperado: %q", got)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer servidor.Close()

	cliente := &ClienteREST{
		clave:   "token-demo",
		baseURL: servidor.URL,
		cliente: servidor.Client(),
	}

	if err := cliente.Mover(context.Background(), "disk:/Fotos/demo.jpg", "disk:/Archivado/demo.jpg"); err != nil {
		t.Fatalf("Mover devolvió error: %v", err)
	}
}

func TestClienteRESTEnviarAPapelera(t *testing.T) {
	t.Parallel()

	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/resources" {
			t.Fatalf("ruta inesperada: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Fatalf("método inesperado: %s", r.Method)
		}
		if got := r.URL.Query().Get("path"); got != "disk:/Fotos/demo.jpg" {
			t.Fatalf("ruta inesperada para papelera: %q", got)
		}
		if got := r.URL.Query().Get("permanent"); got != "false" {
			t.Fatalf("valor de permanent inesperado: %q", got)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer servidor.Close()

	cliente := &ClienteREST{
		clave:   "token-demo",
		baseURL: servidor.URL,
		cliente: servidor.Client(),
	}

	if err := cliente.EnviarAPapelera(context.Background(), "disk:/Fotos/demo.jpg"); err != nil {
		t.Fatalf("EnviarAPapelera devolvió error: %v", err)
	}
}

func TestNormalizarRutaYandex(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		"":              "disk:/",
		"/":             "disk:/",
		"disk:/":        "disk:/",
		"disk:/Fotos":   "disk:/Fotos",
		"/Fotos/2026":   "disk:/Fotos/2026",
		"Fotos/2026":    "disk:/Fotos/2026",
		"disk://Videos": "disk:/Videos",
	}

	for entrada, esperada := range casos {
		if obtenida := normalizarRutaYandex(entrada); obtenida != esperada {
			t.Fatalf("normalización inesperada para %q: %q", entrada, obtenida)
		}
	}
}

func TestNormalizarTamanoPreview(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		"":     "M",
		"m":    "M",
		"XXXL": "XXXL",
		"foo":  "M",
	}

	for entrada, esperada := range casos {
		if obtenida := normalizarTamanoPreview(entrada); obtenida != esperada {
			t.Fatalf("tamaño normalizado inesperado para %q: %q", entrada, obtenida)
		}
	}
}
