/** Punto de entrada en AWS Lambda: la misma app de Express, adaptada a los eventos de API Gateway. */
import serverlessExpress from '@codegenie/serverless-express';
import { crearApp } from './app.js';
import { obtenerSecretos, version } from './config.js';

// await en el nivel superior (solo con ES modules): el secreto se lee durante el arranque del contenedor,
// no en la primera petición, que así no paga esa espera.
await obtenerSecretos();

const proxy = serverlessExpress({ app: crearApp({ obtenerSecretos, version }) });

// Handler async de dos parámetros: Lambda deja de aceptar handlers con callback desde Node.js 24.
export const handler = (event, context) => proxy(event, context);
