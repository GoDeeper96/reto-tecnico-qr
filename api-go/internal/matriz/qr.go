package matriz

import (
	"math"
	"sort"
	"strings"
)

// Factorizador es una forma de calcular la QR (patrón Strategy).
// Devuelve Q y R sin normalizar los signos; de eso se encarga FactorizarQR.
type Factorizador interface {
	Factorizar(a Matriz) (q, r Matriz, err error)
}

// FactorizadorFunc adapta una función a la interfaz Factorizador.
type FactorizadorFunc func(a Matriz) (Matriz, Matriz, error)

func (f FactorizadorFunc) Factorizar(a Matriz) (Matriz, Matriz, error) { return f(a) }

// MetodoPorDefecto es el único método, junto con Givens, que cumple las validaciones con cualquier matriz,
// y el más rápido de los cuatro (ver la sección 5 y 6 del notebook).
const MetodoPorDefecto = "householder"

// Metodos registra las estrategias disponibles por nombre.
var Metodos = map[string]Factorizador{
	"gram-schmidt-clasico":    FactorizadorFunc(func(a Matriz) (Matriz, Matriz, error) { return gramSchmidt(a, false) }),
	"gram-schmidt-modificado": FactorizadorFunc(func(a Matriz) (Matriz, Matriz, error) { return gramSchmidt(a, true) }),
	"householder":             FactorizadorFunc(householder),
	"givens":                  FactorizadorFunc(givens),
}

// NombresMetodos devuelve los métodos disponibles en orden alfabético.
func NombresMetodos() []string {
	nombres := make([]string, 0, len(Metodos))
	for nombre := range Metodos {
		nombres = append(nombres, nombre)
	}
	sort.Strings(nombres)
	return nombres
}

// FactorizarQR calcula A = Q·R con el método indicado y deja la diagonal de R positiva,
// para que todos los métodos devuelvan la misma respuesta.
func FactorizarQR(a Matriz, metodo string) (Matriz, Matriz, error) {
	if metodo == "" {
		metodo = MetodoPorDefecto
	}
	factorizador, ok := Metodos[metodo]
	if !ok {
		return nil, nil, errorEntrada("Método desconocido: %q. Opciones: %s", metodo, strings.Join(NombresMetodos(), ", "))
	}
	q, r, err := factorizador.Factorizar(a)
	if err != nil {
		return nil, nil, err
	}
	normalizarSignos(q, r)
	return q, r, nil
}

// gramSchmidt calcula la QR reducida (Q m×n, R n×n).
// El clásico proyecta la columna ORIGINAL; el modificado, la columna YA CORREGIDA, y acumula menos error.
func gramSchmidt(a Matriz, modificado bool) (Matriz, Matriz, error) {
	m, n := a.Dimensiones()
	if m < n {
		return nil, nil, errorEntrada("Gram-Schmidt necesita al menos tantas filas como columnas; use householder o givens")
	}
	q, r := nueva(m, n), nueva(n, n)
	v := make([]float64, m)
	for j := 0; j < n; j++ {
		normaOriginal := 0.0
		for k := 0; k < m; k++ {
			v[k] = a[k][j]
			normaOriginal += a[k][j] * a[k][j]
		}
		for i := 0; i < j; i++ {
			proyeccion := 0.0
			for k := 0; k < m; k++ {
				if modificado {
					proyeccion += q[k][i] * v[k]
				} else {
					proyeccion += q[k][i] * a[k][j]
				}
			}
			r[i][j] = proyeccion
			for k := 0; k < m; k++ {
				v[k] -= proyeccion * q[k][i]
			}
		}
		norma := 0.0
		for k := 0; k < m; k++ {
			norma += v[k] * v[k]
		}
		norma = math.Sqrt(norma)
		if norma <= tolDependencia*math.Max(1, math.Sqrt(normaOriginal)) {
			return nil, nil, errorEntrada("La columna %d depende linealmente de las anteriores; use householder o givens", j)
		}
		r[j][j] = norma
		for k := 0; k < m; k++ {
			q[k][j] = v[k] / norma
		}
	}
	return q, r, nil
}

