/**
 * Dominio de la API en Node: estadísticas sobre las matrices que envía la API en Go.
 * No depende de Express ni de AWS. La lógica es la misma de la sección 8 del notebook.
 */

/** Margen para considerar que un valor de punto flotante es cero. */
export const TOLERANCIA = 1e-10;
/** Q y R de 200x200 (el máximo de la API en Go) suman 80.000 valores. */
export const MAX_VALORES = 100_000;

export class ErrorEntrada extends Error {}

/**
 * Revisa que `matrices` sea un objeto { nombre: matriz } con matrices rectangulares de números finitos.
 * @param {unknown} matrices
 * @returns {Record<string, number[][]>}
 */
export function validarMatrices(matrices) {
  if (matrices === null || typeof matrices !== 'object' || Array.isArray(matrices)) {
    throw new ErrorEntrada("El campo 'matrices' debe ser un objeto { nombre: matriz }");
  }
  const nombres = Object.keys(matrices);
  if (nombres.length === 0) {
    throw new ErrorEntrada('Debe enviar al menos una matriz');
  }
  let total = 0;
  for (const nombre of nombres) {
    const m = matrices[nombre];
    if (!Array.isArray(m) || m.length === 0) {
      throw new ErrorEntrada(`La matriz '${nombre}' debe ser un arreglo no vacío de filas`);
    }
    const columnas = Array.isArray(m[0]) ? m[0].length : 0;
    m.forEach((fila, i) => {
      if (!Array.isArray(fila) || fila.length === 0) {
        throw new ErrorEntrada(`La fila ${i} de '${nombre}' debe ser un arreglo no vacío`);
      }
      if (fila.length !== columnas) {
        throw new ErrorEntrada(`La fila ${i} de '${nombre}' tiene ${fila.length} elementos y la fila 0 tiene ${columnas}: no es rectangular`);
      }
      fila.forEach((valor, j) => {
        if (typeof valor !== 'number' || !Number.isFinite(valor)) {
          throw new ErrorEntrada(`El elemento [${i}][${j}] de '${nombre}' no es un número finito`);
        }
      });
      total += fila.length;
    });
  }
  if (total > MAX_VALORES) {
    throw new ErrorEntrada(`Se recibieron ${total} valores; el máximo es ${MAX_VALORES}`);
  }
  return matrices;
}

/**
 * Recorre todos los valores UNA sola vez y calcula máximo, mínimo, suma, promedio y qué matrices son diagonales.
 * - La suma usa el método de Neumaier: guarda en `compensacion` lo que se pierde por redondeo.
 * - "Diagonal" significa que todo lo que está fuera de la diagonal principal es cero (con tolerancia);
 *   la definición vale también para matrices no cuadradas.
 * @param {Record<string, number[][]>} matrices
 * @param {number} [tol]
 */
export function calcularEstadisticas(matrices, tol = TOLERANCIA) {
  let maximo = -Infinity;
  let minimo = Infinity;
  let suma = 0;
  let compensacion = 0;
  let cantidad = 0;
  const esDiagonal = {};

  for (const [nombre, m] of Object.entries(matrices)) {
    let diagonal = true;
    for (let i = 0; i < m.length; i++) {
      const fila = m[i];
      for (let j = 0; j < fila.length; j++) {
        const valor = fila[j];
        if (valor > maximo) maximo = valor;
        if (valor < minimo) minimo = valor;
        const t = suma + valor; // suma de Neumaier
        compensacion += Math.abs(suma) >= Math.abs(valor) ? (suma - t) + valor : (valor - t) + suma;
        suma = t;
        cantidad++;
        if (i !== j && Math.abs(valor) > tol) diagonal = false;
      }
    }
    esDiagonal[nombre] = diagonal;
  }
  suma += compensacion;

  return {
    maximo,
    minimo,
    suma,
    promedio: suma / cantidad,
    cantidadValores: cantidad,
    esDiagonal,
    algunaDiagonal: Object.values(esDiagonal).some(Boolean),
  };
}
