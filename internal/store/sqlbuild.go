package store

import "strings"

// inClause retorna "(?,?,?)" com n placeholders. Função pura — sem concat
// de input externo, segura por construção. Usada onde o número de
// placeholders depende do tamanho de um set/slice conhecido em runtime.
func inClause(n int) string {
	if n == 0 {
		return "()"
	}
	return "(" + strings.Repeat("?,", n-1) + "?)"
}