// householder calcula la QR completa (Q m×m, R m×n) con reflejos H = I - 2vvᵀ.
// Funciona con cualquier forma y rango.
func householder(a Matriz) (Matriz, Matriz, error) {
	m, n := a.Dimensiones()
	r, q := a.copiar(), identidad(m)
	v := make([]float64, m)
	for k := 0; k < min(m-1, n); k++ {
		normaX := 0.0
		for i := k; i < m; i++ {
			normaX += r[i][k] * r[i][k]
		}
		normaX = math.Sqrt(normaX)
		if normaX == 0 {
			continue // la columna ya tiene ceros debajo de la diagonal
		}
		for i := k; i < m; i++ {
			v[i] = r[i][k]
		}
		v[k] += math.Copysign(normaX, r[k][k]) // mismo signo que x[0]: evita la cancelación
		normaV := 0.0
		for i := k; i < m; i++ {
			normaV += v[i] * v[i]
		}
		normaV = math.Sqrt(normaV)
		for i := k; i < m; i++ {
			v[i] /= normaV
		}
		// R[k:, k:] -= 2·v·(vᵀ·R[k:, k:]); las columnas anteriores ya son cero en esas filas.
		for j := k; j < n; j++ {
			s := 0.0
			for i := k; i < m; i++ {
				s += v[i] * r[i][j]
			}
			s *= 2
			for i := k; i < m; i++ {
				r[i][j] -= s * v[i]
			}
		}
		// Q[:, k:] -= 2·(Q[:, k:]·v)·vᵀ
		for i := 0; i < m; i++ {
			s := 0.0
			for l := k; l < m; l++ {
				s += q[i][l] * v[l]
			}
			s *= 2
			for l := k; l < m; l++ {
				q[i][l] -= s * v[l]
			}
		}
		for i := k + 1; i < m; i++ {
			r[i][k] = 0 // en teoría ya son ceros; se fija para quitar el residuo de redondeo
		}
	}
	return q, r, nil
}

// givens calcula la QR completa con rotaciones: cada una gira las filas (i-1, i)
// un ángulo θ que deja en cero el elemento R[i][j]. Q es el producto de todas las rotaciones.
func givens(a Matriz) (Matriz, Matriz, error) {
	m, n := a.Dimensiones()
	r, q := a.copiar(), identidad(m)
	for j := 0; j < n; j++ {
		for i := m - 1; i > j; i-- { // de abajo hacia arriba, hasta la diagonal
			x, y := r[i-1][j], r[i][j]
			if y == 0 {
				continue
			}
			h := math.Hypot(x, y) // sqrt(x² + y²) sin desbordar
			c, s := x/h, y/h      // cos θ y sen θ
			for l := j; l < n; l++ {
				arriba, abajo := r[i-1][l], r[i][l]
				r[i-1][l] = c*arriba + s*abajo
				r[i][l] = -s*arriba + c*abajo
			}
			for l := 0; l < m; l++ {
				izquierda, derecha := q[l][i-1], q[l][i]
				q[l][i-1] = c*izquierda + s*derecha
				q[l][i] = -s*izquierda + c*derecha
			}
			r[i][j] = 0 // en teoría ya es cero; se fija para quitar el residuo
		}
	}
	return q, r, nil
}

// normalizarSignos cambia signos para que la diagonal de R sea positiva. Q·R no cambia.
func normalizarSignos(q, r Matriz) {
	filasR, columnasR := r.Dimensiones()
	for i := 0; i < min(filasR, columnasR); i++ {
		if r[i][i] < 0 {
			for j := range r[i] {
				r[i][j] = -r[i][j]
			}
			for k := range q {
				q[k][i] = -q[k][i]
			}
		}
	}
}
