package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"destrellas-dam/internal/modelo"
)

// Almacen implementa persistencia sobre SQLite con pragmas pensados para catálogos grandes.
type Almacen struct {
	base                      *sql.DB
	muEscritura               sync.Mutex
	sincronizacionEtiquetas   sync.Once
	errorSincronizacionIndice error
}

// Nuevo abre o crea la base de datos.
func Nuevo(ruta string) (*Almacen, error) {
	base, err := sql.Open("sqlite", ruta)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir SQLite: %w", err)
	}

	if _, err := base.Exec(pragmasSQLite); err != nil {
		base.Close()
		return nil, fmt.Errorf("no se pudieron aplicar los pragmas de SQLite: %w", err)
	}
	if _, err := base.Exec(esquemaSQLite); err != nil {
		base.Close()
		return nil, fmt.Errorf("no se pudo inicializar el esquema de SQLite: %w", err)
	}

	return &Almacen{base: base}, nil
}

// Cerrar libera la base de datos.
func (a *Almacen) Cerrar() error {
	if a == nil || a.base == nil {
		return nil
	}
	return a.base.Close()
}

// GuardarArchivo inserta o actualiza un registro respetando los datos enriquecidos previos.
func (a *Almacen) GuardarArchivo(ctx context.Context, archivo modelo.Archivo) error {
	if a == nil || a.base == nil {
		return errors.New("almacen sqlite no inicializado")
	}
	a.muEscritura.Lock()
	defer a.muEscritura.Unlock()

	metadatosJSON, err := json.Marshal(archivo.Metadatos)
	if err != nil {
		return fmt.Errorf("no se pudieron serializar los metadatos: %w", err)
	}

	duracionMilis := int64(archivo.Duracion / time.Millisecond)
	modificadoUnix := archivo.Modificado.Unix()
	whereFroms := strings.Join(archivo.Metadatos.WhereFroms, "\n")
	ahora := time.Now().Unix()
	tieneCargaEnriquecida := archivo.Ancho > 0 ||
		archivo.Alto > 0 ||
		duracionMilis > 0 ||
		!archivo.Metadatos.MetadatosVacios() ||
		!archivo.Indicadores.IndicadoresVacios() ||
		archivo.Hashes.MD5 != "" ||
		archivo.Hashes.SHA256 != "" ||
		archivo.Hashes.DHashImagen != "" ||
		archivo.Hashes.DHashVideo != ""

	consulta := consultaUpsertBasico
	if tieneCargaEnriquecida {
		consulta = consultaUpsertEnriquecido
	}

	args := []any{
		archivo.Ruta,
		string(archivo.Origen),
		archivo.RutaPadre,
		archivo.NombreVisible(),
		archivo.Tamano,
		modificadoUnix,
		string(archivo.Tipo),
		boolAEntero(archivo.EsOculto),
		boolAEntero(archivo.EsDirectorio),
		archivo.Ancho,
		archivo.Alto,
		duracionMilis,
		string(metadatosJSON),
		archivo.Hashes.MD5,
		archivo.Hashes.SHA256,
		archivo.Hashes.DHashImagen,
		archivo.Hashes.DHashVideo,
		boolAEntero(archivo.Indicadores.TieneGPS),
		boolAEntero(archivo.Indicadores.TieneRegiones),
		boolAEntero(archivo.Indicadores.TieneWhereFrom),
		boolAEntero(archivo.Indicadores.TieneIA),
		boolAEntero(archivo.Indicadores.TieneSocial),
		boolAEntero(archivo.Indicadores.EsAdulto),
		archivo.Metadatos.Ubicacion,
		whereFroms,
		ahora,
	}

	if !tieneCargaEnriquecida {
		_, err := a.base.ExecContext(ctx, consulta, args...)
		if err != nil {
			return fmt.Errorf("no se pudo guardar el archivo %q: %w", archivo.Ruta, err)
		}
		return nil
	}

	tx, err := a.base.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar la transaccion de guardado: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, consulta, args...); err != nil {
		return fmt.Errorf("no se pudo guardar el archivo enriquecido %q: %w", archivo.Ruta, err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM palabras_clave WHERE ruta = ?`, archivo.Ruta); err != nil {
		return fmt.Errorf("no se pudieron limpiar las palabras clave de %q: %w", archivo.Ruta, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM etiquetas WHERE ruta = ?`, archivo.Ruta); err != nil {
		return fmt.Errorf("no se pudieron limpiar las etiquetas de %q: %w", archivo.Ruta, err)
	}

	for _, palabra := range archivo.Metadatos.PalabrasClave {
		palabra = strings.TrimSpace(palabra)
		if palabra == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO palabras_clave (ruta, palabra) VALUES (?, ?)`, archivo.Ruta, palabra); err != nil {
			return fmt.Errorf("no se pudo guardar la palabra clave %q: %w", palabra, err)
		}
	}

	for _, etiqueta := range archivo.Metadatos.Sujetos {
		etiqueta = strings.TrimSpace(etiqueta)
		if etiqueta == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO etiquetas (ruta, etiqueta) VALUES (?, ?)`, archivo.Ruta, etiqueta); err != nil {
			return fmt.Errorf("no se pudo guardar la etiqueta %q: %w", etiqueta, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("no se pudo confirmar la transaccion de guardado: %w", err)
	}

	return nil
}

// EliminarArchivo quita un archivo del catalogo persistente.
func (a *Almacen) EliminarArchivo(ctx context.Context, ruta string) error {
	if a == nil || a.base == nil {
		return errors.New("almacen sqlite no inicializado")
	}
	a.muEscritura.Lock()
	defer a.muEscritura.Unlock()
	_, err := a.base.ExecContext(ctx, `DELETE FROM archivos WHERE ruta = ?`, ruta)
	if err != nil {
		return fmt.Errorf("no se pudo eliminar el archivo %q del catalogo: %w", ruta, err)
	}
	return nil
}

// ObtenerArchivoPorRuta recupera un archivo conocido por la base.
func (a *Almacen) ObtenerArchivoPorRuta(ctx context.Context, ruta string) (modelo.Archivo, error) {
	fila := a.base.QueryRowContext(ctx, consultaArchivoPorRuta, ruta)
	return escanearArchivo(fila)
}

// ListarPalabrasClave devuelve las palabras mas frecuentes ya indexadas.
func (a *Almacen) ListarPalabrasClave(ctx context.Context, limite int) ([]string, error) {
	if limite < 1 {
		limite = 100
	}

	filas, err := a.base.QueryContext(ctx, `
SELECT palabra
FROM palabras_clave
GROUP BY palabra
ORDER BY COUNT(*) DESC, palabra ASC
LIMIT ?`, limite)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron consultar las palabras clave: %w", err)
	}
	defer filas.Close()

	var palabras []string
	for filas.Next() {
		var palabra string
		if err := filas.Scan(&palabra); err != nil {
			return nil, fmt.Errorf("no se pudo leer una palabra clave: %w", err)
		}
		palabras = append(palabras, palabra)
	}

	return palabras, filas.Err()
}

