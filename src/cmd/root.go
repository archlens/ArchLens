package cmd

import (
	"os"

	"github.com/archlens/ArchLens/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "archlens",
	Short: "ArchLens enables you to create architectural views of your codebase",
	Long:  `ArchLens enables you to create architectural views of your codebase`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Global logger for ease of use
var Sugar *zap.SugaredLogger

var Version = "dev"

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	v, _ := rootCmd.Flags().GetBool("verbose")
	s, _ := rootCmd.Flags().GetBool("verbose")

	Sugar = utils.NewPrettySugaredLogger(v, s)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output (mut-ex with silence)")
	rootCmd.PersistentFlags().BoolP("silence", "s", false, "Enable silence output (mut-ex with verbose)")
	rootCmd.MarkFlagsMutuallyExclusive("verbose", "silence")
}
