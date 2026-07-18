package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"destrellas-dam/internal/modelo"
)

// ListarAsociacionesTexto devuelve los grupos de asociaciones configurados.
func (a *Almacen) ListarAsociacionesTexto(ctx context.Context, limite int) ([]modelo.AsociacionTexto, error) {
	if a == nil || a.base == nil {
		return nil, errors.New("almacen sqlite no inicializado")
	}
	if limite < 1 {
		limite = 1_000
	}
	if err := a.limpiarAsociacionesTextoHuerfanas(ctx); err != nil {
		return nil, err
	}

	filas, err := a.base.QueryContext(ctx, `
SELECT
	grupos.id,
	COALESCE((
		SELECT json_group_array(valor)
		FROM (
			SELECT valor
			FROM asociaciones_texto_originales
			WHERE asociacion_id = grupos.id
			ORDER BY valor COLLATE NOCASE ASC, valor ASC
		)
	), '[]') AS originales_json,
	COALESCE((
		SELECT json_group_array(valor)
		FROM (
			SELECT valor
			FROM asociaciones_texto_sugeridas
			WHERE asociacion_id = grupos.id
			ORDER BY valor COLLATE NOCASE ASC, valor ASC
		)
	), '[]') AS sugeridas_json
FROM asociaciones_texto AS grupos
WHERE EXISTS (
	SELECT 1 FROM asociaciones_texto_originales WHERE asociacion_id = grupos.id
)
ORDER BY
	COALESCE((
		SELECT MIN(valor)
		FROM asociaciones_texto_originales
		WHERE asociacion_id = grupos.id
	), '') COLLATE NOCASE ASC,
	COALESCE((
		SELECT MIN(valor)
		FROM asociaciones_texto_originales
		WHERE asociacion_id = grupos.id
	), '') ASC,
	grupos.id ASC
LIMIT ?`, limite)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar las asociaciones de texto: %w", err)
	}
	defer filas.Close()

	var asociaciones []modelo.AsociacionTexto
	for filas.Next() {
		var (
			asociacion     modelo.AsociacionTexto
			originalesJSON string
			sugeridasJSON  string
		)
		if err := filas.Scan(&asociacion.ID, &originalesJSON, &sugeridasJSON); err != nil {
			return nil, fmt.Errorf("no se pudo leer una asociación de texto: %w", err)
		}

		asociacion.Originales, err = decodificarListaJSONTexto(originalesJSON)
		if err != nil {
			return nil, fmt.Errorf("no se pudieron leer las cadenas originales de la asociación %d: %w", asociacion.ID, err)
		}
		asociacion.Sugeridas, err = decodificarListaJSONTexto(sugeridasJSON)
		if err != nil {
			return nil, fmt.Errorf("no se pudieron leer las cadenas sugeridas de la asociación %d: %w", asociacion.ID, err)
		}
		asociaciones = append(asociaciones, asociacion)
	}

	return asociaciones, filas.Err()
}