// ListarEtiquetas devuelve tags derivados de Subject y palabras clave ya catalogadas.
func (a *Almacen) ListarEtiquetas(ctx context.Context, limite int) ([]string, error) {
	if err := a.asegurarIndiceEtiquetas(ctx); err != nil {
		return nil, err
	}
	if limite < 1 {
		limite = 100
	}

	filas, err := a.base.QueryContext(ctx, `
SELECT etiqueta
FROM (
	SELECT etiqueta AS etiqueta FROM etiquetas
	UNION ALL
	SELECT palabra AS etiqueta FROM palabras_clave
)
GROUP BY etiqueta
ORDER BY COUNT(*) DESC, etiqueta COLLATE NOCASE ASC, etiqueta ASC
LIMIT ?`, limite)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron consultar las etiquetas: %w", err)
	}
	defer filas.Close()

	var etiquetas []string
	for filas.Next() {
		var etiqueta string
		if err := filas.Scan(&etiqueta); err != nil {
			return nil, fmt.Errorf("no se pudo leer una etiqueta: %w", err)
		}
		etiquetas = append(etiquetas, etiqueta)
	}

	return etiquetas, filas.Err()
}

// BuscarEtiquetas busca coincidencias por texto directamente en la base para no
// depender del subconjunto visible ya cargado en memoria.
func (a *Almacen) BuscarEtiquetas(ctx context.Context, consulta string, limite int) ([]string, error) {
	if err := a.asegurarIndiceEtiquetas(ctx); err != nil {
		return nil, err
	}
	consulta = strings.TrimSpace(consulta)
	if consulta == "" {
		return nil, nil
	}
	if limite < 1 {
		limite = 100
	}

	filas, err := a.base.QueryContext(ctx, `
SELECT etiqueta
FROM (
	SELECT etiqueta AS etiqueta
	FROM etiquetas
	WHERE INSTR(LOWER(etiqueta), LOWER(?)) > 0
	UNION ALL
	SELECT palabra AS etiqueta
	FROM palabras_clave
	WHERE INSTR(LOWER(palabra), LOWER(?)) > 0
)
GROUP BY etiqueta
ORDER BY COUNT(*) DESC, etiqueta COLLATE NOCASE ASC, etiqueta ASC
LIMIT ?`, consulta, consulta, limite)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron buscar las etiquetas: %w", err)
	}
	defer filas.Close()

	var etiquetas []string
	for filas.Next() {
		var etiqueta string
		if err := filas.Scan(&etiqueta); err != nil {
			return nil, fmt.Errorf("no se pudo leer una etiqueta buscada: %w", err)
		}
		etiquetas = append(etiquetas, etiqueta)
	}

	return etiquetas, filas.Err()
}

