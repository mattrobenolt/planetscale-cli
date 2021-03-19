package branch

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func SwitchCmd(cfg *config.Config) *cobra.Command {
	var parentBranch string
	var autoCreate bool

	cmd := &cobra.Command{
		Use:   "switch <branch>",
		Short: "Switches the current project to use the specified branch",
		Args:  cmdutil.RequiredArgs("branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			branch := args[0]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			fmt.Printf("Finding branch %s on database %s\n", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(cfg.Database))

			_, err = client.DatabaseBranches.Get(ctx, &planetscale.GetDatabaseBranchRequest{
				Organization: cfg.Organization,
				Database:     cfg.Database,
				Branch:       branch,
			})
			if err != nil && !errorIsNotFound(err) {
				return err
			}

			if errorIsNotFound(err) {
				if !autoCreate {
					return errors.New("branch does not exist in specified database. Use --create to automatically create during switch")
				}

				end := cmdutil.PrintProgress(fmt.Sprintf("Branch does not exist, creating %s branch from %s...", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(parentBranch)))
				defer end()

				createReq := &ps.CreateDatabaseBranchRequest{
					Organization: cfg.Organization,
					Database:     cfg.Database,
					Branch: &ps.DatabaseBranch{
						Name:         branch,
						ParentBranch: parentBranch,
					},
				}

				_, err = client.DatabaseBranches.Create(ctx, createReq)
				if err != nil {
					return err
				}

				end()
			}

			cfg := config.WritableProjectConfig{
				Database: cfg.Database,
				Branch:   branch,
			}

			if err := cfg.WriteDefault(); err != nil {
				return errors.Wrap(err, "error writing project configuration file")
			}

			fmt.Printf("Successfully switched to branch %s on database %s", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(parentBranch))

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&cfg.Database, "database", cfg.Database, "The database this project is using")
	cmd.Flags().StringVar(&parentBranch, "parent-branch", "main", "parent branch to inherit from if a new branch is being created")
	cmd.Flags().BoolVar(&autoCreate, "create", false, "if enabled, will automatically create the branch if it does not exist")

	cmd.MarkPersistentFlagRequired("database") // nolint:errcheck
	return cmd
}

func errorIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == http.StatusText(http.StatusNotFound)
}