// GuardarAsociacionTexto crea o actualiza una asociación. Si alguno de los
// textos originales ya existe en otro grupo, la información se fusiona en un
// único grupo para preservar la unicidad de las cadenas originales.
func (a *Almacen) GuardarAsociacionTexto(ctx context.Context, id int64, originales, sugeridas []string) (modelo.AsociacionTexto, error) {
	if a == nil || a.base == nil {
		return modelo.AsociacionTexto{}, errors.New("almacen sqlite no inicializado")
	}

	originales = normalizarListaAsociacionTexto(originales)
	sugeridas = normalizarListaAsociacionTexto(sugeridas)
	if len(originales) == 0 {
		return modelo.AsociacionTexto{}, errors.New("debe existir al menos una cadena original")
	}
	if len(sugeridas) == 0 {
		return modelo.AsociacionTexto{}, errors.New("debe existir al menos una cadena sugerida")
	}

	a.muEscritura.Lock()
	defer a.muEscritura.Unlock()

	tx, err := a.base.BeginTx(ctx, nil)
	if err != nil {
		return modelo.AsociacionTexto{}, fmt.Errorf("no se pudo iniciar la transacción de asociaciones de texto: %w", err)
	}
	defer tx.Rollback()
	if err := limpiarAsociacionesTextoHuerfanasTx(ctx, tx); err != nil {
		return modelo.AsociacionTexto{}, err
	}

	idObjetivo, finalesOriginales, finalesSugeridas, gruposAEliminar, err := prepararGuardadoAsociacionTextoTx(ctx, tx, id, originales, sugeridas)
	if err != nil {
		return modelo.AsociacionTexto{}, err
	}

	if err := eliminarGruposAsociacionTextoTx(ctx, tx, gruposAEliminar); err != nil {
		return modelo.AsociacionTexto{}, err
	}
	if err := reemplazarContenidoAsociacionTextoTx(ctx, tx, idObjetivo, finalesOriginales, finalesSugeridas); err != nil {
		return modelo.AsociacionTexto{}, err
	}

	if err := tx.Commit(); err != nil {
		return modelo.AsociacionTexto{}, fmt.Errorf("no se pudo confirmar la asociación de texto: %w", err)
	}

	asociacion, err := a.obtenerAsociacionTextoPorID(ctx, idObjetivo)
	if err != nil {
		return modelo.AsociacionTexto{}, err
	}
	return asociacion, nil
}

// EliminarAsociacionTexto borra un grupo completo.
func (a *Almacen) EliminarAsociacionTexto(ctx context.Context, id int64) error {
	if a == nil || a.base == nil {
		return errors.New("almacen sqlite no inicializado")
	}
	if id < 1 {
		return errors.New("la asociación de texto indicada no es válida")
	}

	a.muEscritura.Lock()
	defer a.muEscritura.Unlock()

	tx, err := a.base.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar la transacción para eliminar la asociación de texto %d: %w", id, err)
	}
	defer tx.Rollback()

	if err := eliminarContenidoAsociacionTextoTx(ctx, tx, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM asociaciones_texto WHERE id = ?`, id); err != nil {
		return fmt.Errorf("no se pudo eliminar la asociación de texto %d: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("no se pudo confirmar la eliminación de la asociación de texto %d: %w", id, err)
	}
	return nil
}

func (a *Almacen) limpiarAsociacionesTextoHuerfanas(ctx context.Context) error {
	a.muEscritura.Lock()
	defer a.muEscritura.Unlock()

	if err := limpiarAsociacionesTextoHuerfanasExec(ctx, a.base); err != nil {
		return err
	}
	return nil
}

type ejecutorAsociacionTexto interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func limpiarAsociacionesTextoHuerfanasTx(ctx context.Context, tx *sql.Tx) error {
	return limpiarAsociacionesTextoHuerfanasExec(ctx, tx)
}

func limpiarAsociacionesTextoHuerfanasExec(ctx context.Context, ejecutor ejecutorAsociacionTexto) error {
	// Limpiamos huérfanos explícitamente porque SQLite puede no propagar
	// cascadas si una conexión nueva no heredó el PRAGMA de claves foráneas.
	if _, err := ejecutor.ExecContext(ctx, `
DELETE FROM asociaciones_texto_originales
WHERE asociacion_id NOT IN (SELECT id FROM asociaciones_texto)`); err != nil {
		return fmt.Errorf("no se pudieron limpiar las cadenas originales huérfanas de asociaciones de texto: %w", err)
	}
	if _, err := ejecutor.ExecContext(ctx, `
DELETE FROM asociaciones_texto_sugeridas
WHERE asociacion_id NOT IN (SELECT id FROM asociaciones_texto)`); err != nil {
		return fmt.Errorf("no se pudieron limpiar las cadenas sugeridas huérfanas de asociaciones de texto: %w", err)
	}
	return nil
}

func eliminarContenidoAsociacionTextoTx(ctx context.Context, tx *sql.Tx, id int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM asociaciones_texto_originales WHERE asociacion_id = ?`, id); err != nil {
		return fmt.Errorf("no se pudieron eliminar las cadenas originales de la asociación %d: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM asociaciones_texto_sugeridas WHERE asociacion_id = ?`, id); err != nil {
		return fmt.Errorf("no se pudieron eliminar las cadenas sugeridas de la asociación %d: %w", id, err)
	}
	return nil
}

