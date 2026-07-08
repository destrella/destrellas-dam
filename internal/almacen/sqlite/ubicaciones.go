package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"destrellas-dam/internal/modelo"
)

// ListarUbicacionesGuardadas devuelve nombres de ubicación ya usados, junto con
// su mejor juego de coordenadas y dirección conocido, resolviendo relaciones.
func (a *Almacen) ListarUbicacionesGuardadas(ctx context.Context, limite int) ([]modelo.UbicacionGuardada, error) {
	if limite < 1 {
		limite = 1_000
	}

	filas, err := a.base.QueryContext(ctx, `
WITH nombres_archivos AS (
	SELECT
		MIN(TRIM(ubicacion)) AS nombre,
		LOWER(TRIM(ubicacion)) AS nombre_normalizado,
		COUNT(*) AS cantidad_usos
	FROM archivos
	WHERE es_directorio = 0
		AND TRIM(ubicacion) <> ''
	GROUP BY LOWER(TRIM(ubicacion))
),
nombres_relaciones AS (
	SELECT origen AS nombre, origen_normalizado AS nombre_normalizado, 0 AS cantidad_usos
	FROM ubicaciones_relaciones
	UNION ALL
	SELECT destino AS nombre, destino_normalizado AS nombre_normalizado, 0 AS cantidad_usos
	FROM ubicaciones_relaciones
),
nombres_unificados AS (
	SELECT
		MIN(nombre) AS nombre,
		nombre_normalizado,
		SUM(cantidad_usos) AS cantidad_usos
	FROM (
		SELECT nombre, nombre_normalizado, cantidad_usos FROM nombres_archivos
		UNION ALL
		SELECT nombre, nombre_normalizado, cantidad_usos FROM nombres_relaciones
	)
	GROUP BY nombre_normalizado
),
mejor_uso AS (
	SELECT
		nombre_normalizado,
		latitud,
		longitud,
		ciudad,
		estado,
		pais
	FROM (
		SELECT
			LOWER(TRIM(ubicacion)) AS nombre_normalizado,
			CAST(json_extract(metadatos_json, '$.coordenadas.latitud') AS REAL) AS latitud,
			CAST(json_extract(metadatos_json, '$.coordenadas.longitud') AS REAL) AS longitud,
			TRIM(COALESCE(CAST(json_extract(metadatos_json, '$.ciudad') AS TEXT), '')) AS ciudad,
			TRIM(COALESCE(CAST(json_extract(metadatos_json, '$.estado') AS TEXT), '')) AS estado,
			TRIM(COALESCE(CAST(json_extract(metadatos_json, '$.pais') AS TEXT), '')) AS pais,
			ROW_NUMBER() OVER (
				PARTITION BY LOWER(TRIM(ubicacion))
				ORDER BY
					CASE WHEN json_extract(metadatos_json, '$.coordenadas') IS NOT NULL THEN 0 ELSE 1 END,
					CASE WHEN TRIM(
						COALESCE(CAST(json_extract(metadatos_json, '$.ciudad') AS TEXT), '') ||
						COALESCE(CAST(json_extract(metadatos_json, '$.estado') AS TEXT), '') ||
						COALESCE(CAST(json_extract(metadatos_json, '$.pais') AS TEXT), '')
					) <> '' THEN 0 ELSE 1 END,
					ultima_revision_unix DESC,
					modificado_unix DESC
			) AS fila
		FROM archivos
		WHERE es_directorio = 0
			AND TRIM(ubicacion) <> ''
	)
	WHERE fila = 1
)
SELECT
	nombres_unificados.nombre,
	nombres_unificados.cantidad_usos,
	COALESCE(rel.destino, '') AS relacionado_con,
	COALESCE(destino.latitud, origen.latitud) AS latitud,
	COALESCE(destino.longitud, origen.longitud) AS longitud,
	COALESCE(NULLIF(destino.ciudad, ''), NULLIF(origen.ciudad, ''), '') AS ciudad,
	COALESCE(NULLIF(destino.estado, ''), NULLIF(origen.estado, ''), '') AS estado,
	COALESCE(NULLIF(destino.pais, ''), NULLIF(origen.pais, ''), '') AS pais
FROM nombres_unificados
LEFT JOIN ubicaciones_relaciones AS rel
	ON rel.origen_normalizado = nombres_unificados.nombre_normalizado
LEFT JOIN mejor_uso AS origen
	ON origen.nombre_normalizado = nombres_unificados.nombre_normalizado
LEFT JOIN mejor_uso AS destino
	ON destino.nombre_normalizado = rel.destino_normalizado
ORDER BY
	nombres_unificados.cantidad_usos DESC,
	nombres_unificados.nombre COLLATE NOCASE ASC,
	nombres_unificados.nombre ASC
LIMIT ?`, limite)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron consultar las ubicaciones guardadas: %w", err)
	}
	defer filas.Close()

	var ubicaciones []modelo.UbicacionGuardada
	for filas.Next() {
		var ubicacion modelo.UbicacionGuardada
		var latitud sql.NullFloat64
		var longitud sql.NullFloat64
		if err := filas.Scan(
			&ubicacion.Nombre,
			&ubicacion.CantidadUsos,
			&ubicacion.RelacionadaCon,
			&latitud,
			&longitud,
			&ubicacion.Ciudad,
			&ubicacion.Estado,
			&ubicacion.Pais,
		); err != nil {
			return nil, fmt.Errorf("no se pudo leer una ubicación guardada: %w", err)
		}
		if latitud.Valid || longitud.Valid {
			ubicacion.Coordenadas = &modelo.Coordenadas{
				Latitud:  latitud.Float64,
				Longitud: longitud.Float64,
			}
		}
		ubicaciones = append(ubicaciones, ubicacion)
	}

	return ubicaciones, filas.Err()
}