// ListarUbicaciones devuelve ubicaciones nombradas conocidas.
func (a *Almacen) ListarUbicaciones(ctx context.Context, limite int) ([]string, error) {
	if limite < 1 {
		limite = 100
	}

	filas, err := a.base.QueryContext(ctx, `
SELECT ubicacion
FROM archivos
WHERE TRIM(ubicacion) <> ''
	AND es_directorio = 0
GROUP BY ubicacion
ORDER BY COUNT(*) DESC, ubicacion COLLATE NOCASE ASC, ubicacion ASC
LIMIT ?`, limite)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron consultar las ubicaciones: %w", err)
	}
	defer filas.Close()

	var ubicaciones []string
	for filas.Next() {
		var ubicacion string
		if err := filas.Scan(&ubicacion); err != nil {
			return nil, fmt.Errorf("no se pudo leer una ubicacion: %w", err)
		}
		ubicaciones = append(ubicaciones, ubicacion)
	}

	return ubicaciones, filas.Err()
}

// BuscarUbicaciones consulta la base por coincidencias de Location sin
// depender del límite usado por la colección visible.
func (a *Almacen) BuscarUbicaciones(ctx context.Context, consulta string, limite int) ([]string, error) {
	consulta = strings.TrimSpace(consulta)
	if consulta == "" {
		return nil, nil
	}
	if limite < 1 {
		limite = 100
	}

	filas, err := a.base.QueryContext(ctx, `
SELECT ubicacion
FROM archivos
WHERE TRIM(ubicacion) <> ''
	AND es_directorio = 0
	AND INSTR(LOWER(ubicacion), LOWER(?)) > 0
GROUP BY ubicacion
ORDER BY COUNT(*) DESC, ubicacion COLLATE NOCASE ASC, ubicacion ASC
LIMIT ?`, consulta, limite)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron buscar las ubicaciones: %w", err)
	}
	defer filas.Close()

	var ubicaciones []string
	for filas.Next() {
		var ubicacion string
		if err := filas.Scan(&ubicacion); err != nil {
			return nil, fmt.Errorf("no se pudo leer una ubicación buscada: %w", err)
		}
		ubicaciones = append(ubicaciones, ubicacion)
	}

	return ubicaciones, filas.Err()
}

// TieneArchivosConUbicacionSinNombre informa si existen archivos con GPS pero sin valor Location.
func (a *Almacen) TieneArchivosConUbicacionSinNombre(ctx context.Context) (bool, error) {
	var existe int
	err := a.base.QueryRowContext(ctx, `
SELECT 1
FROM archivos
WHERE es_directorio = 0
	AND tiene_gps = 1
	AND TRIM(ubicacion) = ''
LIMIT 1`).Scan(&existe)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("no se pudo consultar si existen ubicaciones sin nombre: %w", err)
	}
	return existe == 1, nil
}

// ListarArchivosPorEtiqueta devuelve un lote paginado usando el catalogo persistente.
func (a *Almacen) ListarArchivosPorEtiqueta(ctx context.Context, etiqueta string, filtros modelo.FiltrosListado, limite, offset int) ([]modelo.Archivo, error) {
	if err := a.asegurarIndiceEtiquetas(ctx); err != nil {
		return nil, err
	}
	return a.listarArchivosCatalogo(ctx,
		`INNER JOIN (
			SELECT ruta FROM etiquetas WHERE LOWER(TRIM(etiqueta)) = LOWER(TRIM(?))
			UNION
			SELECT ruta FROM palabras_clave WHERE LOWER(TRIM(palabra)) = LOWER(TRIM(?))
		) AS coincidencias ON coincidencias.ruta = archivos.ruta`,
		`1 = 1`,
		[]any{etiqueta, etiqueta},
		filtros,
		limite,
		offset,
	)
}