func prepararGuardadoAsociacionTextoTx(ctx context.Context, tx *sql.Tx, id int64, originales, sugeridas []string) (int64, []string, []string, []int64, error) {
	gruposSolapados, err := gruposAsociacionPorOriginalesTx(ctx, tx, originales)
	if err != nil {
		return 0, nil, nil, nil, err
	}

	if id > 0 {
		existe, err := existeAsociacionTextoTx(ctx, tx, id)
		if err != nil {
			return 0, nil, nil, nil, err
		}
		if !existe {
			// Si la UI conserva temporalmente un id ya borrado, tratamos el guardado
			// como un alta normal o una fusión con otro grupo existente.
			id = 0
		}
	}

	if id > 0 {
		finalesOriginales := append([]string(nil), originales...)
		finalesSugeridas := append([]string(nil), sugeridas...)
		var gruposAEliminar []int64
		for _, candidato := range gruposSolapados {
			if candidato == id {
				continue
			}
			gruposAEliminar = append(gruposAEliminar, candidato)
		}
		if len(gruposAEliminar) > 0 {
			gruposFusionados, err := cargarAsociacionesTextoPorIDsTx(ctx, tx, gruposAEliminar)
			if err != nil {
				return 0, nil, nil, nil, err
			}
			for _, grupo := range gruposFusionados {
				finalesOriginales = combinarListasAsociacionTexto(finalesOriginales, grupo.Originales)
				finalesSugeridas = combinarListasAsociacionTexto(finalesSugeridas, grupo.Sugeridas)
			}
		}
		return id, finalesOriginales, finalesSugeridas, gruposAEliminar, nil
	}

	if len(gruposSolapados) > 0 {
		idObjetivo := gruposSolapados[0]
		gruposFusionados, err := cargarAsociacionesTextoPorIDsTx(ctx, tx, gruposSolapados)
		if err != nil {
			return 0, nil, nil, nil, err
		}

		var finalesOriginales []string
		var finalesSugeridas []string
		for _, grupo := range gruposFusionados {
			finalesOriginales = combinarListasAsociacionTexto(finalesOriginales, grupo.Originales)
			finalesSugeridas = combinarListasAsociacionTexto(finalesSugeridas, grupo.Sugeridas)
		}
		finalesOriginales = combinarListasAsociacionTexto(finalesOriginales, originales)
		finalesSugeridas = combinarListasAsociacionTexto(finalesSugeridas, sugeridas)

		var gruposAEliminar []int64
		for _, candidato := range gruposSolapados {
			if candidato == idObjetivo {
				continue
			}
			gruposAEliminar = append(gruposAEliminar, candidato)
		}
		return idObjetivo, finalesOriginales, finalesSugeridas, gruposAEliminar, nil
	}

	idObjetivo, err := crearAsociacionTextoTx(ctx, tx)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	return idObjetivo, append([]string(nil), originales...), append([]string(nil), sugeridas...), nil, nil
}

func crearAsociacionTextoTx(ctx context.Context, tx *sql.Tx) (int64, error) {
	resultado, err := tx.ExecContext(ctx, `INSERT INTO asociaciones_texto (actualizado_unix) VALUES (?)`, time.Now().Unix())
	if err != nil {
		return 0, fmt.Errorf("no se pudo crear el grupo de asociación de texto: %w", err)
	}
	id, err := resultado.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("no se pudo obtener el identificador de la asociación de texto creada: %w", err)
	}
	return id, nil
}

