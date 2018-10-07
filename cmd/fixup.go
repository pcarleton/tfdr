package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/terraform/terraform"
	"github.com/pcarleton/tfdr/lib"
	"github.com/spf13/cobra"
)

var outState string

var fixupCmd = &cobra.Command{
	Use:   "fixup [planfile]",
	Short: "Match destroys with creates in a plan file",
	Long: `After performing a refactor on a terraform module, a plan will include
	a create and destroy for every moved resource.  This command matches the creates
	with the destroys by looking at their ID field and optionally writes out a new
	state file.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		planFileName := args[0]
		fh, err := os.Open(planFileName)
		if err != nil {
			fmt.Printf("Could not open file: %s\n", err)
			os.Exit(1)
		}
		plan, err := terraform.ReadPlan(fh)

		if err != nil {
			fmt.Printf("Could not read plan: %s\n", err)
			os.Exit(1)
		}

		candidates := lib.PickCandidates(plan)
		if len(candidates.Created) == 0 || len(candidates.Destroyed) == 0 {
			fmt.Println("Could not find any creates or destroys to match.")
			os.Exit(1)
		}

		pairs := lib.MatchPairs(plan, candidates)

		if len(pairs) == 0 {
			fmt.Println("Could not find any matches :(")
			os.Exit(1)
		}

		if outState == "" {
			lib.FmtError("Found the following pairs (pass them to 'terraform state mv' to modify the state):\n")
			for _, pair := range pairs {
				fmt.Printf("  %s %s\n", pair.Old.String(), pair.New.String())
			}
			return
		}

		// Write new state file out
		state := plan.State
		var moves []string
		for _, pair := range pairs {
			moves = append(moves, fmt.Sprintf("%s   ->   %s\n", pair.Old.String(), pair.New.String()))
			err = state.Add(pair.Old.String(), pair.New.String(), pair.State)
			if err != nil {
				fmt.Printf("Could do move %s -> %s: %s\n", err)
				os.Exit(1)
			}
		}

		// Get the resource state
		b, err := json.Marshal(state)
		if err != nil {
			fmt.Printf("Could not marshal state file: %s\n", err)
			os.Exit(1)
		}

		ioutil.WriteFile(outState, b, 0644)

		lib.FmtError("Wrote state file %s with the following moves:\n", outState)
		for _, move := range moves {
			lib.FmtError("\t%s\n", move)
		}
		lib.FmtError("Validate with `terraform plan -state=%s`", outState)
		lib.FmtError("Push to remote with `terraform state push %s`", outState)
	},
}

func init() {
	fixupCmd.Flags().StringVarP(&outState, "out", "o", "", "Path to write fixed up state file.")
	rootCmd.AddCommand(fixupCmd)
}
