package modelo

// UbicacionGuardada resume un nombre de ubicación conocido y los datos
// efectivos que puede aportar al formulario de metadatos.
type UbicacionGuardada struct {
	Nombre         string
	RelacionadaCon string
	CantidadUsos   int
	Coordenadas    *Coordenadas
	Ciudad         string
	Estado         string
	Pais           string
}

// UsoUbicacionGuardada representa una combinación concreta de coordenadas y
// dirección con la que un nombre de ubicación se ha usado en archivos.
type UsoUbicacionGuardada struct {
	Nombre       string
	CantidadUsos int
	Coordenadas  *Coordenadas
	Ciudad       string
	Estado       string
	Pais         string
}

// TieneDireccion informa si existe alguna parte nominal de la dirección.
func (u UbicacionGuardada) TieneDireccion() bool {
	return u.Ciudad != "" || u.Estado != "" || u.Pais != ""
}

// TieneDireccion informa si existe alguna parte nominal de la dirección.
func (u UsoUbicacionGuardada) TieneDireccion() bool {
	return u.Ciudad != "" || u.Estado != "" || u.Pais != ""
}
