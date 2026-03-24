package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestEcoRegistry_AllEcosystemsRegistered(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	if mod := GetEcosystem("python"); mod != nil {
		t.Errorf("GetEcosystem(\"python\") = %v, want nil", mod)
	}
}

func TestEcoRegistry_IsValid(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

// Golang module tests.

func TestGolang_DefaultCmdPath(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("golang")

	got := mod.DefaultCmdPath("myapp")
	if got != "./cmd/myapp" {
		t.Errorf("golang.DefaultCmdPath(\"myapp\") = %q, want \"./cmd/myapp\"", got)
	}
}

func TestGolang_ApplyDefaults(t *testing.T) {
	t.Parallel()

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

func TestGolang_RequiredFields(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("golang")
	if !mod.Requires("module_path") {
		t.Error("golang should require module_path")
	}

	if !mod.Requires("go_version") {
		t.Error("golang should require go_version")
	}

	if mod.Requires("description") {
		t.Error("golang should not require description")
	}
}

func TestGolang_Validate_MissingGoVersion(t *testing.T) {
	t.Parallel()

	cfg := Default()
	cfg.ProjectName = "test"
	cfg.ModulePath = "github.com/test/project"
	cfg.GoVersion = ""
	cfg.Binaries = []Binary{{Name: "test", CmdPath: "./cmd/test"}}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing go_version")
	}

	if !strings.Contains(err.Error(), "go_version is required") {
		t.Errorf("error should mention go_version, got: %s", err)
	}
}

func TestGolang_Validate_Valid(t *testing.T) {
	t.Parallel()

	cfg := Default()
	cfg.ProjectName = "test"
	cfg.ModulePath = "github.com/test/project"
	cfg.GoVersion = "1.22"
	cfg.Binaries = []Binary{{Name: "test", CmdPath: "./cmd/test"}}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestGolang_CommentedFields(t *testing.T) {
	t.Parallel()

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

// Rust module tests.

func TestRust_DefaultCmdPath(t *testing.T) {
	t.Parallel()

	got := GetEcosystem("rust").DefaultCmdPath("myapp")
	if got != "src/main.rs" {
		t.Errorf("rust.DefaultCmdPath(\"myapp\") = %q, want \"src/main.rs\"", got)
	}
}

func TestRust_ApplyDefaults(t *testing.T) {
	t.Parallel()

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

func TestRust_RequiredFields(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("rust")
	if !mod.Requires("rust_edition") {
		t.Error("rust should require rust_edition")
	}
}

func TestRust_Validate_RequiresEdition(t *testing.T) {
	t.Parallel()

	cfg := Default()
	cfg.Ecosystem = "rust"
	cfg.ProjectName = "test"
	cfg.RustEdition = ""
	cfg.UnsafePolicy = "deny"
	cfg.Binaries = []Binary{{Name: "test", CmdPath: "src/main.rs"}}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty RustEdition")
	}

	if !strings.Contains(err.Error(), "rust_edition is required") {
		t.Errorf("error should mention rust_edition, got: %s", err)
	}
}

func TestRust_Validate_InvalidUnsafePolicy(t *testing.T) {
	t.Parallel()

	cfg := &Config{RustEdition: "2021", UnsafePolicy: "yolo"}
	errs := GetEcosystem("rust").Validate(cfg)

	if len(errs) == 0 {
		t.Fatal("rust.Validate: expected error for invalid UnsafePolicy")
	}
}

func TestRust_Validate_Valid(t *testing.T) {
	t.Parallel()

	cfg := &Config{RustEdition: "2021", UnsafePolicy: "deny"}
	errs := GetEcosystem("rust").Validate(cfg)

	if len(errs) != 0 {
		t.Errorf("rust.Validate: got %d errors: %v", len(errs), errs)
	}
}

func TestRust_Validate_EmptyUnsafePolicyAllowed(t *testing.T) {
	t.Parallel()

	cfg := &Config{RustEdition: "2021", UnsafePolicy: ""}
	errs := GetEcosystem("rust").Validate(cfg)

	if len(errs) != 0 {
		t.Errorf("rust.Validate: got %d errors for empty UnsafePolicy: %v", len(errs), errs)
	}
}

func TestRust_CommentedFields(t *testing.T) {
	t.Parallel()

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

// Zig module tests.

func TestZig_DefaultCmdPath(t *testing.T) {
	t.Parallel()

	got := GetEcosystem("zig").DefaultCmdPath("myapp")
	if got != "src/main.zig" {
		t.Errorf("zig.DefaultCmdPath(\"myapp\") = %q, want \"src/main.zig\"", got)
	}
}

func TestZig_ApplyDefaults(t *testing.T) {
	t.Parallel()

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

func TestZig_RequiredFields(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("zig")
	if !mod.Requires("zig_version") {
		t.Error("zig should require zig_version")
	}
}

func TestZig_Validate_AlwaysValid(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	errs := GetEcosystem("zig").Validate(cfg)

	if len(errs) != 0 {
		t.Errorf("zig.Validate: got %d errors: %v", len(errs), errs)
	}
}

func TestZig_CommentedFields(t *testing.T) {
	t.Parallel()

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

// Golang RegisterFlags / ApplyFlags / RunPrompts tests.

func TestGolang_RegisterFlags(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("golang")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	f := cmd.Flags().Lookup("go-version")
	if f == nil {
		t.Fatal("golang.RegisterFlags: --go-version flag not registered")
	}

	if f.DefValue != "" {
		t.Errorf("golang --go-version default = %q, want empty", f.DefValue)
	}
}

func TestGolang_ApplyFlags(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("golang")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	// Set the flag value.
	if err := cmd.Flags().Set("go-version", "1.23"); err != nil {
		t.Fatalf("setting go-version flag: %v", err)
	}

	cfg := &Config{}
	mod.ApplyFlags(cfg)

	if cfg.GoVersion != "1.23" {
		t.Errorf("golang.ApplyFlags: GoVersion = %q, want %q", cfg.GoVersion, "1.23")
	}
}

func TestGolang_ApplyFlags_NoFlag(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("golang")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	// Do not set any flag - ApplyFlags should not change GoVersion.
	cfg := &Config{GoVersion: "1.21"}
	mod.ApplyFlags(cfg)

	if cfg.GoVersion != "1.21" {
		t.Errorf("golang.ApplyFlags without flag: GoVersion = %q, want %q", cfg.GoVersion, "1.21")
	}
}

func TestGolang_RunPrompts(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("golang")
	cfg := &Config{}

	askIdx := 0
	askResponses := []string{"1.23"}
	askFn := func(_, defaultVal string) (string, error) {
		if askIdx >= len(askResponses) {
			return defaultVal, nil
		}

		resp := askResponses[askIdx]
		askIdx++

		return resp, nil
	}

	boolIdx := 0
	boolResponses := []bool{true}
	askBoolFn := func(_ string, defaultVal bool) (bool, error) {
		if boolIdx >= len(boolResponses) {
			return defaultVal, nil
		}

		resp := boolResponses[boolIdx]
		boolIdx++

		return resp, nil
	}

	if err := mod.RunPrompts(cfg, askFn, askBoolFn); err != nil {
		t.Fatalf("golang.RunPrompts: %v", err)
	}

	if cfg.GoVersion != "1.23" {
		t.Errorf("golang.RunPrompts: GoVersion = %q, want %q", cfg.GoVersion, "1.23")
	}

	if !cfg.Features.CGO {
		t.Error("golang.RunPrompts: Features.CGO = false, want true")
	}
}

func TestGolang_RunPrompts_AskError(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("golang")
	cfg := &Config{}

	askFn := func(_, _ string) (string, error) {
		return "", errors.New("prompt canceled")
	}

	askBoolFn := func(_ string, _ bool) (bool, error) {
		return false, nil
	}

	err := mod.RunPrompts(cfg, askFn, askBoolFn)
	if err == nil {
		t.Fatal("golang.RunPrompts should return error when askFn fails")
	}
}

func TestGolang_RunPrompts_AskBoolError(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("golang")
	cfg := &Config{}

	askFn := func(_, defaultVal string) (string, error) {
		return defaultVal, nil
	}

	askBoolFn := func(_ string, _ bool) (bool, error) {
		return false, errors.New("prompt canceled")
	}

	err := mod.RunPrompts(cfg, askFn, askBoolFn)
	if err == nil {
		t.Fatal("golang.RunPrompts should return error when askBoolFn fails")
	}
}

// Golang ApplyDefaults preserves existing GoVersion.

func TestGolang_ApplyDefaults_PreservesExistingVersion(t *testing.T) {
	t.Parallel()

	cfg := &Config{GoVersion: "1.23"}
	mod := GetEcosystem("golang")
	mod.ApplyDefaults(cfg)

	if cfg.GoVersion != "1.23" {
		t.Errorf("golang.ApplyDefaults should preserve existing GoVersion, got %q", cfg.GoVersion)
	}
}

// Rust RegisterFlags / ApplyFlags / RunPrompts tests.

func TestRust_RegisterFlags(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("rust")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	edition := cmd.Flags().Lookup("rust-edition")
	if edition == nil {
		t.Fatal("rust.RegisterFlags: --rust-edition flag not registered")
	}

	policy := cmd.Flags().Lookup("unsafe-policy")
	if policy == nil {
		t.Fatal("rust.RegisterFlags: --unsafe-policy flag not registered")
	}
}

func TestRust_ApplyFlags(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("rust")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	if err := cmd.Flags().Set("rust-edition", "2024"); err != nil {
		t.Fatalf("setting rust-edition flag: %v", err)
	}

	if err := cmd.Flags().Set("unsafe-policy", "forbid"); err != nil {
		t.Fatalf("setting unsafe-policy flag: %v", err)
	}

	cfg := &Config{GoVersion: "1.22"}
	mod.ApplyFlags(cfg)

	if cfg.GoVersion != "" {
		t.Errorf("rust.ApplyFlags: GoVersion = %q, want empty", cfg.GoVersion)
	}

	if cfg.RustEdition != "2024" {
		t.Errorf("rust.ApplyFlags: RustEdition = %q, want %q", cfg.RustEdition, "2024")
	}

	if cfg.UnsafePolicy != "forbid" {
		t.Errorf("rust.ApplyFlags: UnsafePolicy = %q, want %q", cfg.UnsafePolicy, "forbid")
	}
}

func TestRust_ApplyFlags_Defaults(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("rust")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	// Do not set flags - should apply defaults.
	cfg := &Config{}
	mod.ApplyFlags(cfg)

	if cfg.RustEdition != "2021" {
		t.Errorf("rust.ApplyFlags default: RustEdition = %q, want %q", cfg.RustEdition, "2021")
	}

	if cfg.UnsafePolicy != "deny" {
		t.Errorf("rust.ApplyFlags default: UnsafePolicy = %q, want %q", cfg.UnsafePolicy, "deny")
	}
}

func TestRust_ApplyFlags_PreservesExisting(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("rust")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	// Do not set flags, but config already has values.
	cfg := &Config{RustEdition: "2024", UnsafePolicy: "forbid"}
	mod.ApplyFlags(cfg)

	if cfg.RustEdition != "2024" {
		t.Errorf("rust.ApplyFlags should preserve existing RustEdition, got %q", cfg.RustEdition)
	}

	if cfg.UnsafePolicy != "forbid" {
		t.Errorf("rust.ApplyFlags should preserve existing UnsafePolicy, got %q", cfg.UnsafePolicy)
	}
}

func TestRust_RunPrompts(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("rust")
	cfg := &Config{}

	askIdx := 0
	askResponses := []string{"2024", "forbid"}
	askFn := func(_, _ string) (string, error) {
		if askIdx >= len(askResponses) {
			return "", errors.New("unexpected prompt call")
		}

		resp := askResponses[askIdx]
		askIdx++

		return resp, nil
	}

	askBoolFn := func(_ string, _ bool) (bool, error) {
		return false, nil
	}

	if err := mod.RunPrompts(cfg, askFn, askBoolFn); err != nil {
		t.Fatalf("rust.RunPrompts: %v", err)
	}

	if cfg.RustEdition != "2024" {
		t.Errorf("rust.RunPrompts: RustEdition = %q, want %q", cfg.RustEdition, "2024")
	}

	if cfg.UnsafePolicy != "forbid" {
		t.Errorf("rust.RunPrompts: UnsafePolicy = %q, want %q", cfg.UnsafePolicy, "forbid")
	}
}

func TestRust_RunPrompts_EditionError(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("rust")
	cfg := &Config{}

	askFn := func(_, _ string) (string, error) {
		return "", errors.New("prompt canceled")
	}

	askBoolFn := func(_ string, _ bool) (bool, error) {
		return false, nil
	}

	err := mod.RunPrompts(cfg, askFn, askBoolFn)
	if err == nil {
		t.Fatal("rust.RunPrompts should return error when first askFn fails")
	}
}

func TestRust_RunPrompts_PolicyError(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("rust")
	cfg := &Config{}

	callCount := 0
	askFn := func(_, _ string) (string, error) {
		callCount++
		if callCount == 1 {
			return "2021", nil
		}

		return "", errors.New("prompt canceled")
	}

	askBoolFn := func(_ string, _ bool) (bool, error) {
		return false, nil
	}

	err := mod.RunPrompts(cfg, askFn, askBoolFn)
	if err == nil {
		t.Fatal("rust.RunPrompts should return error when second askFn fails")
	}
}

// Rust ApplyDefaults preserves existing values.

func TestRust_ApplyDefaults_PreservesExisting(t *testing.T) {
	t.Parallel()

	cfg := &Config{RustEdition: "2024", UnsafePolicy: "forbid"}
	GetEcosystem("rust").ApplyDefaults(cfg)

	if cfg.RustEdition != "2024" {
		t.Errorf("rust.ApplyDefaults should preserve existing RustEdition, got %q", cfg.RustEdition)
	}

	if cfg.UnsafePolicy != "forbid" {
		t.Errorf("rust.ApplyDefaults should preserve existing UnsafePolicy, got %q", cfg.UnsafePolicy)
	}
}

// Zig RegisterFlags / ApplyFlags / RunPrompts tests.

func TestZig_RegisterFlags(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("zig")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	ver := cmd.Flags().Lookup("zig-version")
	if ver == nil {
		t.Fatal("zig.RegisterFlags: --zig-version flag not registered")
	}

	libc := cmd.Flags().Lookup("link-libc")
	if libc == nil {
		t.Fatal("zig.RegisterFlags: --link-libc flag not registered")
	}
}

func TestZig_ApplyFlags(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("zig")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	if err := cmd.Flags().Set("zig-version", "0.14"); err != nil {
		t.Fatalf("setting zig-version flag: %v", err)
	}

	if err := cmd.Flags().Set("link-libc", "true"); err != nil {
		t.Fatalf("setting link-libc flag: %v", err)
	}

	cfg := &Config{GoVersion: "1.22"}
	mod.ApplyFlags(cfg)

	if cfg.GoVersion != "" {
		t.Errorf("zig.ApplyFlags: GoVersion = %q, want empty", cfg.GoVersion)
	}

	if cfg.ZigVersion != "0.14" {
		t.Errorf("zig.ApplyFlags: ZigVersion = %q, want %q", cfg.ZigVersion, "0.14")
	}

	if !cfg.LinkLibc {
		t.Error("zig.ApplyFlags: LinkLibc = false, want true")
	}
}

func TestZig_ApplyFlags_Defaults(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("zig")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	// Do not set flags - should apply defaults.
	cfg := &Config{}
	mod.ApplyFlags(cfg)

	if cfg.ZigVersion != "0.13" {
		t.Errorf("zig.ApplyFlags default: ZigVersion = %q, want %q", cfg.ZigVersion, "0.13")
	}
}

func TestZig_ApplyFlags_PreservesExisting(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("zig")
	cmd := &cobra.Command{Use: "test"}
	mod.RegisterFlags(cmd)

	// Do not set flags, but config already has values.
	cfg := &Config{ZigVersion: "0.14"}
	mod.ApplyFlags(cfg)

	if cfg.ZigVersion != "0.14" {
		t.Errorf("zig.ApplyFlags should preserve existing ZigVersion, got %q", cfg.ZigVersion)
	}
}

func TestZig_RunPrompts(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("zig")
	cfg := &Config{}

	askFn := func(_, _ string) (string, error) {
		return "0.14", nil
	}

	askBoolFn := func(_ string, _ bool) (bool, error) {
		return true, nil
	}

	if err := mod.RunPrompts(cfg, askFn, askBoolFn); err != nil {
		t.Fatalf("zig.RunPrompts: %v", err)
	}

	if cfg.ZigVersion != "0.14" {
		t.Errorf("zig.RunPrompts: ZigVersion = %q, want %q", cfg.ZigVersion, "0.14")
	}

	if !cfg.LinkLibc {
		t.Error("zig.RunPrompts: LinkLibc = false, want true")
	}
}

func TestZig_RunPrompts_AskError(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("zig")
	cfg := &Config{}

	askFn := func(_, _ string) (string, error) {
		return "", errors.New("prompt canceled")
	}

	askBoolFn := func(_ string, _ bool) (bool, error) {
		return false, nil
	}

	err := mod.RunPrompts(cfg, askFn, askBoolFn)
	if err == nil {
		t.Fatal("zig.RunPrompts should return error when askFn fails")
	}
}

func TestZig_RunPrompts_AskBoolError(t *testing.T) {
	t.Parallel()

	mod := GetEcosystem("zig")
	cfg := &Config{}

	askFn := func(_, defaultVal string) (string, error) {
		return defaultVal, nil
	}

	askBoolFn := func(_ string, _ bool) (bool, error) {
		return false, errors.New("prompt canceled")
	}

	err := mod.RunPrompts(cfg, askFn, askBoolFn)
	if err == nil {
		t.Fatal("zig.RunPrompts should return error when askBoolFn fails")
	}
}

// Zig ApplyDefaults preserves existing version.

func TestZig_ApplyDefaults_PreservesExistingVersion(t *testing.T) {
	t.Parallel()

	cfg := &Config{ZigVersion: "0.14"}
	GetEcosystem("zig").ApplyDefaults(cfg)

	if cfg.ZigVersion != "0.14" {
		t.Errorf("zig.ApplyDefaults should preserve existing ZigVersion, got %q", cfg.ZigVersion)
	}
}

// Test closestWorkflow directly (internal package).

func TestClosestWorkflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"frd", "frd"},
		{"journey", "journey"},
		{"fRd", "frd"},
		{"journy", "journey"},
		{"journe", "journey"},
		{"zzzzzzzzz", ""},
	}

	for _, tt := range tests {
		got := closestWorkflow(tt.input)
		if got != tt.want {
			t.Errorf("closestWorkflow(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// Test quoteVersion directly (internal package).

func TestQuoteVersionDirect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"", `""`},
		{"1.21", `"1.21"`},
		{"2021", `"2021"`},
		{"0.13", `"0.13"`},
	}

	for _, tt := range tests {
		got := quoteVersion(tt.input)
		if got != tt.want {
			t.Errorf("quoteVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// Test Rust CommentedFields values and Quote flags.

func TestRust_CommentedFields_Values(t *testing.T) {
	t.Parallel()

	cfg := &Config{RustEdition: "2024", UnsafePolicy: "forbid"}
	fields := GetEcosystem("rust").CommentedFields(cfg)

	if fields[0].Value != "2024" {
		t.Errorf("rust_edition value = %v, want \"2024\"", fields[0].Value)
	}

	if !fields[0].Quote {
		t.Error("rust_edition should have Quote=true")
	}

	if fields[1].Value != "forbid" {
		t.Errorf("unsafe_policy value = %v, want \"forbid\"", fields[1].Value)
	}

	if fields[1].Quote {
		t.Error("unsafe_policy should have Quote=false")
	}
}

// Test Zig CommentedFields values and Quote flags.

func TestZig_CommentedFields_Values(t *testing.T) {
	t.Parallel()

	cfg := &Config{ZigVersion: "0.14", LinkLibc: false}
	fields := GetEcosystem("zig").CommentedFields(cfg)

	if fields[0].Value != "0.14" {
		t.Errorf("zig_version value = %v, want \"0.14\"", fields[0].Value)
	}

	if !fields[0].Quote {
		t.Error("zig_version should have Quote=true")
	}

	boolVal, ok := fields[1].Value.(bool)
	if !ok || boolVal {
		t.Errorf("link_libc value = %v, want false", fields[1].Value)
	}

	if fields[1].Quote {
		t.Error("link_libc should have Quote=false")
	}
}
