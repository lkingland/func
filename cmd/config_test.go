package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/ory/viper"
	"github.com/spf13/cobra"
	fnCmd "knative.dev/func/cmd"
	"knative.dev/func/cmd/common"
	"knative.dev/func/pkg/ci/github"
	fn "knative.dev/func/pkg/functions"
)

func TestListEnvs(t *testing.T) {
	mock := common.NewMockLoaderSaver()
	foo := "foo"
	bar := "bar"
	envs := []fn.Env{{Name: &foo, Value: &bar}}
	mock.LoadFn = func(path string) (fn.Function, error) {
		if path != "<path>" {
			t.Fatalf("bad path, got %q but expected <path>", path)
		}
		return fn.Function{Run: fn.RunSpec{Envs: envs}}, nil
	}

	cmd := setupConfigEnvCmd(mock, "-o=json", "--path=<path>")

	var buff bytes.Buffer
	cmd.SetOut(&buff)
	cmd.SetErr(&buff)

	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	var data []fn.Env
	err = json.Unmarshal(buff.Bytes(), &data)
	if err != nil {
		t.Fatal(err)
	}
	if !envsEqual(envs, data) {
		t.Errorf("env mismatch, expected %v but got %v", envs, data)
	}
}

func setupConfigCmd(mock common.FunctionLoaderSaver, sub string, args ...string) *cobra.Command {
	cmd := fnCmd.NewConfigCmd(
		mock,
		github.NewBufferWriter(),
		common.CurrentBranchStub("", nil),
		common.WorkDirStub("", nil),
		fnCmd.NewTestCIGeneratorFactory(&github.WorkflowGeneratorMock{}),
		fnCmd.NewClient,
	)
	cmd.SetArgs(append([]string{sub}, args...))
	return cmd
}

func setupConfigEnvCmd(mock common.FunctionLoaderSaver, args ...string) *cobra.Command {
	return setupConfigCmd(mock, "envs", args...)
}

func TestListEnvAdd(t *testing.T) {
	// strings as vars so we can take address of them
	foo := "foo"
	bar := "bar"
	answer := "answer"
	fortyTwo := "42"
	configMapExpression := "{{ configMap:myMap }}"

	mock := common.NewMockLoaderSaver()
	mock.LoadFn = func(path string) (fn.Function, error) {
		return fn.Function{Run: fn.RunSpec{Envs: []fn.Env{{Name: &foo, Value: &bar}}}}, nil
	}
	var expectedEnvs []fn.Env
	mock.SaveFn = func(f fn.Function) error {
		if !envsEqual(expectedEnvs, f.Run.Envs) {
			return fmt.Errorf("unexpected envs: got %v but %v was expected", f.Run.Envs, expectedEnvs)
		}
		return nil
	}

	expectedEnvs = []fn.Env{{Name: &foo, Value: &bar}, {Name: &answer, Value: &fortyTwo}}
	cmd := setupConfigEnvCmd(mock, "add", "--name=answer", "--value=42")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}

	viper.Reset()
	expectedEnvs = []fn.Env{{Name: &foo, Value: &bar}, {Name: nil, Value: &configMapExpression}}
	cmd = setupConfigEnvCmd(mock, "add", "--value={{ configMap:myMap }}")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err = cmd.Execute()
	if err != nil {
		t.Error(err)
	}

	viper.Reset()
	cmd = setupConfigEnvCmd(mock, "add", "--name=1", "--value=abc")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err = cmd.Execute()
	if err == nil {
		t.Error("expected variable name error but got nil")
	}
}

// TestEnvsAddNameWithoutValue ensures that providing --name without --value returns an error
// rather than silently writing a broken fn.Env{Name: &key, Value: nil} to func.yaml.
func TestEnvsAddNameWithoutValue(t *testing.T) {
	mock := common.NewMockLoaderSaver()
	mock.LoadFn = func(path string) (fn.Function, error) {
		return fn.Function{}, nil
	}
	mock.SaveFn = func(f fn.Function) error {
		t.Error("Save must not be called when validation fails")
		return nil
	}

	cmd := setupConfigEnvCmd(mock, "add", "--name=API_KEY")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --name is provided without --value, got nil")
	}
	if !strings.Contains(err.Error(), "--value is required") {
		t.Fatalf("expected the CLI missing-value message, got: %v", err)
	}
}

// TestLabelsAddNameWithoutValue ensures that providing --name without --value returns an error.
func TestLabelsAddNameWithoutValue(t *testing.T) {
	mock := common.NewMockLoaderSaver()
	mock.LoadFn = func(path string) (fn.Function, error) {
		return fn.Function{}, nil
	}
	mock.SaveFn = func(f fn.Function) error {
		t.Error("Save must not be called when validation fails")
		return nil
	}

	cmd := setupConfigCmd(mock, "labels", "add", "--name=env")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --name is provided without --value, got nil")
	}
}

