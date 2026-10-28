// Matrices de ejemplo: las mismas del notebook y de las pruebas.
const hilbert = (n) => Array.from({ length: n }, (_, i) => Array.from({ length: n }, (_, j) => +(1 / (i + j + 1)).toPrecision(15)));

export const EJEMPLOS = [
  { nombre: '3×2 básica', matriz: [[1, 2], [3, 4], [5, 6]] },
  { nombre: 'Ancha 2×3', matriz: [[1, 2, 3], [4, 5, 6]] },
  { nombre: 'Columnas perpendiculares', matriz: [[1, 1], [1, -1], [0, 0]] },
  { nombre: 'Rango deficiente', matriz: [[1, 2], [2, 4], [3, 6]] },
  { nombre: 'Hilbert 10×10', matriz: hilbert(10) },
  { nombre: 'Simétrica (autovalores)', matriz: [[4, 1, 0.5], [1, 3, 0.2], [0.5, 0.2, 1]] },
];

export const METODOS = [
  { valor: 'householder', nombre: 'Householder (recomendado)' },
  { valor: 'givens', nombre: 'Givens (rotaciones)' },
  { valor: 'gram-schmidt-modificado', nombre: 'Gram-Schmidt modificado' },
  { valor: 'gram-schmidt-clasico', nombre: 'Gram-Schmidt clásico' },
];

export const aTexto = (m) => '[\n' + m.map((fila) => '  [' + fila.join(', ') + ']').join(',\n') + '\n]';
