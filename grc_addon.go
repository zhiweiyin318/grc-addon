package main

import (
	"embed"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"open-cluster-management.io/addon-framework/pkg/addonfactory"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/utils"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

//go:embed manifests
//go:embed manifests/charts/cert-policy-controller
//go:embed manifests/charts/cert-policy-controller/templates/_helpers.tpl
var CertChartFS embed.FS

//go:embed manifests
//go:embed manifests/charts/iam-policy-controller
//go:embed manifests/charts/iam-policy-controller/templates/_helpers.tpl
var IamChartFS embed.FS

//go:embed manifests
//go:embed manifests/charts/policy
//go:embed manifests/charts/policy/templates/_helpers.tpl
var PolicyChartFS embed.FS

const (
	CertChartDir   = "manifests/charts/cert-policy-controller"
	IamChartDir    = "manifests/charts/iam-policy-controller"
	PolicyChartDir = "manifests/charts/policy"
)

func newRegistrationOption(kubeConfig *rest.Config, addonName string) *agent.RegistrationOption {
	return &agent.RegistrationOption{
		CSRConfigurations: agent.KubeClientSignerConfigurations(addonName, addonName),
		CSRApproveCheck:   utils.DefaultCSRApprover(addonName),
		PermissionConfig: func(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) error {
			// update the permission of hub for each addon agent here if the addon needs to access the hub.
			// the permission will be given into a kubeConfig secret named <addon-Name>-hub-hubeconfig which
			// the deployment mount on the managed cluster.

			return nil
		},
	}
}

// define all of fields in values.yaml here
//
type GlobalValues struct {
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// do not change these field names,
	// klusterlet-addon-controller will override these values according annotation since we have case that need to change these values.
	ImagePullSecret string            `json:"imagePullSecret"`
	ImageOverrides  map[string]string `json:"imageOverrides"`
	NodeSelector    map[string]string `json:"nodeSelector"`
	ProxyConfig     map[string]string `json:"proxyConfig"`
}

type Values struct {
	FullNameOverride string       `json:"fullnameOverride"`
	Global           GlobalValues `json:"global,omitempty"`
}

func getValues(cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) (addonfactory.Values, error) {
	jsonValues := Values{
		Global: GlobalValues{
			ImagePullPolicy: "IfNotPresent",
			// the default image pull secert is "open-cluster-management-image-pull-credentials",
			//
			ImagePullSecret: "open-cluster-management-image-pull-credentials",
			ImageOverrides: map[string]string{
				// images can get from the cmd options or env
				// or other place
				"governance_policy_spec_sync":       "quay.io/open-cluster-management/governance-policy-spec-sync:latest-dev",
				"governance_policy_status_sync":     "quay.io/open-cluster-management/governance-policy-status-sync:latest-dev",
				"governance_policy_template_sync":   "quay.io/open-cluster-management/governance-policy-template-sync:latest-dev",
				"config_policy_controller":          "quay.io/open-cluster-management/config-policy-controller:latest-dev",
				"klusterlet_addon_lease_controller": "quay.io/open-cluster-management/klusterlet-addon-lease-controller:2.2.0",
			},
			ProxyConfig: map[string]string{
				"HTTP_PROXY":  "",
				"HTTPS_PROXY": "",
				"NO_PROXY":    "",
			},
		},
	}
	values, err := addonfactory.JsonStructToValues(jsonValues)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func getUserValues(cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) (addonfactory.Values, error) {

	// can get the user values from any place.
	// for example, get the user values from a new annotation of addon cr
	userVales := addon.Annotations["user-defined"]
	values := addonfactory.Values{}
	json.Unmarshal([]byte(userVales), &values)

	return values, nil
}
