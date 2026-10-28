import { useEffect, useState } from 'react';
import { ErrorApi, apiKeyDesdeEntorno, calcularAutovalores, factorizarQR, guardarApiKey, obtenerApiKey, salud } from './api.js';
import { EJEMPLOS, METODOS, aTexto } from './ejemplos.js';

const formato = (v) => {
  const a = Math.abs(v);
  if (a < 1e-10) return '0';
  if (a >= 1e4 || a < 1e-3) return v.toExponential(2);
  return v.toFixed(4);
};

const NOMBRES_REGLAS = { qOrtogonal: 'Q ortogonal (QᵀQ = I)', rTriangular: 'R triangular superior', reconstruccion: 'Q·R = A' };

function EstadoServicio({ nombre, resultado }) {
  const ok = resultado?.status === 'fulfilled';
  return (
    <span className={`servicio ${ok ? 'ok' : 'caido'}`} title={ok ? `versión ${resultado.value.version}` : 'sin respuesta'}>
      <span className="punto" /> {nombre}
    </span>
  );
}

function TablaMatriz({ titulo, matriz, resaltarDiagonal }) {
  return (
    <figure className="matriz">
      <figcaption>
        {titulo} <span className="dim">{matriz.length}×{matriz[0].length}</span>
      </figcaption>
      <div className="desplazable">
        <table>
          <tbody>
            {matriz.map((fila, i) => (
              <tr key={i}>
                {fila.map((v, j) => (
                  <td key={j} className={Math.abs(v) < 1e-10 ? 'cero' : resaltarDiagonal && i === j ? 'diagonal' : ''}>
                    {formato(v)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </figure>
  );
}

function Validaciones({ validaciones }) {
  return (
    <ul className="validaciones">
      {Object.entries(validaciones).map(([regla, r]) => (
        <li key={regla} className={r.cumple ? 'cumple' : 'falla'}>
          <span className="icono">{r.cumple ? '✓' : '✗'}</span>
          <span>{NOMBRES_REGLAS[regla] ?? regla}</span>
          <span className="error">error {r.error.toExponential(1)}</span>
        </li>
      ))}
    </ul>
  );
}

function Estadisticas({ e }) {
  const tarjetas = [
    ['Máximo', formato(e.maximo)],
    ['Mínimo', formato(e.minimo)],
    ['Suma total', formato(e.suma)],
    ['Promedio', formato(e.promedio)],
    ['Valores', e.cantidadValores.toLocaleString('es')],
    ['¿Alguna diagonal?', e.algunaDiagonal ? 'Sí' : 'No'],
  ];
  return (
    <>
      <div className="tarjetas">
        {tarjetas.map(([titulo, valor]) => (
          <div className="tarjeta" key={titulo}>
            <span className="etiqueta">{titulo}</span>
            <span className="valor">{valor}</span>
          </div>
        ))}
      </div>
      <p className="nota">
        Diagonal por matriz:{' '}
        {Object.entries(e.esDiagonal)
          .map(([n, d]) => `${n.toUpperCase()} ${d ? 'sí' : 'no'}`)
          .join(' · ')}
      </p>
    </>
  );
}

function AccesoApiKey({ hayClave, onCambio }) {
  const [valor, setValor] = useState('');
  if (apiKeyDesdeEntorno()) return null; // desarrollo local: viene de VITE_API_KEY
  if (hayClave) {
    return (
      <p className="acceso">
        API key ingresada (solo en esta pestaña).{' '}
        <button className="enlace" onClick={() => onCambio('')}>
          Cambiar
        </button>
      </p>
    );
  }
  return (
    <form
      className="panel acceso-form"
      onSubmit={(e) => {
        e.preventDefault();
        onCambio(valor.trim());
      }}
    >
      <label htmlFor="api-key">API key (la recibió por correo)</label>
      <div className="acciones">
        <input id="api-key" type="password" autoComplete="off" value={valor} onChange={(e) => setValor(e.target.value)} />
        <button className="primario" disabled={!valor.trim()}>
          Guardar
        </button>
      </div>
    </form>
  );
}

export default function App() {
  const [texto, setTexto] = useState(aTexto(EJEMPLOS[0].matriz));
  const [metodo, setMetodo] = useState('householder');
  const [estado, setEstado] = useState({ cargando: false });
  const [servicios, setServicios] = useState(null);
  const [hayClave, setHayClave] = useState(Boolean(obtenerApiKey()));

  function cambiarClave(clave) {
    guardarApiKey(clave);
    setHayClave(Boolean(clave));
    setEstado({});
  }

  useEffect(() => {
    salud().then(setServicios);
  }, []);

  async function ejecutar(accion) {
    let matriz;
    try {
      matriz = JSON.parse(texto);
    } catch {
      setEstado({ error: { titulo: 'JSON inválido', detalle: 'Escriba la matriz como un array de arrays, por ejemplo [[1, 2], [3, 4]].' } });
      return;
    }
    setEstado({ cargando: true });
    try {
      if (accion === 'qr') setEstado({ qr: await factorizarQR(matriz, metodo) });
      else setEstado({ autovalores: await calcularAutovalores(matriz) });
    } catch (err) {
      const p = err instanceof ErrorApi ? err.problema : null;
      if (p?.title === 'API key inválida' && !apiKeyDesdeEntorno()) {
        guardarApiKey('');
        setHayClave(false);
      }
      setEstado({
        error: {
          titulo: p?.title ?? 'No se pudo conectar con la API',
          detalle: p?.detail ?? err.message,
          estado: err.estado,
          validaciones: p?.extra?.validaciones,
        },
      });
    }
  }

  const { qr, autovalores, error, cargando } = estado;
  return (
    <main>
      <header>
        <div>
          <h1>Factorización QR</h1>
          <p className="subtitulo">API en Go (QR y validaciones) → API en Node (estadísticas)</p>
        </div>
        <div className="servicios">
          <EstadoServicio nombre="api-go" resultado={servicios?.go} />
          <EstadoServicio nombre="api-node" resultado={servicios?.node} />
        </div>
      </header>

      <AccesoApiKey hayClave={hayClave} onCambio={cambiarClave} />

      <section className="panel entrada">
        <div className="ejemplos">
          {EJEMPLOS.map((e) => (
            <button key={e.nombre} className="chip" onClick={() => setTexto(aTexto(e.matriz))}>
              {e.nombre}
            </button>
          ))}
        </div>
        <label htmlFor="matriz">Matriz (array de arrays de números)</label>
        <textarea id="matriz" value={texto} onChange={(e) => setTexto(e.target.value)} spellCheck={false} rows={8} />
        <div className="acciones">
          <label>
            Método
            <select value={metodo} onChange={(e) => setMetodo(e.target.value)}>
              {METODOS.map((m) => (
                <option key={m.valor} value={m.valor}>
                  {m.nombre}
                </option>
              ))}
            </select>
          </label>
          <button className="primario" disabled={cargando} onClick={() => ejecutar('qr')}>
            {cargando ? 'Calculando…' : 'Calcular QR'}
          </button>
          <button disabled={cargando} onClick={() => ejecutar('autovalores')}>
            Autovalores
          </button>
        </div>
      </section>

      {error && (
        <section className="panel problema" role="alert">
          <h2>
            {error.estado ? `${error.estado} · ` : ''}
            {error.titulo}
          </h2>
          <p>{error.detalle}</p>
          {error.validaciones && <Validaciones validaciones={error.validaciones} />}
        </section>
      )}

      {qr && (
        <>
          <section className="panel">
            <h2>
              Validaciones <span className="dim">· {qr.metodo} · QR en {qr.duracionMs.qr} ms</span>
            </h2>
            <Validaciones validaciones={qr.validaciones} />
            <div className="matrices">
              <TablaMatriz titulo="Q" matriz={qr.q} />
              <TablaMatriz titulo="R" matriz={qr.r} resaltarDiagonal />
            </div>
          </section>
          <section className="panel">
            <h2>
              Estadísticas de Q y R <span className="dim">· API en Node · {qr.duracionMs.estadisticas} ms</span>
            </h2>
            <Estadisticas e={qr.estadisticas} />
          </section>
        </>
      )}

      {autovalores && (
        <section className="panel">
          <h2>
            Autovalores <span className="dim">· algoritmo QR · {autovalores.iteraciones} iteraciones</span>
          </h2>
          <div className="tarjetas">
            {autovalores.autovalores.map((v, i) => (
              <div className="tarjeta" key={i}>
                <span className="etiqueta">λ{i + 1}</span>
                <span className="valor">{formato(v)}</span>
              </div>
            ))}
          </div>
          <div className="matrices">
            <TablaMatriz titulo="Autovectores (por columna)" matriz={autovalores.autovectores} />
          </div>
        </section>
      )}
    </main>
  );
}
