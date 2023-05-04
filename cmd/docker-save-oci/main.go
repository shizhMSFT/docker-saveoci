package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	cmd := &cobra.Command{
		Use:          "docker-save-oci",
		Short:        "A docker plugin to save one or more images to a tar archive in the OCI layout",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		metadataCommand(),
	)
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
