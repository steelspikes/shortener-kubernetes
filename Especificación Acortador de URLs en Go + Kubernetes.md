# Especificación: Acortador de URLs en Go + Kubernetes

Nov 2, 2026 · @Brandon

## 1. Resumen y objetivos

Acortador de URLs con analytics de clics, hecho en Go y desplegado en Kubernetes. La lógica de negocio es mínima a propósito: el foco es aprender Kubernetes.

**Funcionalidad:**

- Crear un link corto a partir de una URL larga.
- Redirigir `GET /{code}` a la URL original (302), rápido y con caché.
- Registrar cada redirección exitosa como un clic, en segundo plano.
- Consultar estadísticas por link (total, por día, navegador, referer).
- Exportar estadísticas a CSV (tarea en segundo plano) y limpiar links expirados (tarea programada).

**Qué cuenta como clic:** toda redirección exitosa (código existente, no expirado), ignorando `HEAD` y bots obvios por User-Agent. Los 404 no cuentan. Clics únicos quedan como mejora futura.

**Lo que se aprende en Kubernetes:**

- Deployments, Services, Ingress, ConfigMaps, Secrets, Namespaces.
- StatefulSet con PVC, Jobs y CronJobs.
- Probes, requests/limits y graceful shutdown.
- Escalado con HPA (CPU) y KEDA (cola).
- Helm o Kustomize, observabilidad, RBAC, NetworkPolicies y GitOps con ArgoCD.

## 2. Arquitectura

La redirección solo toca Redis; los clics se guardan después, en segundo plano, a través de un stream.

&#91;embedded content: arquitectura · 3 servicios, Redis y Postgres\]

En color, `api`: el camino caliente. `worker` nunca atiende a un cliente y `stats` solo lee de Postgres.

**Flujo de una redirección**

1. El cliente llama `GET /abc123`; el Ingress lo manda a `api`.
2. `api` busca `url:abc123` en Redis; si no está, consulta Postgres y lo guarda con TTL.
3. `api` publica el clic en el stream `clicks` y responde `302` sin esperar nada más.
4. `worker` lee el stream en lotes, inserta en Postgres y confirma con `XACK`.
5. `stats` lee `clicks` de Postgres cuando alguien pide estadísticas.

**Flujo de creación:** `POST /links` guarda el link en Postgres y precalienta la caché en Redis.

**Decisiones clave**

- La cola vive en Redis Streams porque cada clic es una escritura de alto volumen y perder unos pocos es aceptable.
- Postgres guarda la verdad; Redis es efímero (con AOF para reducir pérdidas).
- Un solo Postgres compartido, con permisos por servicio como mejora de seguridad.

## 3. Módulos

Tres servicios permanentes en Go, dos tareas en segundo plano y dos piezas de infraestructura.

| Módulo | Tipo en K8s | Qué hace | Habla con |
| --- | --- | --- | --- |
| `api` | Deployment | Crea links, redirige y publica el evento de clic | Redis, Postgres |
| `worker` | Deployment | Consume el stream, agrupa clics y los guarda | Redis, Postgres |
| `stats` | Deployment | Solo lectura: agregaciones por link | Postgres |
| `migrate` | Job | Aplica migraciones SQL al desplegar | Postgres |
| `cleaner` | CronJob | Borra links expirados cada noche | Postgres, Redis |
| `exporter` (opcional) | Job | Genera un CSV de estadísticas y lo sube a MinIO | Postgres, MinIO |
| `postgres` | StatefulSet + PVC | Datos persistentes | — |
| `redis` | StatefulSet + PVC | Caché de URLs y stream de clics (AOF activado) | — |

**api (camino caliente)**

- `POST /links` valida la URL y guarda en Postgres. Si el body trae `code`, lo valida (`[a-zA-Z0-9_-]`, máx. 16 caracteres, y no puede ser una ruta reservada: `healthz`, `readyz`, `metrics`, `links`); si no, genera uno aleatorio de 7 caracteres y reintenta ante colisión. Luego precalienta Redis.
- Si el `code` pedido ya existe, responde `409` sin reintentar.
- `GET /{code}` busca en Redis; si falla, busca en Postgres y guarda en Redis con TTL.
- Si el link existe, hace `XADD` al stream `clicks` y responde 302 sin esperar al worker.
- Ignora `HEAD` y User-Agents de bots conocidos al decidir si publica el evento.

**worker**

- Lee con `XREADGROUP` (grupo `click-workers`) en lotes de hasta 100 mensajes o cada 2 segundos.
- Inserta el lote en `clicks` en una sola sentencia y solo después hace `XACK`.
- Al arrancar, reclama mensajes pendientes de workers caídos con `XAUTOCLAIM`.
- Tras 5 fallos, mueve el mensaje al stream `clicks:dead`.

