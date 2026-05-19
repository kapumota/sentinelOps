#!/usr/bin/env python3
import argparse
import json
import re
from pathlib import Path


def add_finding(findings, fid, severity, category, source, message, recommendation):
    findings.append({
        "id": fid,
        "severity": severity,
        "category": category,
        "source": source,
        "message": message,
        "recommendation": recommendation,
    })


def load_text(path: Path) -> str:
    if not path.exists():
        return ""
    return path.read_text(encoding="utf-8", errors="ignore")


def find_test_files(project_root: Path):
    return list(project_root.rglob("*_test.go"))


def scan_dockerfile(project_root: Path, findings: list):
    dockerfile = project_root / "Dockerfile"
    if not dockerfile.exists():
        add_finding(
            findings,
            "AUD-DOCKER-001",
            "high",
            "container",
            "external-python",
            "No se encontró Dockerfile en la raíz del proyecto.",
            "Agregar un Dockerfile multi-stage con usuario no root.",
        )
        return

    content = load_text(dockerfile)
    lines = content.splitlines()
    from_lines = [line.strip() for line in lines if line.strip().upper().startswith("FROM ")]
    user_lines = [line.strip() for line in lines if line.strip().upper().startswith("USER ")]

    if len(from_lines) < 2:
        add_finding(
            findings,
            "AUD-DOCKER-002",
            "medium",
            "container",
            "external-python",
            "El Dockerfile no evidencia un build multi-stage.",
            "Separar stage de build y stage runtime.",
        )

    for line in from_lines:
        if ":latest" in line:
            add_finding(
                findings,
                "AUD-DOCKER-003",
                "high",
                "supply-chain",
                "external-python",
                f"Se detectó uso de tag mutable en Dockerfile: {line}",
                "Usar una versión fija en la imagen base.",
            )

    if not user_lines:
        add_finding(
            findings,
            "AUD-DOCKER-004",
            "high",
            "container",
            "external-python",
            "El contenedor no define explícitamente un usuario runtime.",
            "Agregar USER appuser o equivalente no privilegiado.",
        )
    else:
        last_user = user_lines[-1].split(maxsplit=1)[1].strip().lower()
        if last_user in {"root", "0"}:
            add_finding(
                findings,
                "AUD-DOCKER-005",
                "critical",
                "container",
                "external-python",
                "El contenedor se ejecuta como root.",
                "Usar usuario no root en la imagen final.",
            )


def scan_auth_service(project_root: Path, findings: list):
    auth_file = project_root / "internal" / "auth" / "service.go"
    if not auth_file.exists():
        add_finding(
            findings,
            "AUD-AUTH-001",
            "medium",
            "identity",
            "external-python",
            "No se encontró el servicio de autenticación esperado.",
            "Verificar la ubicación del módulo internal/auth/service.go.",
        )
        return

    content = load_text(auth_file)
    static_password_pattern = re.compile(r'password:\s*"[^"]+"')
    matches = static_password_pattern.findall(content)

    if matches:
        add_finding(
            findings,
            "AUD-AUTH-002",
            "medium",
            "identity",
            "external-python",
            "Se detectan credenciales estáticas embebidas en el código.",
            "Mover credenciales a variables de entorno, Secret o backend de identidad.",
        )


def scan_metrics(project_root: Path, findings: list):
    metrics_file = project_root / "internal" / "metrics" / "metrics.go"
    if not metrics_file.exists():
        add_finding(
            findings,
            "AUD-OBS-001",
            "medium",
            "observability",
            "external-python",
            "No se encontró el módulo de métricas.",
            "Agregar exposición de métricas para trazabilidad operativa.",
        )
        return

    content = load_text(metrics_file)
    if "promhttp" not in content:
        add_finding(
            findings,
            "AUD-OBS-002",
            "low",
            "observability",
            "external-python",
            "El módulo de métricas no parece exponer /metrics.",
            "Incorporar promhttp.Handler para integración con Prometheus.",
        )


def scan_tests(project_root: Path, findings: list):
    tests = find_test_files(project_root)
    if len(tests) < 2:
        add_finding(
            findings,
            "AUD-TEST-001",
            "medium",
            "quality",
            "external-python",
            "La cobertura mínima de pruebas parece insuficiente para un laboratorio profesional.",
            "Agregar más pruebas unitarias e integrar pruebas end-to-end.",
        )


def scan_profile(profile: str, findings: list):
    if profile == "insecure":
        add_finding(
            findings,
            "AUD-PROFILE-001",
            "high",
            "runtime",
            "external-python",
            "El perfil activo es insecure y está diseñado para demostración de fallos.",
            "Usar el perfil hardened para escenarios de validación positiva.",
        )
        add_finding(
            findings,
            "AUD-PROFILE-002",
            "medium",
            "policy",
            "external-python",
            "El perfil insecure requiere controles compensatorios y no debe asumirse como seguro.",
            "Etiquetar el ambiente como demostrativo y bloquear su promoción.",
        )


def build_result(findings: list, project_root: Path, profile: str):
    severities = {f["severity"] for f in findings}
    status = "pass"
    if severities.intersection({"critical", "high", "medium"}):
        status = "fail"
    elif severities:
        status = "warn"

    return {
        "status": status,
        "profile": profile,
        "total_findings": len(findings),
        "project_root": str(project_root.resolve()),
        "findings": findings,
    }


def main():
    parser = argparse.ArgumentParser(description="SentinelOps external audit")
    parser.add_argument("--profile", default="hardened", help="Perfil operativo a auditar")
    parser.add_argument("--project-root", default=".", help="Raíz del proyecto")
    args = parser.parse_args()

    project_root = Path(args.project_root)
    findings = []

    scan_dockerfile(project_root, findings)
    scan_auth_service(project_root, findings)
    scan_metrics(project_root, findings)
    scan_tests(project_root, findings)
    scan_profile(args.profile, findings)

    print(json.dumps(build_result(findings, project_root, args.profile), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
