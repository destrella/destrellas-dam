package plataforma

import "os/exec"

// ComandoDisponible informa si una utilidad externa esta instalada.
func ComandoDisponible(nombre string) bool {
	_, err := exec.LookPath(nombre)
	return err == nil
}
