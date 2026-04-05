package config

// CurrentVersion is the latest config schema version.
const CurrentVersion = 3

// Migrate applies forward migrations to bring a config to CurrentVersion.
// Returns a list of human-readable changes that were applied.
func Migrate(cfg *Config) []string {
	var changes []string

	if cfg.Version < 1 {
		if cfg.AnalysisCmd == "" {
			defaultCmd := "go vet ./..." // backward compat for pre-ecosystem configs.
			if mod := GetEcosystem(cfg.Ecosystem); mod != nil {
				defaultCmd = mod.DefaultAnalysisCmd
			}

			cfg.AnalysisCmd = defaultCmd
			changes = append(changes, "set analysis_command to '"+defaultCmd+"'")
		}

		if cfg.TemplateOver == "" {
			cfg.TemplateOver = ".promptkit/templates"

			changes = append(changes, "set template_overrides to '.promptkit/templates'")
		}

		cfg.Version = 1
	}

	if cfg.Version < 2 {
		if cfg.Workflow == "" {
			cfg.Workflow = WorkflowFRD

			changes = append(changes, "set workflow to 'frd'")
		}

		cfg.Version = 2
	}

	if cfg.Version < CurrentVersion {
		// v2 → v3: mixtures field added (optional, defaults to empty).
		cfg.Version = CurrentVersion
	}

	return changes
}