// TestLabelsAddValueWithoutName ensures that providing --value without --name returns an error.
func TestLabelsAddValueWithoutName(t *testing.T) {
	mock := common.NewMockLoaderSaver()
	mock.LoadFn = func(path string) (fn.Function, error) {
		return fn.Function{}, nil
	}
	mock.SaveFn = func(f fn.Function) error {
		t.Error("Save must not be called when validation fails")
		return nil
	}

	cmd := setupConfigCmd(mock, "labels", "add", "--value=prod")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --value is provided without --name, got nil")
	}
}

// TestEnvsAddBothFlagsSaves checks that --name and --value together persist.
// The partial-flag tests only cover the error path.
func TestEnvsAddBothFlagsSaves(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	mock := common.NewMockLoaderSaver()
	mock.LoadFn = func(path string) (fn.Function, error) {
		return fn.Function{}, nil
	}
	saved := false
	mock.SaveFn = func(f fn.Function) error {
		saved = true
		if len(f.Run.Envs) != 1 || f.Run.Envs[0].Name == nil || *f.Run.Envs[0].Name != "API_KEY" ||
			f.Run.Envs[0].Value == nil || *f.Run.Envs[0].Value != "secret" {
			t.Errorf("Save got envs %#v", f.Run.Envs)
		}
		return nil
	}

	cmd := setupConfigEnvCmd(mock, "add", "--name=API_KEY", "--value=secret")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !saved {
		t.Fatal("Save was not called")
	}
}

// TestEnvsAddPlainValueWithoutName ensures a bare value is not saved.
// A nil name is valid only for a whole secret or configMap reference.
func TestEnvsAddPlainValueWithoutName(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	mock := common.NewMockLoaderSaver()
	mock.LoadFn = func(path string) (fn.Function, error) {
		return fn.Function{}, nil
	}
	mock.SaveFn = func(f fn.Function) error {
		t.Error("Save must not be called for a plain value with no name")
		return nil
	}

	cmd := setupConfigEnvCmd(mock, "add", "--value=plain")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when --value is a plain string without --name")
	}
}

func TestEnvsAddWholeSecretWithoutNameSaves(t *testing.T) {
	assertEnvValueWithoutNameSaves(t, "{{ secret:S }}")
}

func TestEnvsAddWholeConfigMapWithoutNameSaves(t *testing.T) {
	assertEnvValueWithoutNameSaves(t, "{{ configMap:C }}")
}

func assertEnvValueWithoutNameSaves(t *testing.T, value string) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)

	mock := common.NewMockLoaderSaver()
	mock.LoadFn = func(path string) (fn.Function, error) {
		return fn.Function{}, nil
	}
	saved := false
	mock.SaveFn = func(f fn.Function) error {
		saved = true
		if len(f.Run.Envs) != 1 || f.Run.Envs[0].Name != nil || f.Run.Envs[0].Value == nil || *f.Run.Envs[0].Value != value {
			t.Errorf("Save got envs %#v", f.Run.Envs)
		}
		return nil
	}

	cmd := setupConfigEnvCmd(mock, "add", "--value="+value)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !saved {
		t.Fatal("Save was not called")
	}
}

// TestLabelsAddBothFlagsSaves checks that --name and --value together persist.
// Those checks for a missing flag run only after the both-flags path returns.
func TestLabelsAddBothFlagsSaves(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	mock := common.NewMockLoaderSaver()
	mock.LoadFn = func(path string) (fn.Function, error) {
		return fn.Function{}, nil
	}
	saved := false
	mock.SaveFn = func(f fn.Function) error {
		saved = true
		if len(f.Deploy.Labels) != 1 || f.Deploy.Labels[0].Key == nil || *f.Deploy.Labels[0].Key != "env" ||
			f.Deploy.Labels[0].Value == nil || *f.Deploy.Labels[0].Value != "prod" {
			t.Errorf("Save got labels %#v", f.Deploy.Labels)
		}
		return nil
	}

	cmd := setupConfigCmd(mock, "labels", "add", "--name=env", "--value=prod")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !saved {
		t.Fatal("Save was not called")
	}
}

func envsEqual(a, b []fn.Env) bool {
	if len(a) != len(b) {
		return false
	}

	strPtrEq := func(x, y *string) bool {
		switch {
		case x == nil && y == nil:
			return true
		case x != nil && y != nil:
			return *x == *y
		default:
			return false
		}
	}

	strPtrLess := func(x, y *string) bool {
		switch {
		case x == nil && y == nil:
			return false
		case x != nil && y != nil:
			return *x < *y
		case x == nil:
			return true
		default:
			return false
		}

	}

	lessForSlice := func(s []fn.Env) func(i, j int) bool {
		return func(i, j int) bool {
			x := s[i]
			y := s[j]
			if strPtrLess(x.Name, y.Name) {
				return true
			}
			return strPtrLess(x.Value, y.Value)
		}
	}

	sort.Slice(a, lessForSlice(a))
	sort.Slice(b, lessForSlice(b))

	for i := range a {
		x := a[i]
		y := b[i]
		if !strPtrEq(x.Name, y.Name) || !strPtrEq(x.Value, y.Value) {
			return false
		}
	}
	return true
}
