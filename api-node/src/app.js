/**
 * Adaptador HTTP (Express 5) de la API de estadísticas. Las dependencias se inyectan para poder probarla.
 */
import { randomUUID } from 'node:crypto';
import express from 'express';
import cors from 'cors';
import { requiereToken } from './auth.js';
import { Problema, manejarError } from './problemas.js';
import { ErrorEntrada, validarMatrices, calcularEstadisticas } from './domain/estadisticas.js';

/**
 * @param {{ obtenerSecretos: () => Promise<{ secretoJwt: string }>, version: string }} deps
 */
export function crearApp({ obtenerSecretos, version }) {
  const app = express();
  app.disable('x-powered-by');
  app.use(cors({ exposedHeaders: ['X-Request-ID'] })); // la autenticación va en un encabezado Bearer, no en cookies
  app.use((req, res, next) => {
    req.id = req.get('X-Request-ID') || randomUUID(); // mismo id que la API en Go, para seguir la petición
    res.set('X-Request-ID', req.id);
    const inicio = process.hrtime.bigint();
    res.on('finish', () => {
      const latenciaMs = Number(process.hrtime.bigint() - inicio) / 1e6;
      console.log(JSON.stringify({ requestId: req.id, metodo: req.method, ruta: req.path, estado: res.statusCode, latenciaMs }));
    });
    next();
  });
  app.use(express.json({ limit: '4mb' }));

  app.get('/health', (_req, res) => res.json({ estado: 'ok', servicio: 'api-node', version }));

  app.post('/api/v1/estadisticas', requiereToken(obtenerSecretos), (req, res) => {
    try {
      const matrices = validarMatrices(req.body?.matrices);
      res.json(calcularEstadisticas(matrices));
    } catch (err) {
      if (err instanceof ErrorEntrada) throw new Problema(400, 'Matrices inválidas', err.message);
      throw err;
    }
  });

  app.use((req, _res) => {
    throw new Problema(404, 'No encontrado', `No existe la ruta ${req.method} ${req.path}`);
  });
  app.use(manejarError);
  return app;
}