// ListarArchivosPorUbicacion devuelve un lote paginado por el valor exacto de Location.
func (a *Almacen) ListarArchivosPorUbicacion(ctx context.Context, ubicacion string, filtros modelo.FiltrosListado, limite, offset int) ([]modelo.Archivo, error) {
	return a.listarArchivosCatalogo(ctx,
		``,
		`LOWER(TRIM(archivos.ubicacion)) = LOWER(TRIM(?))`,
		[]any{ubicacion},
		filtros,
		limite,
		offset,
	)
}

// ListarArchivosSinUbicacionNombrada devuelve archivos con GPS pero sin texto en Location.
func (a *Almacen) ListarArchivosSinUbicacionNombrada(ctx context.Context, filtros modelo.FiltrosListado, limite, offset int) ([]modelo.Archivo, error) {
	return a.listarArchivosCatalogo(ctx,
		``,
		`archivos.tiene_gps = 1 AND TRIM(archivos.ubicacion) = ''`,
		nil,
		filtros,
		limite,
		offset,
	)
}

func (a *Almacen) listarArchivosCatalogo(ctx context.Context, union string, condicionBase string, argumentos []any, filtros modelo.FiltrosListado, limite, offset int) ([]modelo.Archivo, error) {
	if limite < 1 {
		limite = 100
	}
	if offset < 0 {
		offset = 0
	}

	condiciones := []string{
		`archivos.es_directorio = 0`,
		condicionBase,
	}
	argumentosConsulta := append([]any(nil), argumentos...)

	if !filtros.MostrarOcultos {
		condiciones = append(condiciones, `archivos.es_oculto = 0`)
	}

	if filtros.SoloVideos || filtros.SoloImagenes || filtros.SoloAudio {
		var marcadores []string
		if filtros.SoloVideos {
			marcadores = append(marcadores, `?`)
			argumentosConsulta = append(argumentosConsulta, string(modelo.TipoVideo))
		}
		if filtros.SoloImagenes {
			marcadores = append(marcadores, `?`)
			argumentosConsulta = append(argumentosConsulta, string(modelo.TipoImagen))
		}
		if filtros.SoloAudio {
			marcadores = append(marcadores, `?`)
			argumentosConsulta = append(argumentosConsulta, string(modelo.TipoAudio))
		}
		condiciones = append(condiciones, fmt.Sprintf("archivos.tipo IN (%s)", strings.Join(marcadores, ", ")))
	} else if filtros.SoloMultimedia {
		condiciones = append(condiciones, `archivos.tipo IN (?, ?, ?)`)
		argumentosConsulta = append(argumentosConsulta,
			string(modelo.TipoImagen),
			string(modelo.TipoVideo),
			string(modelo.TipoAudio),
		)
	}

	consulta := fmt.Sprintf(`
SELECT archivos.ruta, archivos.origen, archivos.ruta_padre, archivos.nombre, archivos.tamano, archivos.modificado_unix, archivos.tipo, archivos.es_oculto, archivos.es_directorio,
	archivos.ancho, archivos.alto, archivos.duracion_ms, archivos.metadatos_json, archivos.hash_md5, archivos.hash_sha256, archivos.hash_dhash_imagen, archivos.hash_dhash_video,
	archivos.tiene_gps, archivos.tiene_regiones, archivos.tiene_where_froms, archivos.tiene_ia, archivos.tiene_social, archivos.es_adulto, archivos.ubicacion, archivos.where_froms
FROM archivos
%s
WHERE %s
%s
LIMIT ? OFFSET ?`, union, strings.Join(condiciones, "\n\tAND "), clausulaOrdenListado(filtros))

	argumentosConsulta = append(argumentosConsulta, limite, offset)
	filas, err := a.base.QueryContext(ctx, consulta, argumentosConsulta...)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron consultar los archivos filtrados del catalogo: %w", err)
	}
	defer filas.Close()

	var archivos []modelo.Archivo
	for filas.Next() {
		archivo, err := escanearArchivo(filas)
		if err != nil {
			return nil, err
		}
		archivos = append(archivos, archivo)
	}

	return archivos, filas.Err()
}

func clausulaOrdenListado(filtros modelo.FiltrosListado) string {
	direccionOrden := "ASC"
	if filtros.OrdenDescendente {
		direccionOrden = "DESC"
	}

	if filtros.CriterioOrdenNormalizado() == modelo.CriterioOrdenFechaModificacion {
		return fmt.Sprintf("ORDER BY archivos.modificado_unix %s, archivos.nombre COLLATE NOCASE ASC, archivos.nombre ASC, archivos.ruta ASC", direccionOrden)
	}

	return fmt.Sprintf("ORDER BY archivos.nombre COLLATE NOCASE %s, archivos.nombre %s, archivos.ruta ASC", direccionOrden, direccionOrden)
}

