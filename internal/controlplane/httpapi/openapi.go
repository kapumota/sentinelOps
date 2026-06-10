package httpapi

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.json
var openAPISpec []byte

const apiVersion = "v1"

func (s *Server) handleOpenAPIJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPISpec)
}

func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerHTML))
}

const swaggerHTML = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <title>SentinelOps API v1</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; line-height: 1.5; }
    code { background: #f2f2f2; padding: 0.1rem 0.3rem; }
    pre { background: #f7f7f7; padding: 1rem; overflow: auto; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #ddd; padding: 0.5rem; text-align: left; }
  </style>
</head>
<body>
  <h1>SentinelOps API v1</h1>
  <p>Documentación ligera compatible con OpenAPI. La especificación JSON está disponible en <code>/api/v1/docs/swagger.json</code>.</p>
  <h2>Health checks</h2>
  <table>
    <tr><th>Método</th><th>Ruta</th><th>Uso</th><th>Auth</th></tr>
    <tr><td>GET</td><td>/healthz/live</td><td>Liveness probe</td><td>No</td></tr>
    <tr><td>GET</td><td>/healthz/ready</td><td>Readiness probe</td><td>No</td></tr>
    <tr><td>GET</td><td>/healthz/startup</td><td>Startup probe</td><td>No</td></tr>
  </table>
  <h2>API administrativa</h2>
  <table>
    <tr><th>Método</th><th>Ruta</th><th>Uso</th><th>Auth</th></tr>
    <tr><td>GET</td><td>/api/v1/admin/status</td><td>Estado general</td><td>Basic</td></tr>
    <tr><td>GET</td><td>/api/v1/admin/sessions</td><td>Sesiones activas</td><td>Basic</td></tr>
    <tr><td>GET</td><td>/api/v1/admin/tunnels</td><td>Túneles activos</td><td>Basic</td></tr>
    <tr><td>POST</td><td>/api/v1/admin/tunnels/{id}/close</td><td>Cerrar túnel</td><td>Basic</td></tr>
  </table>
  <h2>Ejemplos</h2>
  <pre>curl -k https://localhost:9443/healthz/ready
curl -k -u "$APP_CONTROL_API_USER:$APP_CONTROL_API_PASSWORD" https://localhost:9443/api/v1/admin/status
curl -k https://localhost:9443/api/v1/docs/swagger.json</pre>
</body>
</html>`
