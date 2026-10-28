package matriz

import (
	"math"
	"sort"
)

// MaxIteraciones limita el algoritmo QR de autovalores.
const MaxIteraciones = 10_000

// Autovalores contiene los autovalores (de mayor a menor) y sus autovectores (uno por columna).
type Autovalores struct {
	Valores     []float64 `json:"autovalores"`
	Vectores    Matriz    `json:"autovectores"`
	Iteraciones int       `json:"iteraciones"`
}

// CalcularAutovalores aplica el algoritmo QR: A₀ = A, Aₖ = QₖRₖ, Aₖ₊₁ = RₖQₖ.
// Aₖ conserva los autovalores y, en una matriz simétrica, se vuelve diagonal; el producto de las Qₖ da los autovectores.
// Solo acepta matrices simétricas: las demás pueden tener autovalores complejos (por ejemplo, una rotación de 90°).
func CalcularAutovalores(a Matriz) (Autovalores, error) {
	n, m := a.Dimensiones()
	if n != m {
		return Autovalores{}, errorEntrada("Los autovalores solo existen para matrices cuadradas; la matriz es %d x %d", n, m)
	}
	escala := math.Max(1, maximoAbsoluto(a))
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if math.Abs(a[i][j]-a[j][i]) > Tolerancia*escala {
				return Autovalores{}, errorEntrada("Solo se aceptan matrices simétricas: las demás pueden tener autovalores complejos")
			}
		}
	}
	ak, v := a.copiar(), identidad(n)
	for iteracion := 1; iteracion <= MaxIteraciones; iteracion++ {
		q, r, _ := householder(ak)
		ak = Multiplicar(r, q)
		v = Multiplicar(v, q)
		if ErrorTriangular(ak) <= Tolerancia*escala {
			return ordenar(ak, v, iteracion), nil
		}
	}
	return Autovalores{}, errorEntrada("El algoritmo no convergió en %d iteraciones (por ejemplo, hay autovalores λ y -λ)", MaxIteraciones)
}

// ordenar deja los autovalores de mayor a menor, con su autovector en la misma posición.
func ordenar(ak, v Matriz, iteraciones int) Autovalores {
	n := len(ak)
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(x, y int) bool { return ak[indices[x]][indices[x]] > ak[indices[y]][indices[y]] })
	resultado := Autovalores{Valores: make([]float64, n), Vectores: nueva(n, n), Iteraciones: iteraciones}
	for nuevo, original := range indices {
		resultado.Valores[nuevo] = ak[original][original]
		for fila := 0; fila < n; fila++ {
			resultado.Vectores[fila][nuevo] = v[fila][original]
		}
	}
	return resultado
}