**stats**

- `GET /links/{code}/stats` devuelve total, clics por día, top navegadores y top referers.
- Solo consulta `clicks` y `links`; no toca Redis.

**cleaner y exporter**

- `cleaner` borra filas con `expires_at` vencido y limpia su clave en Redis.
- `exporter` recibe un `code`, escribe `stats-<code>.csv` en MinIO y registra el resultado en la tabla `jobs`.

## 4. Modelo de datos

Una sola base Postgres compartida (`shortener`) con tres tablas, más estructuras en Redis para caché y cola.

**Postgres: tabla `links`**

| Columna | Tipo | Notas |
| --- | --- | --- |
| `id` | bigserial | Clave primaria |
| `code` | varchar(16) | Único, indexado |
| `url` | text | URL original, obligatoria |
| `created_at` | timestamptz | Por defecto `now()` |
| `expires_at` | timestamptz | Opcional; `null` = no expira |

**Postgres: tabla `clicks`**

| Columna | Tipo | Notas |
| --- | --- | --- |
| `id` | bigserial | Clave primaria |
| `code` | varchar(16) | Referencia a `links.code` |
| `clicked_at` | timestamptz | Momento de la redirección |
| `ip_hash` | text | Hash de la IP, nunca la IP cruda |
| `user_agent` | text | Tal como llega |
| `browser` | text | Derivado del User-Agent |
| `referer` | text | Puede ser vacío |

Índice compuesto en `(code, clicked_at)` para las consultas de `stats`.

**Postgres: tabla `jobs`** (solo si implementas `exporter`)

| Columna | Tipo | Notas |
| --- | --- | --- |
| `id` | uuid | Clave primaria |
| `type` | text | Por ejemplo `export_csv` |
| `status` | text | `pending`, `running`, `done`, `failed` |
| `result_url` | text | Ubicación del archivo en MinIO |
| `created_at`, `updated_at` | timestamptz | Seguimiento |

**Redis**

| Clave | Tipo | Uso |
| --- | --- | --- |
| `url:<code>` | string con TTL de 24 h | Caché de redirección |
| `clicks` | stream, recorte `MAXLEN ~ 100000` | Cola de eventos de clic |
| `click-workers` | consumer group sobre `clicks` | Reparto entre workers |
| `clicks:dead` | stream | Mensajes que fallaron 5 veces |

Cada evento del stream lleva `code`, `ts`, `ip_hash`, `ua` y `referer`.

Las migraciones viven en `migrations/` (golang-migrate) y las aplica el Job `migrate`.

## 5. Endpoints

| Servicio | Método y ruta | Respuesta |
| --- | --- | --- |
| `api` (8080) | `POST /links` con `{url, code?, expires_at?}` | `201` con `{code, short_url}`; `400` si la URL o el `code` son inválidos; `409` si el `code` ya existe |
| `api` | `GET /{code}` | `302` a la URL original; `404` si no existe; `410` si expiró |
| `stats` (8081) | `GET /links/{code}/stats` | `200` con total, clics por día, top navegadores y top referers; `404` si no existe |
| `stats` (opcional) | `POST /links/{code}/export` | `202` con `{job_id}`; crea el Job `exporter` |
| `stats` (opcional) | `GET /jobs/{id}` | Estado y `result_url` cuando termina |
| Todos (`worker` en 9090) | `GET /healthz` | `200` si el proceso vive (liveness) |
| Todos | `GET /readyz` | `200` si las dependencias responden (readiness) |
| Todos | `GET /metrics` | Métricas en formato Prometheus |

**Reglas comunes**

- Respuestas JSON con errores de la forma `{"error": "mensaje"}`.
- Logs estructurados con `slog`, un campo `service` y un `request_id` por petición.
- Apagado ordenado: al recibir `SIGTERM`, dejar de aceptar tráfico y terminar las peticiones en curso con `http.Server.Shutdown` (el worker termina su lote y hace `XACK` antes de salir).
- Si implementas el export desde `stats`, ese servicio necesita permiso RBAC para crear Jobs (buen ejercicio, ver fase 8).

## 6. Recursos de Kubernetes

Todo vive en el namespace `shortener`, sobre un clúster local con kind o k3d. La observabilidad va en un namespace aparte, `monitoring`.

**Cargas de trabajo (pods)**

