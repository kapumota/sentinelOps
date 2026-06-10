### Política de seguridad

#### Alcance

SentinelOps es un laboratorio académico y técnico de DevSecOps. El proyecto incluye transporte TCP y SSH, API administrativa, OpenAPI, OPA, observabilidad, persistencia y benchmarks.

La versión soportada para reportes de seguridad es:

| Versión | Soporte |
|---|---|
| `v1.0.x` | Soportada |
| Versiones internas previas | Solo referencia histórica |

#### Reporte de vulnerabilidades

Para reportar una vulnerabilidad, usa GitHub Security Advisories si está habilitado en el repositorio. Si no está habilitado, abre un issue privado o contacta al mantenedor por los canales académicos definidos para el proyecto.

No publiques secretos, tokens, contraseñas reales, llaves privadas ni evidencia sensible en issues públicos.

#### Información esperada

Un reporte útil debe incluir:

- descripción del problema,
- archivo o componente afectado,
- pasos mínimos de reproducción,
- impacto esperado,
- versión, commit o rama analizada,
- salida de herramientas como CodeQL, gosec, Trivy o GitHub Security.

#### Política de secretos

El repositorio no debe contener credenciales reales. Los secretos locales se generan con:

```bash
make generate-secrets
```

La verificación mínima antes de un PR es:

```bash
make check-secrets
git diff --check
```

#### Alertas de seguridad

Las alertas de Code scanning, Trivy o gosec deben corregirse cuando afecten código productivo. Si una alerta aplica solo a pruebas, debe documentarse en:

```text
docs/security/alertas-code-scanning.md
```

No se deben cerrar alertas manualmente sin justificación técnica.