// asegurarIndiceEtiquetas rellena la tabla auxiliar de Subjects una sola vez por proceso.
func (a *Almacen) asegurarIndiceEtiquetas(ctx context.Context) error {
	a.sincronizacionEtiquetas.Do(func() {
		a.errorSincronizacionIndice = a.reconstruirIndiceEtiquetas(ctx)
	})
	return a.errorSincronizacionIndice
}

func (a *Almacen) reconstruirIndiceEtiquetas(ctx context.Context) error {
	a.muEscritura.Lock()
	defer a.muEscritura.Unlock()

	_, err := a.base.ExecContext(ctx, `
INSERT OR IGNORE INTO etiquetas (ruta, etiqueta)
SELECT archivos.ruta, TRIM(CAST(json_each.value AS TEXT))
FROM archivos
JOIN json_each(archivos.metadatos_json, '$.sujetos')
WHERE TRIM(CAST(json_each.value AS TEXT)) <> ''`)
	if err != nil {
		return fmt.Errorf("no se pudo reconstruir el indice de etiquetas: %w", err)
	}

	return nil
}

// ListarGruposDuplicados recupera grupos paginados segun su algoritmo y categoria.
func (a *Almacen) ListarGruposDuplicados(ctx context.Context, tipo modelo.TipoCoincidencia, categoria modelo.CategoriaDuplicados, orden modelo.OrdenDuplicados, limite, offset int) ([]modelo.GrupoDuplicados, error) {
	claveExpr, filtroTipo, err := expresionClaveDuplicados(tipo)
	if err != nil {
		return nil, err
	}

	if limite < 1 {
		limite = 50
	}
	if offset < 0 {
		offset = 0
	}

	condicionCategoria := ""
	switch categoria {
	case modelo.CategoriaDuplicadosLocales:
		condicionCategoria = "WHERE origenes = 1 AND origen_min = 'local'"
	case modelo.CategoriaDuplicadosRemotos:
		condicionCategoria = "WHERE origenes = 1 AND origen_min = 'yandex'"
	case modelo.CategoriaDuplicadosMixtos:
		condicionCategoria = "WHERE origenes > 1"
	}

	ordenSQL := "ORDER BY cantidad DESC, nombre_min ASC"
	switch orden {
	case modelo.OrdenPorEspacioRecuperado:
		ordenSQL = "ORDER BY espacio_recuperable DESC, cantidad DESC, nombre_min ASC"
	case modelo.OrdenAlfabetico:
		ordenSQL = "ORDER BY nombre_min ASC, cantidad DESC"
	}

	consultaGrupos := fmt.Sprintf(`
WITH grupos AS (
	SELECT
		%s AS clave,
		COUNT(*) AS cantidad,
		SUM(tamano) AS tamano_total,
		MAX(tamano) AS tamano_max,
		SUM(tamano) - MAX(tamano) AS espacio_recuperable,
		MIN(nombre) AS nombre_min,
		COUNT(DISTINCT origen) AS origenes,
		MIN(origen) AS origen_min
	FROM archivos
	WHERE es_directorio = 0
		AND tamano > 0
		AND %s <> ''
		%s
	GROUP BY clave
	HAVING COUNT(*) > 1
)
SELECT clave, cantidad, espacio_recuperable, nombre_min
FROM grupos
%s
%s
LIMIT ? OFFSET ?`, claveExpr, claveExpr, filtroTipo, condicionCategoria, ordenSQL)

	filas, err := a.base.QueryContext(ctx, consultaGrupos, limite, offset)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron consultar los grupos duplicados: %w", err)
	}
	defer filas.Close()

	var grupos []modelo.GrupoDuplicados
	claves := make([]string, 0, limite)
	indicesGrupos := make(map[string]int, limite)
	for filas.Next() {
		var clave string
		var cantidad int
		var espacioRecuperable int64
		var nombreMin string
		if err := filas.Scan(&clave, &cantidad, &espacioRecuperable, &nombreMin); err != nil {
			return nil, fmt.Errorf("no se pudo leer un grupo duplicado: %w", err)
		}

		indicesGrupos[clave] = len(grupos)
		claves = append(claves, clave)
		grupos = append(grupos, modelo.GrupoDuplicados{
			Clave:              clave,
			Tipo:               tipo,
			TamanoRecuperable:  espacioRecuperable,
			CantidadElementos:  cantidad,
			NombreRepresentivo: nombreMin,
		})
	}

	if err := filas.Err(); err != nil {
		return nil, err
	}
	if len(grupos) == 0 {
		return grupos, nil
	}

	marcadores := make([]string, 0, len(claves))
	args := make([]any, 0, len(claves))
	for _, clave := range claves {
		marcadores = append(marcadores, "?")
		args = append(args, clave)
	}

	consultaElementos := fmt.Sprintf(`
SELECT %s AS clave_duplicado,
	ruta, origen, ruta_padre, nombre, tamano, modificado_unix, tipo, es_oculto, es_directorio,
	ancho, alto, duracion_ms, metadatos_json, hash_md5, hash_sha256, hash_dhash_imagen, hash_dhash_video,
	tiene_gps, tiene_regiones, tiene_where_froms, tiene_ia, tiene_social, es_adulto, ubicacion, where_froms
FROM archivos
WHERE es_directorio = 0
	AND tamano > 0
	AND %s IN (%s)
ORDER BY clave_duplicado ASC, modificado_unix ASC, nombre ASC`, claveExpr, claveExpr, strings.Join(marcadores, ", "))

	filasElementos, err := a.base.QueryContext(ctx, consultaElementos, args...)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron consultar los elementos agrupados de duplicados: %w", err)
	}
	defer filasElementos.Close()

	for filasElementos.Next() {
		clave, archivo, err := escanearArchivoConClave(filasElementos)
		if err != nil {
			return nil, err
		}
		indice, ok := indicesGrupos[clave]
		if !ok {
			continue
		}
		grupos[indice].Elementos = append(grupos[indice].Elementos, archivo)
	}
	if err := filasElementos.Err(); err != nil {
		return nil, err
	}

	for indice := range grupos {
		grupos[indice].CategoriaSugerida = inferirCategoria(grupos[indice].Elementos)
	}

	return grupos, nil
}

