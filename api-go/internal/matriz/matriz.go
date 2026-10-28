// Package matriz es el dominio de la API: validación de la entrada, factorización QR,
// validación del resultado y autovalores. No depende de HTTP ni de AWS.
//
// La lógica es la misma del notebook notebooks/QR_y_autovalores.ipynb, que sirve de especificación.
package matriz

import (
	"fmt"
	"math"
)

// Matriz se guarda por filas: m[i][j] es la fila i, columna j.
type Matriz [][]float64

const (
	// Tolerancia es el margen para considerar que un valor de punto flotante es cero.
	Tolerancia = 1e-10
	// MaxDimension limita filas y columnas. Q y R de 200x200 ya suman unos 2 MB de JSON,
	// cerca del límite de 6 MB de una respuesta de Lambda.
	MaxDimension = 200
	// tolDependencia: si lo que queda de una columna en Gram-Schmidt es menor a esto (relativo), es dependiente.
	tolDependencia = 1e-14
)

// ErrorEntrada indica que la matriz o los parámetros recibidos no son válidos (la API responde 400).
type ErrorEntrada struct{ Mensaje string }

func (e *ErrorEntrada) Error() string { return e.Mensaje }

func errorEntrada(formato string, args ...any) error {
	return &ErrorEntrada{Mensaje: fmt.Sprintf(formato, args...)}
}

// Validar revisa que la matriz no esté vacía, sea rectangular, tenga solo números finitos y no supere MaxDimension.
func Validar(a Matriz) error {
	if len(a) == 0 {
		return errorEntrada("La matriz debe ser un arreglo no vacío de filas")
	}
	columnas := len(a[0])
	for i, fila := range a {
		if len(fila) == 0 {
			return errorEntrada("La fila %d debe ser un arreglo no vacío", i)
		}
		if len(fila) != columnas {
			return errorEntrada("La fila %d tiene %d elementos y la fila 0 tiene %d: la matriz no es rectangular", i, len(fila), columnas)
		}
		for j, valor := range fila {
			if math.IsNaN(valor) || math.IsInf(valor, 0) {
				return errorEntrada("El elemento [%d][%d] no es un número finito", i, j)
			}
		}
	}
	if len(a) > MaxDimension || columnas > MaxDimension {
		return errorEntrada("La matriz supera el tamaño máximo de %d x %d", MaxDimension, MaxDimension)
	}
	return nil
}

// Dimensiones devuelve filas y columnas.
func (a Matriz) Dimensiones() (int, int) {
	if len(a) == 0 {
		return 0, 0
	}
	return len(a), len(a[0])
}

func nueva(filas, columnas int) Matriz {
	m := make(Matriz, filas)
	for i := range m {
		m[i] = make([]float64, columnas)
	}
	return m
}

func identidad(n int) Matriz {
	m := nueva(n, n)
	for i := range m {
		m[i][i] = 1
	}
	return m
}

func (a Matriz) copiar() Matriz {
	m := make(Matriz, len(a))
	for i, fila := range a {
		m[i] = append([]float64(nil), fila...)
	}
	return m
}

// Multiplicar devuelve a·b.
func Multiplicar(a, b Matriz) Matriz {
	m, n := a.Dimensiones()
	_, p := b.Dimensiones()
	c := nueva(m, p)
	for i := 0; i < m; i++ {
		for k := 0; k < n; k++ {
			if a[i][k] == 0 {
				continue
			}
			for j := 0; j < p; j++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return c
}

func maximoAbsoluto(a Matriz) float64 {
	peor := 0.0
	for _, fila := range a {
		for _, v := range fila {
			peor = math.Max(peor, math.Abs(v))
		}
	}
	return peor
}
