/**
 * Verificación del JWT emitido por la API en Go (HS256, secreto compartido).
 * Debe coincidir con api-go/internal/auth: mismo emisor y audiencia.
 */
import jwt from 'jsonwebtoken';
import { Problema } from './problemas.js';

export const EMISOR = 'reto-matrices';
export const AUDIENCIA = 'reto-matrices-apis';

/**
 * @param {() => Promise<{ secretoJwt: string }>} obtenerSecretos
 */
export function requiereToken(obtenerSecretos) {
  return async (req, _res, next) => {
    const [esquema, token] = (req.get('Authorization') ?? '').split(' ');
    if (esquema !== 'Bearer' || !token) {
      throw new Problema(401, 'No autorizado', 'falta el encabezado Authorization: Bearer <token>');
    }
    const { secretoJwt } = await obtenerSecretos();
    try {
      req.usuario = jwt.verify(token, secretoJwt, { algorithms: ['HS256'], issuer: EMISOR, audience: AUDIENCIA });
    } catch (err) {
      throw new Problema(401, 'No autorizado', err.message);
    }
    next();
  };
}
