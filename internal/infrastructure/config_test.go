package infrastructure_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func readFixture(t *testing.T, relPath string) string {
	t.Helper()
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("locate project root failed: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		t.Fatalf("read %s failed: %v", relPath, err)
	}
	return string(contents)
}

func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func assertContains(t *testing.T, contents, needle, file string) {
	t.Helper()
	if !strings.Contains(contents, needle) {
		t.Fatalf("%s missing %q", file, needle)
	}
}

func parseYAML(t *testing.T, relPath string) *yaml.Node {
	t.Helper()
	contents := readFixture(t, relPath)
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(contents), &doc); err != nil {
		t.Fatalf("unmarshal %s failed: %v", relPath, err)
	}
	if len(doc.Content) == 0 {
		t.Fatalf("%s has empty yaml document", relPath)
	}
	return doc.Content[0]
}

func mappingValue(t *testing.T, node *yaml.Node, key string) *yaml.Node {
	t.Helper()
	if node == nil || node.Kind != yaml.MappingNode {
		t.Fatalf("expected mapping node while reading key %q", key)
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		k := node.Content[i]
		v := node.Content[i+1]
		if k.Value == key {
			return v
		}
	}
	t.Fatalf("missing key %q", key)
	return nil
}

func sequenceHasScalar(node *yaml.Node, want string) bool {
	if node == nil || node.Kind != yaml.SequenceNode {
		return false
	}
	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode && item.Value == want {
			return true
		}
	}
	return false
}

func TestIAMServerlessOutputsAndResources(t *testing.T) {
	const relPath = "infrastructure/iam/serverless.yml"
	contents := readFixture(t, relPath)

	assertContains(t, contents, "service: rbac-dev-iam", relPath)
	assertContains(t, contents, "${file(./roles.yml)}", relPath)
	assertContains(t, contents, "${file(./policies.yml)}", relPath)
	assertContains(t, contents, "TaskExecutionRoleArn:", relPath)
	assertContains(t, contents, "TaskRoleArn:", relPath)
	assertContains(t, contents, "rbac-dev-iam-TaskExecutionRoleArn", relPath)
	assertContains(t, contents, "rbac-dev-iam-TaskRoleArn", relPath)
}

func TestECSUsesIAMOutputs(t *testing.T) {
	const relPath = "infrastructure/ecs/serverless.yml"
	contents := readFixture(t, relPath)

	assertContains(t, contents, "ExecutionRoleArn: ${cf:rbac-dev-iam.TaskExecutionRoleArn}", relPath)
	assertContains(t, contents, "TaskRoleArn: ${cf:rbac-dev-iam.TaskRoleArn}", relPath)
}

func TestComposeWiresIAMBeforeECS(t *testing.T) {
	const relPath = "infrastructure/serverless-compose.yml"
	contents := readFixture(t, relPath)

	assertContains(t, contents, "iam:\n    path: ./iam", relPath)
	assertContains(t, contents, "ecs:\n    path: ./ecs", relPath)
	assertContains(t, contents, "- iam", relPath)
}

func TestInfraYAMLStructureForIAMAndECSWiring(t *testing.T) {
	iamRoot := parseYAML(t, "infrastructure/iam/serverless.yml")
	if got := mappingValue(t, iamRoot, "service").Value; got != "rbac-dev-iam" {
		t.Fatalf("unexpected iam service name: %q", got)
	}

	iamResources := mappingValue(t, iamRoot, "resources")
	if !sequenceHasScalar(iamResources, "${file(./roles.yml)}") {
		t.Fatal("iam resources missing roles file include")
	}
	if !sequenceHasScalar(iamResources, "${file(./policies.yml)}") {
		t.Fatal("iam resources missing policies file include")
	}

	var outputs *yaml.Node
	for _, item := range iamResources.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		if v := mappingValue(t, item, "Outputs"); v != nil {
			outputs = v
			break
		}
	}
	if outputs == nil {
		t.Fatal("iam resources missing Outputs mapping")
	}
	taskExec := mappingValue(t, outputs, "TaskExecutionRoleArn")
	taskRole := mappingValue(t, outputs, "TaskRoleArn")
	if got := mappingValue(t, mappingValue(t, taskExec, "Export"), "Name").Value; got != "rbac-dev-iam-TaskExecutionRoleArn" {
		t.Fatalf("unexpected execution role export name: %q", got)
	}
	if got := mappingValue(t, mappingValue(t, taskRole, "Export"), "Name").Value; got != "rbac-dev-iam-TaskRoleArn" {
		t.Fatalf("unexpected task role export name: %q", got)
	}

	ecsRoot := parseYAML(t, "infrastructure/ecs/serverless.yml")
	ecsResources := mappingValue(t, mappingValue(t, ecsRoot, "resources"), "Resources")
	taskDef := mappingValue(t, ecsResources, "TaskDefinition")
	taskProps := mappingValue(t, taskDef, "Properties")
	if got := mappingValue(t, taskProps, "ExecutionRoleArn").Value; got != "${cf:rbac-dev-iam.TaskExecutionRoleArn}" {
		t.Fatalf("unexpected ecs execution role reference: %q", got)
	}
	if got := mappingValue(t, taskProps, "TaskRoleArn").Value; got != "${cf:rbac-dev-iam.TaskRoleArn}" {
		t.Fatalf("unexpected ecs task role reference: %q", got)
	}

	composeRoot := parseYAML(t, "infrastructure/serverless-compose.yml")
	services := mappingValue(t, composeRoot, "services")
	if got := mappingValue(t, mappingValue(t, services, "iam"), "path").Value; got != "./iam" {
		t.Fatalf("unexpected iam compose path: %q", got)
	}
	ecsDependsOn := mappingValue(t, mappingValue(t, services, "ecs"), "dependsOn")
	if !sequenceHasScalar(ecsDependsOn, "iam") {
		t.Fatal("ecs compose dependencies missing iam")
	}
}
