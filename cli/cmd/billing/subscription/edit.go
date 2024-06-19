package subscription

import (
	"github.com/rilldata/rill/cli/cmd/org"
	"github.com/rilldata/rill/cli/pkg/cmdutil"
	adminv1 "github.com/rilldata/rill/proto/gen/rill/admin/v1"
	"github.com/spf13/cobra"
)

func EditCmd(ch *cmdutil.Helper) *cobra.Command {
	var orgName, plan string

	editCmd := &cobra.Command{
		Use:   "edit",
		Args:  cobra.NoArgs,
		Short: "Edit organization subscription",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !cmd.Flags().Changed("org") && ch.Interactive {
				orgNames, err := org.OrgNames(ctx, ch)
				if err != nil {
					return err
				}

				orgName, err = cmdutil.SelectPrompt("Select org to change plan", orgNames, ch.Org)
				if err != nil {
					return err
				}
			}

			subsResp, err := client.ListOrganizationSubscriptions(ctx, &adminv1.ListOrganizationSubscriptionsRequest{
				OrgName: orgName,
			})
			if err != nil {
				return err
			}

			if len(subsResp.Subscriptions) == 0 {
				ch.PrintfWarn("No subscriptions found for organization %q\n", orgName)
				return nil
			}

			ch.PrintfBold("Organization has following subscription(s)\n")
			ch.PrintSubscriptions(subsResp.Subscriptions)

			ok, err := cmdutil.ConfirmPrompt("\nPlan changes will take place immediately, Do you want to Continue ?\n", "", false)
			if err != nil {
				return err
			}
			if !ok {
				ch.PrintfWarn("Aborted\n")
				return nil
			}

			resp, err := client.UpdateOrganizationBillingPlan(cmd.Context(), &adminv1.UpdateOrganizationBillingPlanRequest{
				OrgName:      orgName,
				RillPlanId:   &plan,
				BillerPlanId: &plan,
			})
			if err != nil {
				return err
			}

			ch.PrintfSuccess("Successfully subscribed to plan %q for org %q\n", plan, orgName)
			ch.PrintSubscriptions(resp.Subscriptions)
			return nil
		},
	}
	editCmd.Flags().SortFlags = false
	editCmd.Flags().StringVar(&orgName, "org", ch.Org, "Organization name")
	editCmd.Flags().StringVar(&plan, "plan", "", "Plan ID to change subscription to")

	return editCmd
}
