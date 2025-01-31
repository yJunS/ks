package component

import (
	"context"
	"fmt"
	"github.com/linuxsuren/ks/kubectl-plugin/common"
	kstypes "github.com/linuxsuren/ks/kubectl-plugin/types"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"strconv"
)

// EnableOption is the option for component enable command
type EnableOption struct {
	Option

	Edit   bool
	Toggle bool
}

func getAvailableComponents() func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return common.ArrayCompletion("devops", "alerting", "auditing", "events", "logging", "metrics_server", "networkpolicy", "notification", "openpitrix", "servicemesh")
}

// NewComponentEnableCmd returns a command to enable (or disable) a component by name
func NewComponentEnableCmd(client dynamic.Interface) (cmd *cobra.Command) {
	opt := &EnableOption{
		Option: Option{
			Client: client,
		},
	}

	availableComs := getAvailableComponents()

	cmd = &cobra.Command{
		Use:               "enable",
		Short:             "Enable or disable the specific KubeSphere component",
		PreRunE:           opt.enablePreRunE,
		ValidArgsFunction: availableComs,
		RunE:              opt.enableRunE,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opt.Edit, "edit", "e", false,
		"Indicate if you want to edit it instead of enable/disable a specified one. This flag will make others not work.")
	flags.BoolVarP(&opt.Toggle, "toggle", "t", false,
		"Indicate if you want to disable a component")
	flags.StringVarP(&opt.Name, "name", "n", "",
		"The name of target component which you want to enable/disable. Please provide option --sonarqube if you want to enable SonarQube.")
	flags.StringVarP(&opt.SonarQube, "sonarqube", "", "",
		"The SonarQube URL")
	flags.StringVarP(&opt.SonarQube, "sonar", "", "",
		"The SonarQube URL")
	flags.StringVarP(&opt.SonarQubeToken, "sonarqube-token", "", "",
		"The token of SonarQube")

	_ = cmd.RegisterFlagCompletionFunc("name", availableComs)

	// these are aliased options
	_ = flags.MarkHidden("sonar")
	return
}

func (o *EnableOption) enablePreRunE(cmd *cobra.Command, args []string) (err error) {
	if o.Edit {
		return
	}

	return o.componentNameCheck(cmd, args)
}

func (o *EnableOption) enableRunE(cmd *cobra.Command, args []string) (err error) {
	if o.Edit {
		err = common.UpdateWithEditor(kstypes.GetClusterConfiguration(), "kubesphere-system", "ks-installer", o.Client)
	} else {
		enabled := strconv.FormatBool(!o.Toggle)
		ns, name := "kubesphere-system", "ks-installer"
		var patchTarget string
		switch o.Name {
		case "devops", "alerting", "auditing", "events", "logging", "metrics_server", "networkpolicy", "notification", "openpitrix", "servicemesh":
			patchTarget = o.Name
		case "sonarqube", "sonar":
			if o.SonarQube == "" || o.SonarQubeToken == "" {
				err = fmt.Errorf("SonarQube or token is empty, please provide --sonarqube")
			} else {
				name = "ks-console-config"
				err = integrateSonarQube(o.Client, ns, name, o.SonarQube, o.SonarQubeToken)
			}
			return
		default:
			err = fmt.Errorf("not support [%s] yet", o.Name)
			return
		}

		patch := fmt.Sprintf(`[{"op": "replace", "path": "/spec/%s/enabled", "value": %s}]`, patchTarget, enabled)
		ctx := context.TODO()
		_, err = o.Client.Resource(kstypes.GetClusterConfiguration()).Namespace(ns).Patch(ctx,
			name, types.JSONPatchType,
			[]byte(patch),
			metav1.PatchOptions{})
	}
	return
}
