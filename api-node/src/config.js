/**
 * Configuración por variables de entorno. En AWS el secreto del JWT viene de Secrets Manager
 * (JWT_SECRET_ARN) y se lee una sola vez por contenedor; en local, de JWT_SECRET.
 */
let secretos;

export function obtenerSecretos() {
  secretos ??= cargarSecretos().catch((err) => {
    secretos = undefined; // permite reintentar en la siguiente petición
    throw err;
  });
  return secretos;
}

async function cargarSecretos() {
  if (process.env.JWT_SECRET) {
    return { secretoJwt: process.env.JWT_SECRET };
  }
  const arn = process.env.JWT_SECRET_ARN;
  if (!arn) {
    throw new Error('Faltan JWT_SECRET o JWT_SECRET_ARN');
  }
  // Import dinámico: el SDK de AWS solo se carga cuando se usa (en Lambda), no en local ni en las pruebas.
  const { SecretsManagerClient, GetSecretValueCommand } = await import('@aws-sdk/client-secrets-manager');
  const salida = await new SecretsManagerClient({}).send(new GetSecretValueCommand({ SecretId: arn }));
  return { secretoJwt: salida.SecretString };
}

export const version = process.env.VERSION ?? 'dev';
export const puerto = Number(process.env.PORT ?? 3000);
