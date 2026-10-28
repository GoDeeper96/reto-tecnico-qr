import { test } from 'node:test';
import assert from 'node:assert/strict';
import jwt from 'jsonwebtoken';
import request from 'supertest';
import { crearApp } from '../src/app.js';
import { EMISOR, AUDIENCIA } from '../src/auth.js';

const SECRETO = 'secreto-de-prueba';
const app = crearApp({ obtenerSecretos: async () => ({ secretoJwt: SECRETO }), version: 'test' });
const token = (opciones = {}) =>
  'Bearer ' + jwt.sign({ sub: 'test' }, SECRETO, { issuer: EMISOR, audience: AUDIENCIA, expiresIn: '5m', ...opciones });

test('GET /health responde sin token', async () => {
  const res = await request(app).get('/health');
  assert.equal(res.status, 200);
  assert.equal(res.body.servicio, 'api-node');
});

test('POST /api/v1/estadisticas con token válido', async () => {
  const res = await request(app)
    .post('/api/v1/estadisticas')
    .set('Authorization', token())
    .set('X-Request-ID', 'req-123')
    .send({ matrices: { q: [[1, 0], [0, 1]], r: [[2, 3], [0, 4]] } });
  assert.equal(res.status, 200);
  assert.equal(res.headers['x-request-id'], 'req-123');
  assert.deepEqual(res.body.esDiagonal, { q: true, r: false });
  assert.equal(res.body.suma, 11);
});

test('rechaza peticiones sin token, con token falso o vencido', async () => {
  const cuerpo = { matrices: { q: [[1]] } };
  const casos = [
    undefined,
    'Bearer no.es.token',
    'Bearer ' + jwt.sign({ sub: 'x' }, 'otro-secreto', { issuer: EMISOR, audience: AUDIENCIA }),
    token({ expiresIn: -10 }),
    token({ audience: 'otra-api' }),
  ];
  for (const autorizacion of casos) {
    const req = request(app).post('/api/v1/estadisticas').send(cuerpo);
    if (autorizacion) req.set('Authorization', autorizacion);
    const res = await req;
    assert.equal(res.status, 401, String(autorizacion));
    assert.match(res.headers['content-type'], /application\/problem\+json/);
  }
});

test('entradas inválidas responden 400 con Problem Details', async () => {
  for (const cuerpo of [{}, { matrices: { q: [[1, 2], [3]] } }, { matrices: { q: [['a']] } }]) {
    const res = await request(app).post('/api/v1/estadisticas').set('Authorization', token()).send(cuerpo);
    assert.equal(res.status, 400, JSON.stringify(cuerpo));
    assert.equal(res.body.title, 'Matrices inválidas');
  }
  const res = await request(app)
    .post('/api/v1/estadisticas')
    .set('Authorization', token())
    .set('Content-Type', 'application/json')
    .send('{no es json');
  assert.equal(res.status, 400);
});

test('ruta inexistente responde 404', async () => {
  const res = await request(app).get('/no-existe');
  assert.equal(res.status, 404);
});
