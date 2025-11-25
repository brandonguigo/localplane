package create

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// askCreateConfirmation prompts the user to confirm cluster creation unless
// the --yes flag is provided. Returns true to proceed, false to abort.
func askCreateConfirmation(cmd *cobra.Command, clusterName string) bool {
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Printf("Proceed to create kind cluster '%s'? (y/N): ", clusterName)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if !(strings.EqualFold(input, "y") || strings.EqualFold(input, "yes")) {
			log.Info().Msg("aborting cluster creation")
			return false
		}
	}
	return true
}
