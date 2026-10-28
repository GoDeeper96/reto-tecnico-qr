package matriz

import (
	"errors"
	"math"
	"math/rand"
	"testing"
)

func hilbert(n int) Matriz {
	h := nueva(n, n)
	for i := range h {
		for j := range h[i] {
			h[i][j] = 1 / float64(i+j+1)
		}
	}
	return h
}

func aleatoria(filas, columnas int, semilla int64) Matriz {
	rng := rand.New(rand.NewSource(semilla))
	a := nueva(filas, columnas)
	for i := range a {
		for j := range a[i] {
			a[i][j] = rng.NormFloat64()
		}
	}
	return a
}

func TestValidar(t *testing.T) {
	casos := []struct {
		nombre string
		a      Matriz
		valida bool
	}{
		{"correcta 2x3", Matriz{{1, 2, 3}, {4, 5, 6}}, true},
		{"vacía", Matriz{}, false},
		{"fila vacía", Matriz{{}}, false},
		{"no rectangular", Matriz{{1, 2}, {3}}, false},
		{"infinito", Matriz{{1, math.Inf(1)}}, false},
		{"NaN", Matriz{{math.NaN()}}, false},
		{"demasiado grande", nueva(1, MaxDimension+1), false},
	}
	for _, c := range casos {
		err := Validar(c.a)
		var entrada *ErrorEntrada
		if c.valida && err != nil {
			t.Errorf("%s: se esperaba válida, error %v", c.nombre, err)
		}
		if !c.valida && !errors.As(err, &entrada) {
			t.Errorf("%s: se esperaba ErrorEntrada, se obtuvo %v", c.nombre, err)
		}
	}
}

// Ejemplo calculado en el notebook: A = [[1,2],[3,4],[5,6]] -> R = [[5.9161, 7.4374], [0, 0.8281], [0, 0]].
func TestFactorizarQRCoincideConElNotebook(t *testing.T) {
	a := Matriz{{1, 2}, {3, 4}, {5, 6}}
	esperada := Matriz{{5.916079783099616, 7.437357441610946}, {0, 0.828078671210825}}
	for _, metodo := range NombresMetodos() {
		_, r, err := FactorizarQR(a, metodo)
		if err != nil {
			t.Fatalf("%s: %v", metodo, err)
		}
		for i := range esperada {
			for j := range esperada[i] {
				if math.Abs(r[i][j]-esperada[i][j]) > 1e-12 {
					t.Errorf("%s: R[%d][%d] = %v, se esperaba %v", metodo, i, j, r[i][j], esperada[i][j])
				}
			}
		}
	}
}

// Misma tabla de estabilidad que la sección 5 del notebook.
func TestEstabilidadPorMetodo(t *testing.T) {
	const (
		cumple  = "cumple"
		falla   = "falla"
		rechaza = "rechaza"
	)
	eps := 1e-8
	casos := []struct {
		nombre    string
		a         Matriz
		esperados map[string]string
	}{
		{"aleatoria 50x30", aleatoria(50, 30, 0), map[string]string{
			"gram-schmidt-clasico": cumple, "gram-schmidt-modificado": cumple, "householder": cumple, "givens": cumple}},
		{"Hilbert 10x10", hilbert(10), map[string]string{
			"gram-schmidt-clasico": falla, "gram-schmidt-modificado": falla, "householder": cumple, "givens": cumple}},
		{"Läuchli", Matriz{{1, 1, 1}, {eps, 0, 0}, {0, eps, 0}, {0, 0, eps}}, map[string]string{
			"gram-schmidt-clasico": falla, "gram-schmidt-modificado": falla, "householder": cumple, "givens": cumple}},
		{"rango deficiente", Matriz{{1, 2}, {2, 4}, {3, 6}}, map[string]string{
			"gram-schmidt-clasico": rechaza, "gram-schmidt-modificado": rechaza, "householder": cumple, "givens": cumple}},
		{"ancha 2x3", Matriz{{1, 2, 3}, {4, 5, 6}}, map[string]string{
			"gram-schmidt-clasico": rechaza, "gram-schmidt-modificado": rechaza, "householder": cumple, "givens": cumple}},
	}
	for _, c := range casos {
		for metodo, esperado := range c.esperados {
			q, r, err := FactorizarQR(c.a, metodo)
			obtenido := rechaza
			if err == nil {
				obtenido = falla
				if ValidarQR(c.a, q, r).TodasCumplen() {
					obtenido = cumple
				}
			}
			if obtenido != esperado {
				t.Errorf("%s con %s: se esperaba %q, se obtuvo %q", c.nombre, metodo, esperado, obtenido)
			}
		}
	}
}

