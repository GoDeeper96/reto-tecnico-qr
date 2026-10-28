/**
 * Errores con el formato RFC 7807 (Problem Details), igual que en la API en Go.
 */
export class Problema extends Error {
  /**
   * @param {number} estado
   * @param {string} titulo
   * @param {string} [detalle]
   */
  constructor(estado, titulo, detalle) {
    super(`${titulo}: ${detalle ?? ''}`);
    this.estado = estado;
    this.titulo = titulo;
    this.detalle = detalle;
  }
}

/** Middleware final de Express: convierte cualquier error en application/problem+json. */
export function manejarError(err, req, res, _next) {
  let p = err;
  if (err?.type === 'entity.parse.failed') {
    p = new Problema(400, 'Cuerpo inválido', 'El cuerpo debe ser un JSON válido');
  } else if (err?.type === 'entity.too.large') {
    p = new Problema(413, 'Cuerpo demasiado grande', 'El cuerpo supera el límite permitido');
  } else if (!(err instanceof Problema)) {
    console.error(JSON.stringify({ requestId: req.id, error: String(err?.stack ?? err) }));
    p = new Problema(500, 'Error interno', 'Ocurrió un error inesperado');
  }
  if (p.estado === 401) res.set('WWW-Authenticate', 'Bearer realm="reto-matrices"');
  res
    .status(p.estado)
    .type('application/problem+json')
    .send(JSON.stringify({ type: 'about:blank', title: p.titulo, status: p.estado, detail: p.detalle, requestId: req.id }));
}
