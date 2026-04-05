package main

import "path/filepath"

func defaultWorkflowLocalConfigPath() string {
	return filepath.ToSlash(filepath.Join("research", "config", "workflow", "default-local.yaml"))
}

func workflowConfigPath(name string) string {
	return filepath.ToSlash(filepath.Join("research", "config", "workflow", name))
}

func directConfigPath(name string) string {
	return filepath.ToSlash(filepath.Join("research", "config", "direct", name))
}

func ndConfigPath(name string) string {
	return filepath.ToSlash(filepath.Join("research", "config", "nd", name))
}

func compositionalConfigPath(name string) string {
	return filepath.ToSlash(filepath.Join("research", "config", "compositional", name))
}

func bridgeConfigPath(name string) string {
	return filepath.ToSlash(filepath.Join("research", "config", "compositional", "bridge", name))
}

func goldConfigPath(name string) string {
	return filepath.ToSlash(filepath.Join("research", "config", "compositional", "gold", name))
}
