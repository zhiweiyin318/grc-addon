package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	goflag "flag"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/addonfactory"
	"open-cluster-management.io/addon-framework/pkg/version"

	"open-cluster-management.io/addon-framework/pkg/addonmanager"
)

const (
	PolicyAddonName = "policy-controller"
	IamAddonName    = "iam-policy-controller"
	CertAddonName   = "cert-policy-controller"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()

	command := newCommand()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addon",
		Short: "grc  addon",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			os.Exit(1)
		},
	}

	if v := version.Get().String(); len(v) == 0 {
		cmd.Version = "<unknown>"
	} else {
		cmd.Version = v
	}

	cmd.AddCommand(newControllerCommand())

	return cmd
}

func newControllerCommand() *cobra.Command {
	cmd := controllercmd.
		NewControllerCommandConfig("grc-addon-controller", version.Get(), runController).
		NewCommand()
	cmd.Use = "controller"
	cmd.Short = "Start the addon controller"

	return cmd
}

func runController(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	var mgr, err = addonmanager.New(controllerContext.KubeConfig)
	if err != nil {
		klog.Errorf("failed to new addon manager %v", err)
		return err
	}

	certRegistrationOption := newRegistrationOption(
		controllerContext.KubeConfig,
		CertAddonName,
	)
	iamRegistrationOption := newRegistrationOption(
		controllerContext.KubeConfig,
		IamAddonName,
	)

	policyRegistrationOption := newRegistrationOption(
		controllerContext.KubeConfig,
		PolicyAddonName,
	)

	// should add addonfactory.GetValuesFromAddonAnnotation for the WithGetValuesFuncs
	// klustelet-adon-controller will override images, proxy env by annotations .
	// you can define several GetValuesFunc, the values got from the big index Func will override the one from small index Func.
	// getUserValues overrides getValues,  addonfactory.GetValuesFromAddonAnnotation overrides getValues and getUserValues for example.
	certAgentAddon, err := addonfactory.NewAgentAddonFactory(CertAddonName, CertChartFS, CertChartDir).
		WithGetValuesFuncs(getValues, getUserValues, addonfactory.GetValuesFromAddonAnnotation).
		WithAgentRegistrationOption(certRegistrationOption).
		BuildHelmAgentAddon()
	if err != nil {
		klog.Errorf("failed to build agent %v", err)
		return err
	}

	iamAgentAddon, err := addonfactory.NewAgentAddonFactory(IamAddonName, IamChartFS, IamChartDir).
		WithGetValuesFuncs(getValues, getUserValues, addonfactory.GetValuesFromAddonAnnotation).
		WithAgentRegistrationOption(iamRegistrationOption).
		BuildHelmAgentAddon()
	if err != nil {
		klog.Errorf("failed to build agent %v", err)
		return err
	}

	policyAgentAddon, err := addonfactory.NewAgentAddonFactory(PolicyAddonName, PolicyChartFS, PolicyChartDir).
		WithGetValuesFuncs(getValues, getUserValues, addonfactory.GetValuesFromAddonAnnotation).
		WithAgentRegistrationOption(policyRegistrationOption).
		BuildHelmAgentAddon()
	if err != nil {
		klog.Errorf("failed to build agent %v", err)
		return err
	}

	err = mgr.AddAgent(certAgentAddon)
	if err != nil {
		klog.Fatal(err)
	}
	err = mgr.AddAgent(iamAgentAddon)
	if err != nil {
		klog.Fatal(err)
	}
	err = mgr.AddAgent(policyAgentAddon)
	if err != nil {
		klog.Fatal(err)
	}

	err = mgr.Start(ctx)
	if err != nil {
		klog.Fatal(err)
	}
	<-ctx.Done()

	return nil
}
