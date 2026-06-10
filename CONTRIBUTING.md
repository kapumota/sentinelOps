### Contribuir a SentinelOps

#### Flujo de ramas

El proyecto usa ramas por fase o por release.

Ejemplos:

```bash
git checkout main
git pull origin main
git checkout -b fase-10-benchmarks-rendimiento
```

Para releases:

```bash
git checkout -b release/v1.0.0-documentacion-final
```

#### Commits

Usa mensajes de commit en español y descriptivos.

Ejemplos:

```bash
git commit -m "fase 10: agrega benchmarks de rendimiento"
git commit -m "prepara documentacion final para v1.0.0"
```

Evita mensajes genéricos como `update`, `fix` sin contexto o `commit1`.

#### Pull requests

Cada PR debe incluir:

- resumen del cambio,
- archivos principales modificados,
- validación ejecutada,
- riesgos conocidos,
- notas de limpieza si generó artefactos locales.

#### Validación mínima

Antes de abrir un PR ejecuta:

```bash
make check-secrets
make vet
make test
make storage-test
TESTCONTAINERS_RYUK_DISABLED=true make test-integration
make rust-test
make validator-grpc-build
make validator-grpc-test
git diff --check
```

Para release final también ejecuta:

```bash
make release-verify
```

#### Estilo de documentación

La documentación debe usar títulos con `###` y subtítulos con `####`.

Los comentarios y cadenas de texto deben estar en español. Las firmas y nombres de funciones deben mantenerse en inglés.

No uses guiones largos. Usa guion simple cuando sea necesario.