func existeAsociacionTextoTx(ctx context.Context, tx *sql.Tx, id int64) (bool, error) {
	var existe int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM asociaciones_texto WHERE id = ? LIMIT 1`, id).Scan(&existe)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("no se pudo comprobar la asociación de texto %d: %w", id, err)
	}
	return existe == 1, nil
}

func gruposAsociacionPorOriginalesTx(ctx context.Context, tx *sql.Tx, originales []string) ([]int64, error) {
	normalizados := make([]string, 0, len(originales))
	for _, original := range originales {
		clave := normalizarValorAsociacionTexto(original)
		if clave == "" {
			continue
		}
		normalizados = append(normalizados, clave)
	}
	if len(normalizados) == 0 {
		return nil, nil
	}

	marcadores := make([]string, len(normalizados))
	argumentos := make([]any, len(normalizados))
	for indice, valor := range normalizados {
		marcadores[indice] = "?"
		argumentos[indice] = valor
	}

	filas, err := tx.QueryContext(ctx, fmt.Sprintf(`
SELECT DISTINCT originales.asociacion_id
FROM asociaciones_texto_originales AS originales
INNER JOIN asociaciones_texto AS grupos
	ON grupos.id = originales.asociacion_id
WHERE valor_normalizado IN (%s)
ORDER BY originales.asociacion_id ASC`, strings.Join(marcadores, ", ")), argumentos...)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron consultar las asociaciones de texto solapadas: %w", err)
	}
	defer filas.Close()

	var ids []int64
	for filas.Next() {
		var id int64
		if err := filas.Scan(&id); err != nil {
			return nil, fmt.Errorf("no se pudo leer un grupo de asociación de texto: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, filas.Err()
}

func cargarAsociacionesTextoPorIDsTx(ctx context.Context, tx *sql.Tx, ids []int64) ([]modelo.AsociacionTexto, error) {
	asociaciones := make([]modelo.AsociacionTexto, 0, len(ids))
	for _, id := range ids {
		asociacion, err := obtenerAsociacionTextoTx(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		asociaciones = append(asociaciones, asociacion)
	}
	return asociaciones, nil
}

func reemplazarContenidoAsociacionTextoTx(ctx context.Context, tx *sql.Tx, id int64, originales, sugeridas []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM asociaciones_texto_originales WHERE asociacion_id = ?`, id); err != nil {
		return fmt.Errorf("no se pudieron limpiar las cadenas originales de la asociación %d: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM asociaciones_texto_sugeridas WHERE asociacion_id = ?`, id); err != nil {
		return fmt.Errorf("no se pudieron limpiar las cadenas sugeridas de la asociación %d: %w", id, err)
	}

	for _, valor := range originales {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO asociaciones_texto_originales (asociacion_id, valor, valor_normalizado)
VALUES (?, ?, ?)`, id, valor, normalizarValorAsociacionTexto(valor)); err != nil {
			return fmt.Errorf("no se pudo guardar la cadena original %q: %w", valor, err)
		}
	}
	for _, valor := range sugeridas {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO asociaciones_texto_sugeridas (asociacion_id, valor, valor_normalizado)
VALUES (?, ?, ?)`, id, valor, normalizarValorAsociacionTexto(valor)); err != nil {
			return fmt.Errorf("no se pudo guardar la cadena sugerida %q: %w", valor, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE asociaciones_texto SET actualizado_unix = ? WHERE id = ?`, time.Now().Unix(), id); err != nil {
		return fmt.Errorf("no se pudo actualizar la marca de tiempo de la asociación %d: %w", id, err)
	}
	return nil
}

func eliminarGruposAsociacionTextoTx(ctx context.Context, tx *sql.Tx, ids []int64) error {
	for _, id := range ids {
		if err := eliminarContenidoAsociacionTextoTx(ctx, tx, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM asociaciones_texto WHERE id = ?`, id); err != nil {
			return fmt.Errorf("no se pudo fusionar la asociación de texto %d: %w", id, err)
		}
	}
	return nil
}