// ObtenerEstadisticasDuplicados resume los grupos encontrados en todos los algoritmos.
func (a *Almacen) ObtenerEstadisticasDuplicados(ctx context.Context) (modelo.EstadisticasDuplicados, error) {
	tipos := []modelo.TipoCoincidencia{
		modelo.CoincidenciaExacta,
		modelo.CoincidenciaParcialImagen,
		modelo.CoincidenciaParcialVideo,
	}

	var estadisticas modelo.EstadisticasDuplicados
	for _, tipo := range tipos {
		for _, categoria := range []modelo.CategoriaDuplicados{
			modelo.CategoriaDuplicadosLocales,
			modelo.CategoriaDuplicadosRemotos,
			modelo.CategoriaDuplicadosMixtos,
		} {
			grupos, err := a.ListarGruposDuplicados(ctx, tipo, categoria, modelo.OrdenPorTamanoGrupo, 1_000_000, 0)
			if err != nil {
				return modelo.EstadisticasDuplicados{}, err
			}
			switch categoria {
			case modelo.CategoriaDuplicadosLocales:
				estadisticas.Locales += len(grupos)
			case modelo.CategoriaDuplicadosRemotos:
				estadisticas.Remotos += len(grupos)
			case modelo.CategoriaDuplicadosMixtos:
				estadisticas.Mixtos += len(grupos)
			}
			estadisticas.TotalGrupos += len(grupos)
		}
	}

	return estadisticas, nil
}

const consultaArchivoPorRuta = `
SELECT ruta, origen, ruta_padre, nombre, tamano, modificado_unix, tipo, es_oculto, es_directorio,
	ancho, alto, duracion_ms, metadatos_json, hash_md5, hash_sha256, hash_dhash_imagen, hash_dhash_video,
	tiene_gps, tiene_regiones, tiene_where_froms, tiene_ia, tiene_social, es_adulto, ubicacion, where_froms
FROM archivos
WHERE ruta = ?`

