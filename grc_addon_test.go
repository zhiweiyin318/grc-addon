package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"open-cluster-management.io/addon-framework/pkg/addonfactory"
	"open-cluster-management.io/addon-framework/pkg/agent"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/yaml"
)

func newAgentAddon(t *testing.T) (agent.AgentAddon, error) {
	policyRegistrationOption := newRegistrationOption(nil, PolicyAddonName)
	agentAddon, err := addonfactory.NewAgentAddonFactory(PolicyAddonName, PolicyChartFS, PolicyChartDir).
		WithGetValuesFuncs(getValues, addonfactory.GetValuesFromAddonAnnotation).
		WithAgentRegistrationOption(policyRegistrationOption).
		BuildHelmAgentAddon()
	if err != nil {
		t.Errorf("failed to build agentAddon")
		return agentAddon, err
	}
	return agentAddon, nil
}
func newManagedCluster(clusterName string) *clusterv1.ManagedCluster {
	return &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec:   clusterv1.ManagedClusterSpec{},
		Status: clusterv1.ManagedClusterStatus{Version: clusterv1.ManagedClusterVersion{Kubernetes: "1.10.1"}},
	}
}
func newManagedClusterAddon(clusterName, installNamespace, values string) *addonapiv1alpha1.ManagedClusterAddOn {
	return &addonapiv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PolicyAddonName,
			Namespace: clusterName,
			Annotations: map[string]string{
				"addon.open-cluster-management.io/values": values,
			},
		},
		Spec: addonapiv1alpha1.ManagedClusterAddOnSpec{
			InstallNamespace: installNamespace,
		},
	}
}

func TestAddonAgentManifests(t *testing.T) {
	agentAddon, err := newAgentAddon(t)
	if err != nil {
		t.Fatalf("failed to new agentAddon %v", err)
	}
	cluster := newManagedCluster("test")
	annotaitonValues := `{"global":{"imagePullSecret":"mySecret","imageOverrides":{"helloWorldHelm":"quay.io/test:test"}}}`
	addon := newManagedClusterAddon("test", "myNs", annotaitonValues)
	objects, err := agentAddon.Manifests(cluster, addon)
	if err != nil {
		t.Fatalf("failed to get manifests %v", err)
	}

	tmpDir, err := os.MkdirTemp("./", "tmp_render")
	if err != nil {
		t.Fatalf("failed to create temp %v", err)
	}

	for _, o := range objects {
		data, err := yaml.Marshal(o)
		if err != nil {
			t.Fatalf("failed yaml marshal %v", err)
		}

		err = ioutil.WriteFile(fmt.Sprintf("%v/%v.yaml", tmpDir, o.GetObjectKind().GroupVersionKind().Kind), data, 0644)
		if err != nil {
			t.Fatalf("failed to Marshal object.%v", err)
		}

	}

}
