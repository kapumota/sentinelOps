package kubernetes.security

import rego.v1

deny contains msg if {
  input.kind == "Deployment"
  container := input.spec.template.spec.containers[_]
  endswith(container.image, ":latest")
  msg := sprintf("container %s uses forbidden mutable tag latest", [container.name])
}