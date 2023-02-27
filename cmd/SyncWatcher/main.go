package main

import "github.com/spf13/cobra"

var RootCommand = &cobra.Command{
	Use: "SyncWatcher",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

var (
	paramConfig string
)

func init() {
	RootCommand.PersistentFlags().StringVarP(&paramConfig, "config", "c", "", "config file")
}

func main() {
	RootCommand.Execute()
}
