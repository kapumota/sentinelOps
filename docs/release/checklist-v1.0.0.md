### Checklist de release v1.0.0

#### Preparación

```bash
git checkout main
git pull origin main
git checkout -b release/v1.0.0-documentacion-final
```

#### Validación local

```bash
make release-verify
```

#### Validación de benchmarks

```bash
make benchmarks
make benchmarks-summary
make benchmarks-clean
```

#### Limpieza

```bash
make release-clean
git status --short
git diff --check
```

#### Pull request

El PR debe indicar:

- documentación final actualizada,
- badges agregados,
- release notes agregadas,
- versión `1.0.0`,
- validación ejecutada,
- chaos engineering reservado para versión posterior.

#### Tag final

Después del merge:

```bash
git checkout main
git pull origin main
git tag -a v1.0.0 -m "release v1.0.0: SentinelOps con seguridad, observabilidad, persistencia y benchmarks"
git push origin v1.0.0
```
