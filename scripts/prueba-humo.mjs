// Prueba de punta a punta contra las APIs desplegadas (local o AWS).
// Uso: node scripts/prueba-humo.mjs <url-api-go> <url-api-node> <api-key>
//   o con variables de entorno API_GO_URL, API_NODE_URL y API_KEY, por ejemplo desde un .env:
//   node --env-file=.env scripts/prueba-humo.mjs
import assert from 'node:assert/strict';

const [argGo, argNode, argKey] = process.argv.slice(2);
const urlGo = (argGo ?? process.env.API_GO_URL)?.replace(/\/$/, '');
const urlNode = (argNode ?? process.env.API_NODE_URL)?.replace(/\/$/, '');
const apiKey = argKey ?? process.env.API_KEY;
if (!urlGo || !urlNode || !apiKey) {
  console.error('Uso: node scripts/prueba-humo.mjs <url-api-go> <url-api-node> <api-key>  (o API_GO_URL, API_NODE_URL y API_KEY)');
  process.exit(1);
}

async function llamar(url, opciones = {}) {
  const res = await fetch(url, { ...opciones, headers: { 'Content-Type': 'application/json', ...opciones.headers } });
  const texto = await res.text();
  return { estado: res.status, cuerpo: texto ? JSON.parse(texto) : null };
}

const pasos = [];
async function paso(nombre, fn) {
  try {
    await fn();
    pasos.push(`✅ ${nombre}`);
  } catch (err) {
    pasos.push(`❌ ${nombre}: ${err.message}`);
  }
}

let token;
await paso('GET /health de las dos APIs', async () => {
  assert.equal((await llamar(`${urlGo}/health`)).cuerpo.servicio, 'api-go');
  assert.equal((await llamar(`${urlNode}/health`)).cuerpo.servicio, 'api-node');
});
await paso('POST /auth/token con API key incorrecta responde 401', async () => {
  assert.equal((await llamar(`${urlGo}/api/v1/auth/token`, { method: 'POST', headers: { 'X-Api-Key': 'incorrecta' } })).estado, 401);
});
await paso('POST /auth/token entrega un JWT', async () => {
  const r = await llamar(`${urlGo}/api/v1/auth/token`, { method: 'POST', headers: { 'X-Api-Key': apiKey } });
  assert.equal(r.estado, 200, JSON.stringify(r.cuerpo));
  token = r.cuerpo.accessToken;
});
const conToken = () => ({ Authorization: `Bearer ${token}` });

await paso('POST /qr sin token responde 401', async () => {
  assert.equal((await llamar(`${urlGo}/api/v1/qr`, { method: 'POST', body: '{"matriz":[[1]]}' })).estado, 401);
});
await paso('POST /qr: Go factoriza, valida y Node calcula las estadísticas', async () => {
  const r = await llamar(`${urlGo}/api/v1/qr`, { method: 'POST', headers: conToken(), body: JSON.stringify({ matriz: [[1, 2], [3, 4], [5, 6]] }) });
  assert.equal(r.estado, 200, JSON.stringify(r.cuerpo));
  assert.ok(r.cuerpo.validaciones.qOrtogonal.cumple && r.cuerpo.validaciones.rTriangular.cumple);
  assert.ok(Math.abs(r.cuerpo.r[0][0] - 5.916079783099616) < 1e-12);
  assert.equal(r.cuerpo.estadisticas.cantidadValores, 15);
  assert.equal(r.cuerpo.estadisticas.algunaDiagonal, false);
});
await paso('POST /qr con columnas perpendiculares: R es diagonal', async () => {
  const r = await llamar(`${urlGo}/api/v1/qr`, { method: 'POST', headers: conToken(), body: JSON.stringify({ matriz: [[1, 1], [1, -1], [0, 0]] }) });
  assert.equal(r.cuerpo.estadisticas.esDiagonal.r, true);
});
await paso('POST /qr con Gram-Schmidt y Hilbert 10x10 responde 422', async () => {
  const hilbert = Array.from({ length: 10 }, (_, i) => Array.from({ length: 10 }, (_, j) => 1 / (i + j + 1)));
  const r = await llamar(`${urlGo}/api/v1/qr`, { method: 'POST', headers: conToken(), body: JSON.stringify({ matriz: hilbert, metodo: 'gram-schmidt-clasico' }) });
  assert.equal(r.estado, 422);
  assert.equal(r.cuerpo.extra.validaciones.qOrtogonal.cumple, false);
});
await paso('POST /qr con matriz no rectangular responde 400', async () => {
  assert.equal((await llamar(`${urlGo}/api/v1/qr`, { method: 'POST', headers: conToken(), body: '{"matriz":[[1,2],[3]]}' })).estado, 400);
});
await paso('POST /autovalores de [[2,1],[1,2]] da 3 y 1', async () => {
  const r = await llamar(`${urlGo}/api/v1/autovalores`, { method: 'POST', headers: conToken(), body: '{"matriz":[[2,1],[1,2]]}' });
  assert.ok(Math.abs(r.cuerpo.autovalores[0] - 3) < 1e-9 && Math.abs(r.cuerpo.autovalores[1] - 1) < 1e-9);
});
await paso('POST /estadisticas directo a Node con el mismo token', async () => {
  const r = await llamar(`${urlNode}/api/v1/estadisticas`, { method: 'POST', headers: conToken(), body: '{"matrices":{"a":[[1,0],[0,2]]}}' });
  assert.equal(r.estado, 200, JSON.stringify(r.cuerpo));
  assert.equal(r.cuerpo.suma, 3);
});

console.log(pasos.join('\n'));
process.exit(pasos.some((p) => p.startsWith('❌')) ? 1 : 0);
