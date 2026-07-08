//go:build !darwin

package plataforma

// EsBundle fuera de macOS devuelve siempre falso.
func EsBundle(_ string) bool {
	return false
}
