/** Servidor HTTP para local y Docker. */
import { crearApp } from './app.js';
import { obtenerSecretos, version, puerto } from './config.js';

crearApp({ obtenerSecretos, version }).listen(puerto, () => {
  console.log(JSON.stringify({ mensaje: `api-node escuchando en el puerto ${puerto}` }));
});
