package modelo

import (
	"encoding/json"
	"testing"
)

func TestFiltrosPorDefectoUsanOrdenAlfabetico(t *testing.T) {
	t.Parallel()

	filtros := FiltrosPorDefecto()
	if filtros.CriterioOrdenNormalizado() != CriterioOrdenNombre {
		t.Fatalf("criterio por defecto inesperado: %q", filtros.CriterioOrdenNormalizado())
	}
	if filtros.OrdenDescendente {
		t.Fatal("no se esperaba orden descendente por defecto")
	}
}

func TestFiltrosUnmarshalJSONAplicaCriterioOrdenPorDefecto(t *testing.T) {
	t.Parallel()

	var filtros FiltrosListado
	if err := json.Unmarshal([]byte(`{"mostrar_ocultos":true,"orden_descendente":true}`), &filtros); err != nil {
		t.Fatalf("no se pudo deserializar el filtro: %v", err)
	}

	if !filtros.MostrarOcultos {
		t.Fatal("se esperaba conservar mostrar_ocultos=true")
	}
	if !filtros.OrdenDescendente {
		t.Fatal("se esperaba conservar orden_descendente=true")
	}
	if filtros.CriterioOrdenNormalizado() != CriterioOrdenNombre {
		t.Fatalf("criterio heredado inesperado: %q", filtros.CriterioOrdenNormalizado())
	}
}

func TestCriterioOrdenNormalizadoDescartaValoresDesconocidos(t *testing.T) {
	t.Parallel()

	filtros := FiltrosListado{CriterioOrden: CriterioOrdenListado("desconocido")}
	if filtros.CriterioOrdenNormalizado() != CriterioOrdenNombre {
		t.Fatalf("se esperaba volver al criterio por nombre, se obtuvo %q", filtros.CriterioOrdenNormalizado())
	}
}
