package config

import (
	"testing"
)

func TestEcoRegistry_AllEcosystemsRegistered(t *testing.T) {
	want := []string{"golang", "rust", "zig"}
	got := ValidEcosystemNames()

	if len(got) != len(want) {
		t.Fatalf("ValidEcosystemNames() = %v, want %v", got, want)
	}

	for i, name := range got {
		if name != want[i] {
			t.Errorf("ValidEcosystemNames()[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestEcoRegistry_GetReturnsModule(t *testing.T) {
	for _, name := range []string{"golang", "rust", "zig"} {
		mod := GetEcosystem(name)
		if mod == nil {
			t.Errorf("GetEcosystem(%q) returned nil", name)
			continue
		}

		if mod.Name != name {
			t.Errorf("GetEcosystem(%q).Name = %q", name, mod.Name)
		}
	}
}

func TestEcoRegistry_GetUnknownReturnsNil(t *testing.T) {
	if mod := GetEcosystem("python"); mod != nil {
		t.Errorf("GetEcosystem(\"python\") = %v, want nil", mod)
	}
}

func TestEcoRegistry_IsValid(t *testing.T) {
	for _, name := range []string{"golang", "rust", "zig"} {
		if !ValidEcosystem(name) {
			t.Errorf("ValidEcosystem(%q) = false, want true", name)
		}
	}

	if ValidEcosystem("python") {
		t.Error("ValidEcosystem(\"python\") = true, want false")
	}
}

func TestEcoRegistry_Descriptions(t *testing.T) {
	descs := EcosystemDescriptions()

	for _, name := range ValidEcosystemNames() {
		desc, ok := descs[name]
		if !ok {
			t.Errorf("EcosystemDescriptions() missing key %q", name)
			continue
		}

		if desc == "" {
			t.Errorf("EcosystemDescriptions()[%q] is empty", name)
		}
	}
}

func TestEcoRegistry_AllModulesHaveRequiredFields(t *testing.T) {
	for _, mod := range AllEcosystems() {
		if mod.Description == "" {
			t.Errorf("module %q: Description is empty", mod.Name)
		}

		if mod.DefaultAnalysisCmd == "" {
			t.Errorf("module %q: DefaultAnalysisCmd is empty", mod.Name)
		}

		if mod.DefaultCmdPath == nil {
			t.Errorf("module %q: DefaultCmdPath is nil", mod.Name)
		}

		if mod.ApplyDefaults == nil {
			t.Errorf("module %q: ApplyDefaults is nil", mod.Name)
		}

		if mod.Validate == nil {
			t.Errorf("module %q: Validate is nil", mod.Name)
		}

		if mod.RegisterFlags == nil {
			t.Errorf("module %q: RegisterFlags is nil", mod.Name)
		}

		if mod.ApplyFlags == nil {
			t.Errorf("module %q: ApplyFlags is nil", mod.Name)
		}

		if mod.RunPrompts == nil {
			t.Errorf("module %q: RunPrompts is nil", mod.Name)
		}

		if mod.CommentedFields == nil {
			t.Errorf("module %q: CommentedFields is nil", mod.Name)
		}

		if len(mod.FieldEntries) == 0 {
			t.Errorf("module %q: FieldEntries is empty", mod.Name)
		}
	}
}

func TestClosestEcosystem(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"golang", "golang"},
		{"golng", "golang"},
		{"rustt", "rust"},
		{"GOLANG", "golang"},
		{"python", ""},
		{"", ""},
		{"zi", "zig"},
	}

	for _, tt := range tests {
		got := ClosestEcosystem(tt.input)
		if got != tt.want {
			t.Errorf("ClosestEcosystem(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Golang module tests ---

func TestGolang_DefaultCmdPath(t *testing.T) {
	mod := GetEcosystem("golang")

	got := mod.DefaultCmdPath("myapp")
	if got != "./cmd/myapp" {
		t.Errorf("golang.DefaultCmdPath(\"myapp\") = %q, want \"./cmd/myapp\"", got)
	}
}

func TestGolang_ApplyDefaults(t *testing.T) {
	cfg := &Config{}
	mod := GetEcosystem("golang")
	mod.ApplyDefaults(cfg)

	if cfg.GoVersion == "" {
		t.Error("golang.ApplyDefaults: GoVersion is empty")
	}

	if cfg.AnalysisCmd != "go vet ./..." {
		t.Errorf("golang.ApplyDefaults: AnalysisCmd = %q", cfg.AnalysisCmd)
	}
}

func TestGolang_Validate_RequiresGoVersion(t *testing.T) {
	cfg := &Config{GoVersion: ""}
	errs := GetEcosystem("golang").Validate(cfg)

	if len(errs) != 1 {
		t.Fatalf("golang.Validate: got %d errors, want 1", len(errs))
	}

	if errs[0].Error() != "go_version is required for golang ecosystem (e.g. \"1.22\")" {
		t.Errorf("golang.Validate: error = %q", errs[0])
	}
}

func TestGolang_Validate_Valid(t *testing.T) {
	cfg := &Config{GoVersion: "1.22"}
	errs := GetEcosystem("golang").Validate(cfg)

	if len(errs) != 0 {
		t.Errorf("golang.Validate: got %d errors, want 0: %v", len(errs), errs)
	}
}

func TestGolang_CommentedFields(t *testing.T) {
	cfg := &Config{GoVersion: "1.22"}
	fields := GetEcosystem("golang").CommentedFields(cfg)

	if len(fields) != 1 {
		t.Fatalf("golang.CommentedFields: got %d fields, want 1", len(fields))
	}

	if fields[0].Key != "go_version" {
		t.Errorf("golang.CommentedFields[0].Key = %q, want \"go_version\"", fields[0].Key)
	}

	if !fields[0].Quote {
		t.Error("golang.CommentedFields[0].Quote = false, want true")
	}
}

// --- Rust module tests ---

func TestRust_DefaultCmdPath(t *testing.T) {
	got := GetEcosystem("rust").DefaultCmdPath("myapp")
	if got != "src/main.rs" {
		t.Errorf("rust.DefaultCmdPath(\"myapp\") = %q, want \"src/main.rs\"", got)
	}
}

func TestRust_ApplyDefaults(t *testing.T) {
	cfg := &Config{GoVersion: "1.22"}
	GetEcosystem("rust").ApplyDefaults(cfg)

	if cfg.GoVersion != "" {
		t.Errorf("rust.ApplyDefaults: GoVersion = %q, want empty", cfg.GoVersion)
	}

	if cfg.RustEdition == "" {
		t.Error("rust.ApplyDefaults: RustEdition is empty")
	}

	if cfg.UnsafePolicy == "" {
		t.Error("rust.ApplyDefaults: UnsafePolicy is empty")
	}

	if cfg.AnalysisCmd != "cargo clippy -- -D warnings" {
		t.Errorf("rust.ApplyDefaults: AnalysisCmd = %q", cfg.AnalysisCmd)
	}
}

func TestRust_Validate_RequiresEdition(t *testing.T) {
	cfg := &Config{RustEdition: "", UnsafePolicy: "deny"}
	errs := GetEcosystem("rust").Validate(cfg)

	if len(errs) == 0 {
		t.Fatal("rust.Validate: expected error for empty RustEdition")
	}
}

func TestRust_Validate_InvalidUnsafePolicy(t *testing.T) {
	cfg := &Config{RustEdition: "2021", UnsafePolicy: "yolo"}
	errs := GetEcosystem("rust").Validate(cfg)

	if len(errs) == 0 {
		t.Fatal("rust.Validate: expected error for invalid UnsafePolicy")
	}
}

func TestRust_Validate_Valid(t *testing.T) {
	cfg := &Config{RustEdition: "2021", UnsafePolicy: "deny"}
	errs := GetEcosystem("rust").Validate(cfg)

	if len(errs) != 0 {
		t.Errorf("rust.Validate: got %d errors: %v", len(errs), errs)
	}
}

func TestRust_Validate_EmptyUnsafePolicyAllowed(t *testing.T) {
	cfg := &Config{RustEdition: "2021", UnsafePolicy: ""}
	errs := GetEcosystem("rust").Validate(cfg)

	if len(errs) != 0 {
		t.Errorf("rust.Validate: got %d errors for empty UnsafePolicy: %v", len(errs), errs)
	}
}

func TestRust_CommentedFields(t *testing.T) {
	cfg := &Config{RustEdition: "2021", UnsafePolicy: "deny"}
	fields := GetEcosystem("rust").CommentedFields(cfg)

	if len(fields) != 2 {
		t.Fatalf("rust.CommentedFields: got %d fields, want 2", len(fields))
	}

	if fields[0].Key != "rust_edition" {
		t.Errorf("fields[0].Key = %q, want \"rust_edition\"", fields[0].Key)
	}

	if fields[1].Key != "unsafe_policy" {
		t.Errorf("fields[1].Key = %q, want \"unsafe_policy\"", fields[1].Key)
	}
}

// --- Zig module tests ---

func TestZig_DefaultCmdPath(t *testing.T) {
	got := GetEcosystem("zig").DefaultCmdPath("myapp")
	if got != "src/main.zig" {
		t.Errorf("zig.DefaultCmdPath(\"myapp\") = %q, want \"src/main.zig\"", got)
	}
}

func TestZig_ApplyDefaults(t *testing.T) {
	cfg := &Config{GoVersion: "1.22"}
	GetEcosystem("zig").ApplyDefaults(cfg)

	if cfg.GoVersion != "" {
		t.Errorf("zig.ApplyDefaults: GoVersion = %q, want empty", cfg.GoVersion)
	}

	if cfg.ZigVersion == "" {
		t.Error("zig.ApplyDefaults: ZigVersion is empty")
	}

	if cfg.AnalysisCmd != "zig build test" {
		t.Errorf("zig.ApplyDefaults: AnalysisCmd = %q", cfg.AnalysisCmd)
	}
}

func TestZig_Validate_AlwaysValid(t *testing.T) {
	cfg := &Config{}
	errs := GetEcosystem("zig").Validate(cfg)

	if len(errs) != 0 {
		t.Errorf("zig.Validate: got %d errors: %v", len(errs), errs)
	}
}

func TestZig_CommentedFields(t *testing.T) {
	cfg := &Config{ZigVersion: "0.13", LinkLibc: true}
	fields := GetEcosystem("zig").CommentedFields(cfg)

	if len(fields) != 2 {
		t.Fatalf("zig.CommentedFields: got %d fields, want 2", len(fields))
	}

	if fields[0].Key != "zig_version" {
		t.Errorf("fields[0].Key = %q, want \"zig_version\"", fields[0].Key)
	}

	if fields[1].Key != "link_libc" {
		t.Errorf("fields[1].Key = %q, want \"link_libc\"", fields[1].Key)
	}
}
