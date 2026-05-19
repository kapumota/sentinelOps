package policy

func BuildDeploymentInput(profile string) map[string]any {
	image := "sentinelops:1.0.0"
	privileged := false
	runAsNonRoot := true
	allowPrivilegeEscalation := false
	readOnlyRootFilesystem := true

	if profile == "insecure" {
		image = "sentinelops:latest"
		privileged = true
		runAsNonRoot = false
		allowPrivilegeEscalation = true
		readOnlyRootFilesystem = false
	}

	return map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name": "sentinelops",
			"labels": map[string]any{
				"app":     "sentinelops",
				"profile": profile,
			},
		},
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []map[string]any{
						{
							"name":  "server",
							"image": image,
							"securityContext": map[string]any{
								"privileged":               privileged,
								"runAsNonRoot":             runAsNonRoot,
								"allowPrivilegeEscalation": allowPrivilegeEscalation,
								"readOnlyRootFilesystem":   readOnlyRootFilesystem,
							},
						},
					},
				},
			},
		},
	}
}
