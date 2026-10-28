package matriz

import "math"

// Regla es el resultado de una validación: si se cumple y el error medido.
type Regla struct {
	Cumple bool    `json:"cumple"`
	Error  float64 `json:"error"`
}

// Validaciones son las tres reglas que debe cumplir toda factorización QR.
type Validaciones struct {
	QOrtogonal     Regla `json:"qOrtogonal"`     // QᵀQ = I
	RTriangular    Regla `json:"rTriangular"`    // todo lo que está debajo de la diagonal de R es 0
	Reconstruccion Regla `json:"reconstruccion"` // Q·R = A
}

// TodasCumplen indica si la factorización es válida.
func (v Validaciones) TodasCumplen() bool {
	return v.QOrtogonal.Cumple && v.RTriangular.Cumple && v.Reconstruccion.Cumple
}

func regla(err float64) Regla { return Regla{Cumple: err <= Tolerancia, Error: err} }

// ValidarQR mide las tres reglas. Revisar solo Q·R = A no basta: Q = A y R = I también reconstruyen A.
func ValidarQR(a, q, r Matriz) Validaciones {
	return Validaciones{
		QOrtogonal:     regla(ErrorOrtogonalidad(q)),
		RTriangular:    regla(ErrorTriangular(r)),
		Reconstruccion: regla(ErrorReconstruccion(a, q, r)),
	}
}

// ErrorOrtogonalidad es la mayor diferencia entre QᵀQ y la identidad.
func ErrorOrtogonalidad(q Matriz) float64 {
	filas, columnas := q.Dimensiones()
	peor := 0.0
	for i := 0; i < columnas; i++ {
		for j := i; j < columnas; j++ { // QᵀQ es simétrica: basta la mitad
			producto := 0.0
			for k := 0; k < filas; k++ {
				producto += q[k][i] * q[k][j]
			}
			esperado := 0.0
			if i == j {
				esperado = 1
			}
			peor = math.Max(peor, math.Abs(producto-esperado))
		}
	}
	return peor
}

// ErrorTriangular es el mayor valor absoluto debajo de la diagonal.
func ErrorTriangular(r Matriz) float64 {
	peor := 0.0
	for i, fila := range r {
		for j := 0; j < min(i, len(fila)); j++ {
			peor = math.Max(peor, math.Abs(fila[j]))
		}
	}
	return peor
}

// ErrorReconstruccion es la mayor diferencia entre Q·R y A, relativa al mayor valor de A.
func ErrorReconstruccion(a, q, r Matriz) float64 {
	producto := Multiplicar(q, r)
	escala := math.Max(1, maximoAbsoluto(a))
	peor := 0.0
	for i, fila := range a {
		for j, valor := range fila {
			peor = math.Max(peor, math.Abs(producto[i][j]-valor))
		}
	}
	return peor / escala
}
