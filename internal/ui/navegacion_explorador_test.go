package ui

import (
	"testing"

	"gioui.org/io/key"
	"gioui.org/layout"

	"destrellas-dam/internal/modelo"
)

func TestIndiceDestinoNavegacionExploradorLista(t *testing.T) {
	t.Parallel()

	filas := []filaNavegacionExplorador{
		{Visual: 0, Indices: []int{0}},
		{Visual: 1, Indices: []int{1}},
		{Visual: 2, Indices: []int{2}},
	}

	if destino, fila, ok := indiceDestinoNavegacionExplorador(filas, 1, key.NameUpArrow); !ok || destino != 0 || fila != 0 {
		t.Fatalf("flecha arriba en lista inesperada: destino=%d fila=%d ok=%v", destino, fila, ok)
	}
	if destino, fila, ok := indiceDestinoNavegacionExplorador(filas, 1, key.NameDownArrow); !ok || destino != 2 || fila != 2 {
		t.Fatalf("flecha abajo en lista inesperada: destino=%d fila=%d ok=%v", destino, fila, ok)
	}
	if destino, fila, ok := indiceDestinoNavegacionExplorador(filas, 1, key.NameLeftArrow); !ok || destino != 0 || fila != 0 {
		t.Fatalf("flecha izquierda en lista inesperada: destino=%d fila=%d ok=%v", destino, fila, ok)
	}
	if destino, fila, ok := indiceDestinoNavegacionExplorador(filas, 1, key.NameRightArrow); !ok || destino != 2 || fila != 2 {
		t.Fatalf("flecha derecha en lista inesperada: destino=%d fila=%d ok=%v", destino, fila, ok)
	}
}

func TestIndiceDestinoNavegacionExploradorGaleria(t *testing.T) {
	t.Parallel()

	filas := []filaNavegacionExplorador{
		{Visual: 0, Indices: []int{0, 1, 2}},
		{Visual: 1, Indices: []int{3, 4, 5}},
		{Visual: 2, Indices: []int{6}},
	}

	if destino, fila, ok := indiceDestinoNavegacionExplorador(filas, 1, key.NameDownArrow); !ok || destino != 4 || fila != 1 {
		t.Fatalf("flecha abajo en galería inesperada: destino=%d fila=%d ok=%v", destino, fila, ok)
	}
	if destino, fila, ok := indiceDestinoNavegacionExplorador(filas, 4, key.NameUpArrow); !ok || destino != 1 || fila != 0 {
		t.Fatalf("flecha arriba en galería inesperada: destino=%d fila=%d ok=%v", destino, fila, ok)
	}
	if destino, fila, ok := indiceDestinoNavegacionExplorador(filas, 2, key.NameRightArrow); !ok || destino != 3 || fila != 1 {
		t.Fatalf("flecha derecha con salto de fila inesperada: destino=%d fila=%d ok=%v", destino, fila, ok)
	}
	if destino, fila, ok := indiceDestinoNavegacionExplorador(filas, 3, key.NameLeftArrow); !ok || destino != 2 || fila != 0 {
		t.Fatalf("flecha izquierda con retroceso de fila inesperada: destino=%d fila=%d ok=%v", destino, fila, ok)
	}
	if destino, fila, ok := indiceDestinoNavegacionExplorador(filas, 5, key.NameDownArrow); !ok || destino != 6 || fila != 2 {
		t.Fatalf("flecha abajo hacia fila corta inesperada: destino=%d fila=%d ok=%v", destino, fila, ok)
	}
}

func TestFilasNavegacionExploradorRespetaGruposRecursivos(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		origenListado: origenListadoCarpeta,
		filtros: modelo.FiltrosListado{
			VistaGaleria: true,
			Recursivo:    true,
		},
		elementos: []modelo.Archivo{
			{Ruta: "/a/uno.jpg", RutaPadre: "/a", Tipo: modelo.TipoImagen},
			{Ruta: "/a/dos.jpg", RutaPadre: "/a", Tipo: modelo.TipoImagen},
			{Ruta: "/b/tres.jpg", RutaPadre: "/b", Tipo: modelo.TipoImagen},
			{Ruta: "/b/cuatro.jpg", RutaPadre: "/b", Tipo: modelo.TipoImagen},
			{Ruta: "/b/cinco.jpg", RutaPadre: "/b", Tipo: modelo.TipoImagen},
		},
	}

	filas := app.filasNavegacionExplorador(2)
	esperadas := []filaNavegacionExplorador{
		{Visual: 1, Indices: []int{0, 1}},
		{Visual: 3, Indices: []int{2, 3}},
		{Visual: 4, Indices: []int{4}},
	}
	if len(filas) != len(esperadas) {
		t.Fatalf("cantidad de filas inesperada: %+v", filas)
	}
	for indice := range esperadas {
		if filas[indice].Visual != esperadas[indice].Visual {
			t.Fatalf("visual inesperado en fila %d: %+v", indice, filas[indice])
		}
		if len(filas[indice].Indices) != len(esperadas[indice].Indices) {
			t.Fatalf("fila %d inesperada: %+v", indice, filas[indice])
		}
		for columna := range esperadas[indice].Indices {
			if filas[indice].Indices[columna] != esperadas[indice].Indices[columna] {
				t.Fatalf("fila %d columna %d inesperada: %+v", indice, columna, filas)
			}
		}
	}
}

func TestAjustarPosicionFilaExploradorVisibleBordeSuperior(t *testing.T) {
	t.Parallel()

	inicial := layout.Position{
		BeforeEnd: true,
		First:     5,
		Offset:    18,
		Count:     6,
	}

	ajustada, ok := ajustarPosicionFilaExploradorVisible(inicial, 5)
	if !ok {
		t.Fatal("se esperaba ajuste en el borde superior parcial")
	}
	if ajustada.First != 5 || ajustada.Offset != 0 || ajustada.OffsetLast != 0 || !ajustada.BeforeEnd {
		t.Fatalf("posicion superior ajustada inesperada: %+v", ajustada)
	}
}

func TestAjustarPosicionFilaExploradorVisibleBordeInferior(t *testing.T) {
	t.Parallel()

	inicial := layout.Position{
		BeforeEnd:  true,
		First:      10,
		Offset:     0,
		OffsetLast: -24,
		Count:      4,
	}

	ajustada, ok := ajustarPosicionFilaExploradorVisible(inicial, 13)
	if !ok {
		t.Fatal("se esperaba ajuste en el borde inferior parcial")
	}
	if ajustada.First != 11 || ajustada.Offset != 0 || ajustada.OffsetLast != 0 || !ajustada.BeforeEnd {
		t.Fatalf("posicion inferior ajustada inesperada: %+v", ajustada)
	}
}

func TestAjustarPosicionFilaExploradorVisibleSinCambios(t *testing.T) {
	t.Parallel()

	inicial := layout.Position{
		BeforeEnd:  true,
		First:      3,
		Offset:     0,
		OffsetLast: 12,
		Count:      5,
	}

	ajustada, ok := ajustarPosicionFilaExploradorVisible(inicial, 5)
	if ok {
		t.Fatalf("no se esperaba ajuste: %+v", ajustada)
	}
	if ajustada != inicial {
		t.Fatalf("la posicion no debia cambiar: inicial=%+v ajustada=%+v", inicial, ajustada)
	}
}
