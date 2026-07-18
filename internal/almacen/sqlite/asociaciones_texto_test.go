package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestGuardarYListarAsociacionesTexto(t *testing.T) {
	t.Parallel()

	almacen := nuevoAlmacenPruebaAsociacionesTexto(t)

	asociacion, err := almacen.GuardarAsociacionTexto(context.Background(), 0, []string{"cadenaTexto1"}, []string{"Texto A", "Texto A2"})
	if err != nil {
		t.Fatalf("no se pudo guardar la asociación de texto: %v", err)
	}
	if asociacion.ID == 0 {
		t.Fatal("la asociación guardada debería tener identificador")
	}

	listado, err := almacen.ListarAsociacionesTexto(context.Background(), 100)
	if err != nil {
		t.Fatalf("no se pudieron listar las asociaciones de texto: %v", err)
	}
	if len(listado) != 1 {
		t.Fatalf("se esperaba 1 asociación de texto, se obtuvieron %d", len(listado))
	}

	if !listasTextoEquivalentes(listado[0].Originales, []string{"cadenaTexto1"}) {
		t.Fatalf("originales inesperadas: %+v", listado[0].Originales)
	}
	if !listasTextoEquivalentes(listado[0].Sugeridas, []string{"Texto A", "Texto A2"}) {
		t.Fatalf("sugeridas inesperadas: %+v", listado[0].Sugeridas)
	}
}

func TestGuardarAsociacionTextoFusionaGruposCuandoReutilizaUnaCadenaOriginal(t *testing.T) {
	t.Parallel()

	almacen := nuevoAlmacenPruebaAsociacionesTexto(t)

	primera, err := almacen.GuardarAsociacionTexto(context.Background(), 0, []string{"cadenaTxt1", "texto1", "texto2"}, []string{"Cadena C"})
	if err != nil {
		t.Fatalf("no se pudo guardar la primera asociación: %v", err)
	}

	segunda, err := almacen.GuardarAsociacionTexto(context.Background(), 0, []string{"texto2"}, []string{"Cadena D"})
	if err != nil {
		t.Fatalf("no se pudo guardar la segunda asociación solapada: %v", err)
	}

	if segunda.ID != primera.ID {
		t.Fatalf("la asociación solapada debería fusionarse con la existente. ids: %d y %d", primera.ID, segunda.ID)
	}

	listado, err := almacen.ListarAsociacionesTexto(context.Background(), 100)
	if err != nil {
		t.Fatalf("no se pudieron listar las asociaciones fusionadas: %v", err)
	}
	if len(listado) != 1 {
		t.Fatalf("debería existir un solo grupo fusionado, se obtuvieron %d", len(listado))
	}

	if !listasTextoEquivalentes(listado[0].Originales, []string{"cadenaTxt1", "texto1", "texto2"}) {
		t.Fatalf("las cadenas originales del grupo fusionado no son correctas: %+v", listado[0].Originales)
	}
	if !listasTextoEquivalentes(listado[0].Sugeridas, []string{"Cadena C", "Cadena D"}) {
		t.Fatalf("las cadenas sugeridas del grupo fusionado no son correctas: %+v", listado[0].Sugeridas)
	}
}

func TestGuardarAsociacionTextoToleraIDSoletoEliminado(t *testing.T) {
	t.Parallel()

	almacen := nuevoAlmacenPruebaAsociacionesTexto(t)

	creada, err := almacen.GuardarAsociacionTexto(context.Background(), 0, []string{"cadenaTexto1"}, []string{"Texto A"})
	if err != nil {
		t.Fatalf("no se pudo guardar la asociación inicial: %v", err)
	}

	if err := almacen.EliminarAsociacionTexto(context.Background(), creada.ID); err != nil {
		t.Fatalf("no se pudo eliminar la asociación inicial: %v", err)
	}

	recreada, err := almacen.GuardarAsociacionTexto(context.Background(), creada.ID, []string{"cadenaTexto1"}, []string{"Texto B"})
	if err != nil {
		t.Fatalf("el guardado debería tolerar un id eliminado y recrear la asociación: %v", err)
	}
	if recreada.ID == 0 {
		t.Fatal("la asociación recreada debería tener identificador")
	}

	listado, err := almacen.ListarAsociacionesTexto(context.Background(), 100)
	if err != nil {
		t.Fatalf("no se pudieron listar las asociaciones recreadas: %v", err)
	}
	if len(listado) != 1 {
		t.Fatalf("se esperaba 1 asociación tras recrearla, se obtuvieron %d", len(listado))
	}
	if !listasTextoEquivalentes(listado[0].Originales, []string{"cadenaTexto1"}) {
		t.Fatalf("originales inesperadas tras recrear la asociación: %+v", listado[0].Originales)
	}
	if !listasTextoEquivalentes(listado[0].Sugeridas, []string{"Texto B"}) {
		t.Fatalf("sugeridas inesperadas tras recrear la asociación: %+v", listado[0].Sugeridas)
	}
}