| Recurso | Tipo | Réplicas | Puerto | Recursos y notas |
| --- | --- | --- | --- | --- |
| `api` | Deployment | 2 (HPA de 2 a 10) | 8080 | Requests 50m CPU y 64Mi; limits 200m y 128Mi; HPA al 60% de CPU |
| `worker` | Deployment | 1 (KEDA de 1 a 5) | 9090 | Escala por pendientes en el stream `clicks`; `terminationGracePeriodSeconds` de 30 |
| `stats` | Deployment | 1 | 8081 | Mismos límites que `api` |
| `postgres` | StatefulSet | 1 | 5432 | PVC de 5Gi; startup probe con `pg_isready` |
| `redis` | StatefulSet | 1 | 6379 | PVC de 1Gi; AOF activado (`appendonly yes`) |
| `migrate` | Job | 1 | — | Corre antes de cada despliegue (hook de Helm o PreSync de ArgoCD) |
| `cleaner` | CronJob | — | — | Diario a las 03:00; `concurrencyPolicy: Forbid` |
| `minio` (opcional) | StatefulSet | 1 | 9000 | PVC de 2Gi; solo si haces `exporter` |

**Red**

- Services `ClusterIP` para `api`, `stats` y `redis`; Service headless para `postgres`.
- Ingress con ingress-nginx y dos hosts: `go.localtest.me` hacia `api` y `stats.localtest.me` hacia `stats` (el dominio `localtest.me` resuelve a 127.0.0.1).
- NetworkPolicy de denegación por defecto, con permisos explícitos: `ingress` a `api` y `stats`; `api`, `worker` y `cleaner` a `redis`; `api`, `worker`, `stats`, `cleaner` y `migrate` a `postgres`.

**Configuración y secretos**

- ConfigMap `shortener-config`: `REDIS_ADDR`, `DB_HOST`, `DB_NAME`, `CACHE_TTL`, `BATCH_SIZE`, `BASE_URL`.
- Secret `postgres-credentials`: usuario y contraseña, inyectados como variables de entorno.

**Salud y seguridad de los pods**

- Liveness en `/healthz`, readiness en `/readyz`, y `startupProbe` para Postgres.
- `securityContext`: `runAsNonRoot`, `readOnlyRootFilesystem` y `allowPrivilegeEscalation: false`.
- Imágenes multi-stage con base distroless y etiquetas fijas, nunca `latest`.
- Un ServiceAccount por servicio; el de `stats` con Role mínimo para crear Jobs solo si haces `exporter`.
- PodDisruptionBudget en `api` con `minAvailable: 1`.

**Observabilidad (namespace `monitoring`)**

- kube-prometheus-stack (Prometheus y Grafana) con un ServiceMonitor por servicio.
- Loki y Promtail para logs.
- Dashboards: peticiones por segundo y latencia de `api`, tasa de aciertos de caché, largo del stream, clics procesados por segundo.

## 7. Stack y estructura del repo

| Necesidad | Herramienta |
| --- | --- |
| HTTP | `net/http` de la librería estándar (o `chi` si quieres rutas más cómodas) |
| Postgres | `pgx` |
| Redis | `go-redis` |
| Migraciones | `golang-migrate` |
| Logs | `slog` |
| Métricas | `prometheus/client_golang` |
| Clúster local | kind o k3d |
| Empaquetado | Helm (o Kustomize con overlays `dev` y `prod`) |
| Escalado | HPA (metrics-server) y KEDA |
| CI y entrega | GitHub Actions y ArgoCD |
| Pruebas de carga | `k6` o `hey` |

**Estructura sugerida**

```
shortener/
├── services/
│   ├── api/       (cmd/, internal/, Dockerfile)
│   ├── worker/
│   ├── stats/
│   ├── cleaner/
│   └── exporter/  (opcional)
├── internal/shared/   (config, db, redis, logging, metrics)
├── migrations/
├── deploy/
│   ├── helm/shortener/
│   └── argocd/
├── loadtest/k6.js
├── .github/workflows/
├── docker-compose.yml
└── Makefile
```

**Convenciones**

- Un solo módulo Go en la raíz, un binario por servicio en `services/<nombre>/cmd`.
- Toda la configuración llega por variables de entorno (12-factor), sin archivos dentro de la imagen.
- Un `Makefile` con `make build`, `make test`, `make cluster-up`, `make deploy` para no repetir comandos.

## 8. Fases de desarrollo

Cada fase termina con algo que funciona; no avances sin cumplir su criterio de salida.

### Fase 1. Base local (sin Kubernetes)