const consultaUpsertBasico = `
INSERT INTO archivos (
	ruta, origen, ruta_padre, nombre, tamano, modificado_unix, tipo, es_oculto, es_directorio,
	ancho, alto, duracion_ms, metadatos_json, hash_md5, hash_sha256, hash_dhash_imagen, hash_dhash_video,
	tiene_gps, tiene_regiones, tiene_where_froms, tiene_ia, tiene_social, es_adulto, ubicacion, where_froms, ultima_revision_unix
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(ruta) DO UPDATE SET
	origen = excluded.origen,
	ruta_padre = excluded.ruta_padre,
	nombre = excluded.nombre,
	tamano = excluded.tamano,
	modificado_unix = excluded.modificado_unix,
	tipo = excluded.tipo,
	es_oculto = excluded.es_oculto,
	es_directorio = excluded.es_directorio,
	ancho = CASE WHEN excluded.ancho > 0 THEN excluded.ancho ELSE archivos.ancho END,
	alto = CASE WHEN excluded.alto > 0 THEN excluded.alto ELSE archivos.alto END,
	duracion_ms = CASE WHEN excluded.duracion_ms > 0 THEN excluded.duracion_ms ELSE archivos.duracion_ms END,
	metadatos_json = CASE WHEN excluded.metadatos_json <> '{}' THEN excluded.metadatos_json ELSE archivos.metadatos_json END,
	hash_md5 = CASE WHEN excluded.hash_md5 <> '' THEN excluded.hash_md5 ELSE archivos.hash_md5 END,
	hash_sha256 = CASE WHEN excluded.hash_sha256 <> '' THEN excluded.hash_sha256 ELSE archivos.hash_sha256 END,
	hash_dhash_imagen = CASE WHEN excluded.hash_dhash_imagen <> '' THEN excluded.hash_dhash_imagen ELSE archivos.hash_dhash_imagen END,
	hash_dhash_video = CASE WHEN excluded.hash_dhash_video <> '' THEN excluded.hash_dhash_video ELSE archivos.hash_dhash_video END,
	tiene_gps = MAX(archivos.tiene_gps, excluded.tiene_gps),
	tiene_regiones = MAX(archivos.tiene_regiones, excluded.tiene_regiones),
	tiene_where_froms = MAX(archivos.tiene_where_froms, excluded.tiene_where_froms),
	tiene_ia = MAX(archivos.tiene_ia, excluded.tiene_ia),
	tiene_social = MAX(archivos.tiene_social, excluded.tiene_social),
	es_adulto = MAX(archivos.es_adulto, excluded.es_adulto),
	ubicacion = CASE WHEN excluded.ubicacion <> '' THEN excluded.ubicacion ELSE archivos.ubicacion END,
	where_froms = CASE WHEN excluded.where_froms <> '' THEN excluded.where_froms ELSE archivos.where_froms END,
	ultima_revision_unix = CASE WHEN excluded.ultima_revision_unix > 0 THEN excluded.ultima_revision_unix ELSE archivos.ultima_revision_unix END`

const consultaUpsertEnriquecido = `
INSERT INTO archivos (
	ruta, origen, ruta_padre, nombre, tamano, modificado_unix, tipo, es_oculto, es_directorio,
	ancho, alto, duracion_ms, metadatos_json, hash_md5, hash_sha256, hash_dhash_imagen, hash_dhash_video,
	tiene_gps, tiene_regiones, tiene_where_froms, tiene_ia, tiene_social, es_adulto, ubicacion, where_froms, ultima_revision_unix
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(ruta) DO UPDATE SET
	origen = excluded.origen,
	ruta_padre = excluded.ruta_padre,
	nombre = excluded.nombre,
	tamano = excluded.tamano,
	modificado_unix = excluded.modificado_unix,
	tipo = excluded.tipo,
	es_oculto = excluded.es_oculto,
	es_directorio = excluded.es_directorio,
	ancho = excluded.ancho,
	alto = excluded.alto,
	duracion_ms = excluded.duracion_ms,
	metadatos_json = excluded.metadatos_json,
	hash_md5 = excluded.hash_md5,
	hash_sha256 = excluded.hash_sha256,
	hash_dhash_imagen = excluded.hash_dhash_imagen,
	hash_dhash_video = excluded.hash_dhash_video,
	tiene_gps = excluded.tiene_gps,
	tiene_regiones = excluded.tiene_regiones,
	tiene_where_froms = excluded.tiene_where_froms,
	tiene_ia = excluded.tiene_ia,
	tiene_social = excluded.tiene_social,
	es_adulto = excluded.es_adulto,
	ubicacion = excluded.ubicacion,
	where_froms = excluded.where_froms,
	ultima_revision_unix = excluded.ultima_revision_unix`

func escanearArchivo(escaner interface {
	Scan(dest ...any) error
}) (modelo.Archivo, error) {
	return escanearArchivoConPrefijo(escaner)
}

func escanearArchivoConClave(escaner interface {
	Scan(dest ...any) error
}) (string, modelo.Archivo, error) {
	var clave string
	archivo, err := escanearArchivoConPrefijo(escaner, &clave)
	if err != nil {
		return "", modelo.Archivo{}, err
	}
	return clave, archivo, nil
}