func (a *Almacen) obtenerAsociacionTextoPorID(ctx context.Context, id int64) (modelo.AsociacionTexto, error) {
	return obtenerAsociacionTextoTx(ctx, a.base, id)
}

type consultorAsociacionTexto interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func obtenerAsociacionTextoTx(ctx context.Context, consultor consultorAsociacionTexto, id int64) (modelo.AsociacionTexto, error) {
	var (
		asociacion     modelo.AsociacionTexto
		originalesJSON string
		sugeridasJSON  string
	)
	asociacion.ID = id
	err := consultor.QueryRowContext(ctx, `
SELECT
	COALESCE((
		SELECT json_group_array(valor)
		FROM (
			SELECT valor
			FROM asociaciones_texto_originales
			WHERE asociacion_id = ?
			ORDER BY valor COLLATE NOCASE ASC, valor ASC
		)
	), '[]') AS originales_json,
	COALESCE((
		SELECT json_group_array(valor)
		FROM (
			SELECT valor
			FROM asociaciones_texto_sugeridas
			WHERE asociacion_id = ?
			ORDER BY valor COLLATE NOCASE ASC, valor ASC
		)
	), '[]') AS sugeridas_json
FROM asociaciones_texto
WHERE id = ?`, id, id, id).Scan(&originalesJSON, &sugeridasJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return modelo.AsociacionTexto{}, fmt.Errorf("la asociación de texto %d no existe", id)
	}
	if err != nil {
		return modelo.AsociacionTexto{}, fmt.Errorf("no se pudo leer la asociación de texto %d: %w", id, err)
	}

	asociacion.Originales, err = decodificarListaJSONTexto(originalesJSON)
	if err != nil {
		return modelo.AsociacionTexto{}, fmt.Errorf("no se pudieron leer las cadenas originales de la asociación %d: %w", id, err)
	}
	asociacion.Sugeridas, err = decodificarListaJSONTexto(sugeridasJSON)
	if err != nil {
		return modelo.AsociacionTexto{}, fmt.Errorf("no se pudieron leer las cadenas sugeridas de la asociación %d: %w", id, err)
	}
	return asociacion, nil
}

func decodificarListaJSONTexto(texto string) ([]string, error) {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return nil, nil
	}

	var valores []string
	if err := json.Unmarshal([]byte(texto), &valores); err != nil {
		return nil, err
	}
	return normalizarListaAsociacionTexto(valores), nil
}

func normalizarListaAsociacionTexto(valores []string) []string {
	vistos := make(map[string]struct{}, len(valores))
	salida := make([]string, 0, len(valores))
	for _, valor := range valores {
		valor = strings.TrimSpace(valor)
		if valor == "" {
			continue
		}
		clave := normalizarValorAsociacionTexto(valor)
		if clave == "" {
			continue
		}
		if _, existe := vistos[clave]; existe {
			continue
		}
		vistos[clave] = struct{}{}
		salida = append(salida, valor)
	}
	return salida
}

func combinarListasAsociacionTexto(base, extras []string) []string {
	base = normalizarListaAsociacionTexto(base)
	vistos := make(map[string]struct{}, len(base)+len(extras))
	for _, valor := range base {
		vistos[normalizarValorAsociacionTexto(valor)] = struct{}{}
	}
	for _, valor := range extras {
		valor = strings.TrimSpace(valor)
		clave := normalizarValorAsociacionTexto(valor)
		if clave == "" {
			continue
		}
		if _, existe := vistos[clave]; existe {
			continue
		}
		vistos[clave] = struct{}{}
		base = append(base, valor)
	}
	return base
}

func normalizarValorAsociacionTexto(valor string) string {
	return strings.ToLower(strings.TrimSpace(valor))
}