func TestGuardarAsociacionTextoLimpiaFilasHuerfanasPrevias(t *testing.T) {
	t.Parallel()

	almacen := nuevoAlmacenPruebaAsociacionesTexto(t)
	ctx := context.Background()

	conn, err := almacen.base.Conn(ctx)
	if err != nil {
		t.Fatalf("no se pudo abrir una conexión dedicada para la prueba: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("no se pudo desactivar temporalmente la validación de claves foráneas: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `
INSERT INTO asociaciones_texto_originales (asociacion_id, valor, valor_normalizado)
VALUES (999, 'cadenaTexto1', 'cadenatexto1')`); err != nil {
		t.Fatalf("no se pudo insertar la fila huérfana de originales: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `
INSERT INTO asociaciones_texto_sugeridas (asociacion_id, valor, valor_normalizado)
VALUES (999, 'Texto A', 'texto a')`); err != nil {
		t.Fatalf("no se pudo insertar la fila huérfana de sugeridas: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("no se pudo restaurar la validación de claves foráneas: %v", err)
	}

	guardada, err := almacen.GuardarAsociacionTexto(ctx, 0, []string{"cadenaTexto1"}, []string{"Texto B"})
	if err != nil {
		t.Fatalf("el guardado debería limpiar huérfanos previos y continuar: %v", err)
	}
	if guardada.ID == 0 {
		t.Fatal("la asociación guardada debería tener identificador")
	}

	listado, err := almacen.ListarAsociacionesTexto(ctx, 100)
	if err != nil {
		t.Fatalf("no se pudieron listar las asociaciones tras limpiar huérfanos: %v", err)
	}
	if len(listado) != 1 {
		t.Fatalf("se esperaba una única asociación válida, se obtuvieron %d", len(listado))
	}

	var cantidadOriginalesHuerfanos int
	if err := almacen.base.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM asociaciones_texto_originales AS originales
LEFT JOIN asociaciones_texto AS grupos ON grupos.id = originales.asociacion_id
WHERE grupos.id IS NULL`).Scan(&cantidadOriginalesHuerfanos); err != nil && err != sql.ErrNoRows {
		t.Fatalf("no se pudo contar los originales huérfanos: %v", err)
	}
	if cantidadOriginalesHuerfanos != 0 {
		t.Fatalf("no deberían quedar originales huérfanos, se encontraron %d", cantidadOriginalesHuerfanos)
	}
}

func nuevoAlmacenPruebaAsociacionesTexto(t *testing.T) *Almacen {
	t.Helper()

	rutaBase := filepath.Join(t.TempDir(), "catalogo.sqlite")
	almacen, err := Nuevo(rutaBase)
	if err != nil {
		t.Fatalf("no se pudo crear el almacén sqlite de prueba: %v", err)
	}
	t.Cleanup(func() {
		_ = almacen.Cerrar()
	})
	return almacen
}

func listasTextoEquivalentes(obtenida, esperada []string) bool {
	if len(obtenida) != len(esperada) {
		return false
	}

	conjunto := make(map[string]struct{}, len(obtenida))
	for _, valor := range obtenida {
		conjunto[normalizarValorAsociacionTexto(valor)] = struct{}{}
	}
	for _, valor := range esperada {
		if _, existe := conjunto[normalizarValorAsociacionTexto(valor)]; !existe {
			return false
		}
	}
	return true
}