func escanearArchivoConPrefijo(escaner interface {
	Scan(dest ...any) error
}, prefijo ...any) (modelo.Archivo, error) {
	var archivo modelo.Archivo
	var origen string
	var tipo string
	var modificadoUnix int64
	var duracionMilis int64
	var esOculto int
	var esDirectorio int
	var tieneGPS int
	var tieneRegiones int
	var tieneWhereFroms int
	var tieneIA int
	var tieneSocial int
	var esAdulto int
	var metadatosJSON string
	var whereFroms string
	var ubicacion string

	destinos := append(prefijo,
		&archivo.Ruta,
		&origen,
		&archivo.RutaPadre,
		&archivo.Nombre,
		&archivo.Tamano,
		&modificadoUnix,
		&tipo,
		&esOculto,
		&esDirectorio,
		&archivo.Ancho,
		&archivo.Alto,
		&duracionMilis,
		&metadatosJSON,
		&archivo.Hashes.MD5,
		&archivo.Hashes.SHA256,
		&archivo.Hashes.DHashImagen,
		&archivo.Hashes.DHashVideo,
		&tieneGPS,
		&tieneRegiones,
		&tieneWhereFroms,
		&tieneIA,
		&tieneSocial,
		&esAdulto,
		&ubicacion,
		&whereFroms,
	)
	err := escaner.Scan(destinos...)
	if err != nil {
		return modelo.Archivo{}, err
	}

	archivo.Origen = modelo.Origen(origen)
	archivo.Tipo = modelo.TipoArchivo(tipo)
	archivo.EsOculto = esOculto == 1
	archivo.EsDirectorio = esDirectorio == 1
	archivo.Modificado = time.Unix(modificadoUnix, 0)
	archivo.Duracion = time.Duration(duracionMilis) * time.Millisecond
	archivo.Indicadores = modelo.IndicadoresArchivo{
		TieneGPS:       tieneGPS == 1,
		TieneRegiones:  tieneRegiones == 1,
		TieneWhereFrom: tieneWhereFroms == 1,
		TieneIA:        tieneIA == 1,
		TieneSocial:    tieneSocial == 1,
		EsAdulto:       esAdulto == 1,
	}

	if metadatosJSON != "" {
		if err := json.Unmarshal([]byte(metadatosJSON), &archivo.Metadatos); err != nil {
			return modelo.Archivo{}, fmt.Errorf("no se pudieron deserializar los metadatos de %q: %w", archivo.Ruta, err)
		}
	}
	if archivo.Metadatos.Ubicacion == "" {
		archivo.Metadatos.Ubicacion = ubicacion
	}
	if len(archivo.Metadatos.WhereFroms) == 0 && whereFroms != "" {
		archivo.Metadatos.WhereFroms = strings.Split(whereFroms, "\n")
	}

	return archivo, nil
}

func expresionClaveDuplicados(tipo modelo.TipoCoincidencia) (string, string, error) {
	switch tipo {
	case modelo.CoincidenciaExacta:
		return `CASE WHEN hash_sha256 <> '' THEN hash_sha256 || '|' || hash_md5 ELSE hash_md5 END`, "", nil
	case modelo.CoincidenciaParcialImagen:
		return `hash_dhash_imagen`, `AND tipo = 'imagen'`, nil
	case modelo.CoincidenciaParcialVideo:
		return `hash_dhash_video`, `AND tipo = 'video'`, nil
	default:
		return "", "", fmt.Errorf("tipo de coincidencia no soportado: %q", tipo)
	}
}

func inferirCategoria(elementos []modelo.Archivo) modelo.CategoriaDuplicados {
	if len(elementos) == 0 {
		return modelo.CategoriaDuplicadosTodos
	}

	encontradoLocal := false
	encontradoRemoto := false
	for _, elemento := range elementos {
		switch elemento.Origen {
		case modelo.OrigenLocal:
			encontradoLocal = true
		case modelo.OrigenYandex:
			encontradoRemoto = true
		}
	}

	switch {
	case encontradoLocal && encontradoRemoto:
		return modelo.CategoriaDuplicadosMixtos
	case encontradoLocal:
		return modelo.CategoriaDuplicadosLocales
	case encontradoRemoto:
		return modelo.CategoriaDuplicadosRemotos
	default:
		return modelo.CategoriaDuplicadosTodos
	}
}

func boolAEntero(valor bool) int {
	if valor {
		return 1
	}
	return 0
}
