package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/mfojtik/cluster-up/cmd/cluster/up"
	"github.com/mfojtik/cluster-up/pkg/log"
	"github.com/spf13/cobra"
)

const ClusterCommandName = "cluster"

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	command := NewClusterCommand()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

}

func NewClusterCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   ClusterCommandName,
		Short: "Minimal OpenShift cluster bootstrap tool",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	rootCmd.PersistentFlags().IntVar(&log.LogLevel, "loglevel", 3, "Sets the logging verbosity")

	upCommand := up.NewClusterUpCommand(up.RecommendedClusterUpName, ClusterCommandName, os.Stdout, os.Stderr)
	rootCmd.AddCommand(upCommand)

	return rootCmd
}
