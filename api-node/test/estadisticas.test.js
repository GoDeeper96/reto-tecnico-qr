import { test } from 'node:test';
import assert from 'node:assert/strict';
import { calcularEstadisticas, validarMatrices, ErrorEntrada } from '../src/domain/estadisticas.js';

test('calcula máximo, mínimo, suma y promedio de todas las matrices', () => {
  const r = calcularEstadisticas({ q: [[1, -2], [3, 4]], r: [[5, 0], [0, 6]] });
  assert.equal(r.maximo, 6);
  assert.equal(r.minimo, -2);
  assert.equal(r.suma, 17);
  assert.equal(r.promedio, 17 / 8);
  assert.equal(r.cantidadValores, 8);
});

test('detecta matrices diagonales, también rectangulares', () => {
  const r = calcularEstadisticas({ q: [[1, 2], [3, 4]], r: [[1.41, 0], [0, 1.41], [0, 0]] });
  assert.deepEqual(r.esDiagonal, { q: false, r: true });
  assert.equal(r.algunaDiagonal, true);
});

test('la tolerancia ignora residuos de redondeo, pero no valores reales', () => {
  assert.equal(calcularEstadisticas({ m: [[2, 1e-17], [0, 3]] }).algunaDiagonal, true);
  assert.equal(calcularEstadisticas({ m: [[2, 1e-17], [0, 3]] }, 0).algunaDiagonal, false);
  assert.equal(calcularEstadisticas({ m: [[2, 1e-3], [0, 3]] }).algunaDiagonal, false);
});

test('la suma de Neumaier no pierde los valores pequeños', () => {
  // 1e16 + 1 + (-1e16) da 0 con una suma normal; el resultado correcto es 1.
  assert.equal(calcularEstadisticas({ m: [[1e16, 1, -1e16]] }).suma, 1);
});

test('rechaza entradas inválidas', () => {
  for (const entrada of [undefined, null, [], {}, { q: [] }, { q: [[1, 2], [3]] }, { q: [[1, 'a']] }, { q: [[1, null]] }, { q: [[]] }]) {
    assert.throws(() => validarMatrices(entrada), ErrorEntrada, JSON.stringify(entrada));
  }
  assert.doesNotThrow(() => validarMatrices({ q: [[1, 2], [3, 4]] }));
});