- [ ] `api`: crear link y redirigir, usando Postgres.
- [ ] Añadir caché en Redis y publicar el evento al stream.
- [ ] `worker`: consumir, insertar por lotes y confirmar.
- [ ] `stats`: endpoint de estadísticas.
- [ ] Dockerfile multi-stage por servicio y `docker-compose.yml` con todo.

**Listo cuando:** creas un link, lo visitas 10 veces y `stats` muestra 10 clics.

### Fase 2. Primer despliegue en Kubernetes

- [ ] Crear clúster kind o k3d y cargar las imágenes.
- [ ] Namespace, Deployments y Services de los tres servicios.
- [ ] ConfigMap y Secret.
- [ ] Instalar ingress-nginx y publicar `go.localtest.me` y `stats.localtest.me`.

**Listo cuando:** la redirección funciona desde el navegador a través del Ingress.

### Fase 3. Estado y migraciones

- [ ] Postgres y Redis como StatefulSet con PVC.
- [ ] Job `migrate` que aplica las migraciones.
- [ ] CronJob `cleaner`.
- [ ] Borrar el pod de Postgres y comprobar que los datos siguen.

**Listo cuando:** reinicias Postgres y Redis sin perder links ni clics.

### Fase 4. Salud y recursos

- [ ] Probes de liveness, readiness y startup.
- [ ] Requests y limits en todos los pods.
- [ ] Apagado ordenado con `SIGTERM` en `api` y `worker`.
- [ ] Rolling update de `api` mientras corre una carga ligera.

**Listo cuando:** un despliegue nuevo no produce ni un error 5xx.

### Fase 5. Escalado

- [ ] Instalar metrics-server y crear el HPA de `api`.
- [ ] Prueba de carga con `k6` y ver cómo suben las réplicas.
- [ ] Instalar KEDA y escalar `worker` según el stream.

**Listo cuando:** bajo carga, `api` y `worker` suben réplicas y vuelven a bajar solos.

### Fase 6. Empaquetado

- [ ] Chart de Helm (o Kustomize) con valores para `dev` y `prod`.
- [ ] El Job `migrate` como hook previo al despliegue.

**Listo cuando:** `helm install` levanta todo el sistema desde cero con un comando.

### Fase 7. Observabilidad

- [ ] Endpoint `/metrics` en cada servicio.
- [ ] kube-prometheus-stack con ServiceMonitors.
- [ ] Dashboard en Grafana y logs en Loki.

**Listo cuando:** ves la latencia de `api` y el largo del stream en un dashboard.

### Fase 8. Seguridad

- [ ] ServiceAccounts por servicio y RBAC mínimo.
- [ ] NetworkPolicies de denegación por defecto con permisos explícitos.
- [ ] `securityContext` no-root y sistema de archivos de solo lectura.
- [ ] Secrets fuera del repo (Sealed Secrets o External Secrets).

**Listo cuando:** un pod de prueba no puede alcanzar Postgres, pero `api` sí.

### Fase 9. CI y GitOps

- [ ] GitHub Actions: tests, build y push de imágenes con etiqueta por commit.
- [ ] ArgoCD sincronizando `deploy/` con el clúster.

**Listo cuando:** un `git push` termina desplegado sin comandos manuales.

### Fase 10. Extras opcionales

- [ ] `exporter` como Job y tabla `jobs` con seguimiento de estado.
- [ ] cert-manager con TLS en el Ingress.
- [ ] Un operator mínimo en Go con un CRD `ShortLink`.
- [ ] Service mesh (Linkerd) para mTLS entre servicios.

## 9. Criterios de aceptación y mejoras

El proyecto está completo cuando se cumple todo esto:

- [ ] Un solo comando levanta el clúster y despliega el sistema completo.
- [ ] Crear y visitar un link funciona de punta a punta por el Ingress.
- [ ] Matar cualquier pod (`api`, `worker`, `stats`, `postgres`, `redis`) no pierde datos y el sistema se recupera solo.
- [ ] Bajo carga con `k6`, `api` y `worker` escalan solos y vuelven a su tamaño mínimo después.
- [ ] Un despliegue nuevo se hace sin errores visibles para quien usa el servicio.
- [ ] Grafana muestra latencia, tasa de aciertos de caché y largo del stream.
- [ ] Las NetworkPolicies bloquean el tráfico que no está permitido.
- [ ] Un `git push` a `main` termina desplegado por ArgoCD.

**Mejoras futuras (fuera del alcance de la v1)**

- Clics únicos con una ventana de 24 horas.
- Detección de país con GeoIP.
- Límite de creación por IP (rate limiting con Redis).
- Alta disponibilidad de Postgres con un operator (CloudNativePG).
- Réplicas de Redis con Sentinel.