func TestFormaDeLaQR(t *testing.T) {
	a := aleatoria(6, 4, 1)
	formas := map[string][4]int{
		"householder":          {6, 6, 6, 4}, // completa
		"givens":               {6, 6, 6, 4},
		"gram-schmidt-clasico": {6, 4, 4, 4}, // reducida
	}
	for metodo, f := range formas {
		q, r, err := FactorizarQR(a, metodo)
		if err != nil {
			t.Fatal(err)
		}
		mq, nq := q.Dimensiones()
		mr, nr := r.Dimensiones()
		if [4]int{mq, nq, mr, nr} != f {
			t.Errorf("%s: Q %dx%d, R %dx%d; se esperaba %v", metodo, mq, nq, mr, nr, f)
		}
		for i := 0; i < min(mr, nr); i++ {
			if r[i][i] < 0 {
				t.Errorf("%s: R[%d][%d] es negativo; la diagonal debe ser positiva", metodo, i, i)
			}
		}
	}
}

func TestValidarQRDetectaFalsasFactorizaciones(t *testing.T) {
	a := Matriz{{1, 2}, {3, 4}, {5, 6}}
	// Q = A y R = identidad reconstruyen A, pero Q no es ortogonal.
	v := ValidarQR(a, a, identidad(2))
	if v.QOrtogonal.Cumple || !v.Reconstruccion.Cumple || !v.RTriangular.Cumple {
		t.Errorf("validaciones inesperadas: %+v", v)
	}
	if v.TodasCumplen() {
		t.Error("una factorización con Q no ortogonal no debe ser válida")
	}
}

func TestMetodoDesconocido(t *testing.T) {
	_, _, err := FactorizarQR(Matriz{{1}}, "lu")
	var entrada *ErrorEntrada
	if !errors.As(err, &entrada) {
		t.Fatalf("se esperaba ErrorEntrada, se obtuvo %v", err)
	}
}

func TestAutovalores(t *testing.T) {
	resultado, err := CalcularAutovalores(Matriz{{2, 1}, {1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	esperados := []float64{3, 1}
	for i, v := range esperados {
		if math.Abs(resultado.Valores[i]-v) > 1e-9 {
			t.Errorf("autovalor %d = %v, se esperaba %v", i, resultado.Valores[i], v)
		}
	}
	// A·v = λ·v para cada autovector, y los autovectores son ortogonales.
	a := Matriz{{4, 1, 0.5}, {1, 3, 0.2}, {0.5, 0.2, 1}}
	resultado, err = CalcularAutovalores(a)
	if err != nil {
		t.Fatal(err)
	}
	av := Multiplicar(a, resultado.Vectores)
	for i := range av {
		for j := range av[i] {
			if math.Abs(av[i][j]-resultado.Vectores[i][j]*resultado.Valores[j]) > 1e-8 {
				t.Fatalf("A·v != λ·v en [%d][%d]", i, j)
			}
		}
	}
	if e := ErrorOrtogonalidad(resultado.Vectores); e > Tolerancia {
		t.Errorf("autovectores no ortogonales: %v", e)
	}
}

func TestAutovaloresRechazaNoSimetricas(t *testing.T) {
	rotacion90 := Matriz{{0, -1}, {1, 0}}
	if _, err := CalcularAutovalores(rotacion90); err == nil {
		t.Error("la rotación de 90° no es simétrica y debe rechazarse")
	}
	if _, err := CalcularAutovalores(Matriz{{1, 2, 3}, {4, 5, 6}}); err == nil {
		t.Error("una matriz no cuadrada debe rechazarse")
	}
}

func BenchmarkHouseholder100(b *testing.B) {
	a := aleatoria(100, 100, 2)
	for i := 0; i < b.N; i++ {
		_, _, _ = FactorizarQR(a, "householder")
	}
}