// ListarUsosUbicacionGuardada agrupa las combinaciones reales con las que un
// nombre de ubicación se ha usado en archivos locales.
func (a *Almacen) ListarUsosUbicacionGuardada(ctx context.Context, nombre string, limite int) ([]modelo.UsoUbicacionGuardada, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return nil, nil
	}
	if limite < 1 {
		limite = 100
	}

	filas, err := a.base.QueryContext(ctx, `
SELECT
	CAST(json_extract(metadatos_json, '$.coordenadas.latitud') AS REAL) AS latitud,
	CAST(json_extract(metadatos_json, '$.coordenadas.longitud') AS REAL) AS longitud,
	TRIM(COALESCE(CAST(json_extract(metadatos_json, '$.ciudad') AS TEXT), '')) AS ciudad,
	TRIM(COALESCE(CAST(json_extract(metadatos_json, '$.estado') AS TEXT), '')) AS estado,
	TRIM(COALESCE(CAST(json_extract(metadatos_json, '$.pais') AS TEXT), '')) AS pais,
	COUNT(*) AS cantidad_usos
FROM archivos
WHERE es_directorio = 0
	AND LOWER(TRIM(ubicacion)) = LOWER(TRIM(?))
GROUP BY latitud, longitud, ciudad, estado, pais
ORDER BY
	cantidad_usos DESC,
	ciudad COLLATE NOCASE ASC,
	estado COLLATE NOCASE ASC,
	pais COLLATE NOCASE ASC
LIMIT ?`, nombre, limite)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron consultar los usos de la ubicación %q: %w", nombre, err)
	}
	defer filas.Close()

	var usos []modelo.UsoUbicacionGuardada
	for filas.Next() {
		var uso modelo.UsoUbicacionGuardada
		var latitud sql.NullFloat64
		var longitud sql.NullFloat64
		uso.Nombre = nombre
		if err := filas.Scan(&latitud, &longitud, &uso.Ciudad, &uso.Estado, &uso.Pais, &uso.CantidadUsos); err != nil {
			return nil, fmt.Errorf("no se pudo leer un uso de ubicación: %w", err)
		}
		if latitud.Valid || longitud.Valid {
			uso.Coordenadas = &modelo.Coordenadas{
				Latitud:  latitud.Float64,
				Longitud: longitud.Float64,
			}
		}
		usos = append(usos, uso)
	}

	return usos, filas.Err()
}

// GuardarRelacionUbicacion enlaza un nombre de ubicación con otro para poder
// reutilizar sus coordenadas y dirección al seleccionarlo desde el formulario.
func (a *Almacen) GuardarRelacionUbicacion(ctx context.Context, origen, destino string) error {
	if a == nil || a.base == nil {
		return errors.New("almacen sqlite no inicializado")
	}

	origen = strings.TrimSpace(origen)
	destino = strings.TrimSpace(destino)
	if origen == "" || destino == "" {
		return errors.New("el origen y el destino de la relación son obligatorios")
	}

	origenNormalizado := normalizarNombreUbicacion(origen)
	destinoNormalizado := normalizarNombreUbicacion(destino)
	if origenNormalizado == "" || destinoNormalizado == "" {
		return errors.New("el origen y el destino de la relación son obligatorios")
	}
	if origenNormalizado == destinoNormalizado {
		return errors.New("una ubicación no puede relacionarse consigo misma")
	}

	a.muEscritura.Lock()
	defer a.muEscritura.Unlock()

	_, err := a.base.ExecContext(ctx, `
INSERT INTO ubicaciones_relaciones (
	origen, origen_normalizado, destino, destino_normalizado, actualizado_unix
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(origen_normalizado) DO UPDATE SET
	origen = excluded.origen,
	destino = excluded.destino,
	destino_normalizado = excluded.destino_normalizado,
	actualizado_unix = excluded.actualizado_unix`,
		origen,
		origenNormalizado,
		destino,
		destinoNormalizado,
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("no se pudo guardar la relación de ubicación %q -> %q: %w", origen, destino, err)
	}
	return nil
}

// EliminarRelacionUbicacion quita la relación saliente asociada a un nombre.
func (a *Almacen) EliminarRelacionUbicacion(ctx context.Context, origen string) error {
	if a == nil || a.base == nil {
		return errors.New("almacen sqlite no inicializado")
	}

	origenNormalizado := normalizarNombreUbicacion(origen)
	if origenNormalizado == "" {
		return errors.New("el origen de la relación es obligatorio")
	}

	a.muEscritura.Lock()
	defer a.muEscritura.Unlock()

	_, err := a.base.ExecContext(ctx, `DELETE FROM ubicaciones_relaciones WHERE origen_normalizado = ?`, origenNormalizado)
	if err != nil {
		return fmt.Errorf("no se pudo eliminar la relación de ubicación %q: %w", origen, err)
	}
	return nil
}

func normalizarNombreUbicacion(nombre string) string {
	return strings.ToLower(strings.TrimSpace(nombre))
}
