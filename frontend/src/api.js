// Cliente de las dos APIs. El JWT se pide con la API key y se reutiliza hasta poco antes de vencer.
//
// La API key NO viaja en el build publicado: en local se lee de VITE_API_KEY (archivo .env, fuera de git);
// en el sitio publicado, quien la recibe la ingresa en la página y queda solo en esta pestaña (sessionStorage).
const URL_GO = (import.meta.env.VITE_API_GO_URL ?? 'http://localhost:8080').replace(/\/$/, '');
const URL_NODE = (import.meta.env.VITE_API_NODE_URL ?? 'http://localhost:3000').replace(/\/$/, '');
const API_KEY_ENTORNO = import.meta.env.VITE_API_KEY ?? '';
const CLAVE_SESION = 'reto-matrices-api-key';
let apiKeyMemoria = ''; // respaldo si el navegador bloquea sessionStorage

let token = null;
let venceEn = 0;

export class ErrorApi extends Error {
  constructor(estado, problema) {
    super(problema?.detail || problema?.title || `Error ${estado}`);
    this.estado = estado;
    this.problema = problema;
  }
}

async function pedir(url, opciones = {}) {
  const res = await fetch(url, { ...opciones, headers: { 'Content-Type': 'application/json', ...opciones.headers } });
  const texto = await res.text();
  const cuerpo = texto ? JSON.parse(texto) : null;
  if (!res.ok) throw new ErrorApi(res.status, cuerpo);
  return cuerpo;
}

function leerSesion() {
  try {
    return sessionStorage.getItem(CLAVE_SESION) ?? '';
  } catch {
    return '';
  }
}

/** API key disponible: la del entorno (desarrollo local) o la ingresada en la página. */
export const obtenerApiKey = () => API_KEY_ENTORNO || leerSesion() || apiKeyMemoria;
export const apiKeyDesdeEntorno = () => Boolean(API_KEY_ENTORNO);

export function guardarApiKey(clave) {
  token = null;
  apiKeyMemoria = clave;
  try {
    if (clave) sessionStorage.setItem(CLAVE_SESION, clave);
    else sessionStorage.removeItem(CLAVE_SESION);
  } catch {
    // sin sessionStorage (modo privado estricto): la clave dura solo mientras la página esté abierta
  }
}

async function obtenerToken(forzar = false) {
  if (!forzar && token && Date.now() < venceEn) return token;
  const apiKey = obtenerApiKey();
  if (!apiKey) throw new ErrorApi(401, { title: 'Falta la API key', detail: 'Ingrese la API key que recibió por correo.' });
  const r = await pedir(`${URL_GO}/api/v1/auth/token`, { method: 'POST', headers: { 'X-Api-Key': apiKey } });
  token = r.accessToken;
  venceEn = Date.now() + (r.expiresIn - 60) * 1000; // se renueva un minuto antes
  return token;
}

async function conToken(url, cuerpo) {
  const enviar = async (forzar) =>
    pedir(url, { method: 'POST', headers: { Authorization: `Bearer ${await obtenerToken(forzar)}` }, body: JSON.stringify(cuerpo) });
  try {
    return await enviar(false);
  } catch (err) {
    if (err instanceof ErrorApi && err.estado === 401) return enviar(true); // token vencido o rotado
    throw err;
  }
}

export const salud = () =>
  Promise.allSettled([pedir(`${URL_GO}/health`), pedir(`${URL_NODE}/health`)]).then(([go, node]) => ({ go, node }));

export const factorizarQR = (matriz, metodo) => conToken(`${URL_GO}/api/v1/qr`, { matriz, metodo });

export const calcularAutovalores = (matriz) => conToken(`${URL_GO}/api/v1/autovalores`, { matriz });
