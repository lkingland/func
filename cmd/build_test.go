package cmd

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
	fn "knative.dev/kn-plugin-func"
	"knative.dev/kn-plugin-func/mock"
	. "knative.dev/kn-plugin-func/testing"
)

// TestBuild_InvalidRegistry ensures that running build specifying the name of the
// registry explicitly as an argument invokes the registry validation code.
func TestBuild_InvalidRegistry(t *testing.T) {
	root, rm := Mktemp(t)
	defer rm()

	f := fn.Function{
		Root:    root,
		Name:    "myFunc",
		Runtime: "go",
	}
	if err := fn.New().Create(f); err != nil {
		t.Fatal(err)
	}

	cmd := NewBuildCmd(NewClientFactory(func() *fn.Client {
		return fn.New()
	}))

	cmd.SetArgs([]string{"--registry=foo/bar/invald/myfunc"})

	if err := cmd.Execute(); err == nil {
		// TODO: typed ErrInvalidRegistry
		t.Fatal("invalid registry did not generate expected error")
	}
}

func TestBuild_runBuild(t *testing.T) {
	tests := []struct {
		name         string
		pushFlag     bool
		fileContents string
		shouldBuild  bool
		shouldPush   bool
		wantErr      bool
	}{
		{
			name:     "push flag triggers push after build",
			pushFlag: true,
			fileContents: `name: test-func
runtime: go
created: 2009-11-10 23:00:00`,
			shouldBuild: true,
			shouldPush:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPusher := mock.NewPusher()
			failPusher := &mock.Pusher{
				PushFn: func(f fn.Function) (string, error) {
					return "", fmt.Errorf("push failed")
				},
			}
			mockBuilder := mock.NewBuilder()
			cmd := NewBuildCmd(NewClientFactory(func() *fn.Client {
				pusher := mockPusher
				if tt.wantErr {
					pusher = failPusher
				}
				return fn.New(
					fn.WithBuilder(mockBuilder),
					fn.WithPusher(pusher),
				)
			}))

			tempDir, err := os.MkdirTemp("", "func-tests")
			if err != nil {
				t.Fatalf("temp dir couldn't be created %v", err)
			}
			t.Log("tempDir created:", tempDir)
			t.Cleanup(func() {
				os.RemoveAll(tempDir)
			})

			fullPath := tempDir + "/func.yaml"
			tempFile, err := os.Create(fullPath)
			if err != nil {
				t.Fatalf("temp file couldn't be created %v", err)
			}
			_, err = tempFile.WriteString(tt.fileContents)
			if err != nil {
				t.Fatalf("file content was not written %v", err)
			}

			cmd.SetArgs([]string{
				"--path=" + tempDir,
				fmt.Sprintf("--push=%t", tt.pushFlag),
				"--registry=docker.io/tigerteam",
			})

			err = cmd.Execute()
			if tt.wantErr != (err != nil) {
				t.Errorf("Wanted error %v but actually got %v", tt.wantErr, err)
			}

			if mockBuilder.BuildInvoked != tt.shouldBuild {
				t.Errorf("Build execution expected: %v but was actually %v", tt.shouldBuild, mockBuilder.BuildInvoked)
			}

			if tt.shouldPush != (mockPusher.PushInvoked || failPusher.PushInvoked) {
				t.Errorf("Push execution expected: %v but was actually mockPusher invoked: %v failPusher invoked %v", tt.shouldPush, mockPusher.PushInvoked, failPusher.PushInvoked)
			}
		})
	}
}

func testBuilderPersistence(t *testing.T, testRegistry string, cmdBuilder func(ClientFactory) *cobra.Command) {
	//add this to work with all other tests in deploy_test.go
	t.Setenv("KUBECONFIG", fmt.Sprintf("%s/testdata/kubeconfig_deploy_namespace", cwd()))

	root, rm := Mktemp(t)
	defer rm()

	client := fn.New(fn.WithRegistry(testRegistry))

	f := fn.Function{Runtime: "go", Root: root, Name: "myfunc", Registry: testRegistry}

	if err := client.New(context.Background(), f); err != nil {
		t.Fatal(err)
	}

	cmd := cmdBuilder(NewClientFactory(func() *fn.Client {
		return client
	}))

	cmd.SetArgs([]string{"--registry", testRegistry})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var err error
	// Assert the function has persisted a value of builder (has a default)
	if f, err = fn.NewFunction(root); err != nil {
		t.Fatal(err)
	}
	if f.Builder == "" {
		t.Fatal("value of builder not persisted using a flag default")
	}

	// Build the function, specifying a Builder
	cmd.SetArgs([]string{"--builder=s2i"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	// Assert the function has persisted the value of builder
	if f, err = fn.NewFunction(root); err != nil {
		t.Fatal(err)
	}
	if f.Builder != "s2i" {
		t.Fatal("value of builder flag not persisted when provided")
	}
	// Build the function without specifying a Builder
	cmd = cmdBuilder(NewClientFactory(func() *fn.Client {
		return client
	}))
	cmd.SetArgs([]string{"--registry", testRegistry})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Assert the function has retained its original value
	if f, err = fn.NewFunction(root); err != nil {
		t.Fatal(err)
	}

	if f.Builder != "s2i" {
		t.Fatal("value of builder updated when not provided")
	}

	// Build the function again using a different builder
	cmd.SetArgs([]string{"--builder=pack"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Assert the function has persisted the new value
	if f, err = fn.NewFunction(root); err != nil {
		t.Fatal(err)
	}
	if f.Builder != "pack" {
		t.Fatal("value of builder flag not persisted on subsequent build")
	}

	// Build the function, specifying a platform with "pack" Builder
	cmd.SetArgs([]string{"--platform", "linux"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("Expected error")
	}

	// Set an invalid builder
	cmd.SetArgs([]string{"--builder", "invalid"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("Expected error")
	}
}

// TestBuild_BuilderPersistence ensures that the builder chosen is read from
// the function by default, and is able to be overridden by flags/env vars.
func TestBuild_BuilderPersistence(t *testing.T) {
	testBuilderPersistence(t, "docker.io/tigerteam", NewBuildCmd)
}
