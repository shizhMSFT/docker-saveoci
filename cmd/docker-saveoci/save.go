package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/shizhMSFT/docker-saveoci/internal/convert"
	"github.com/spf13/cobra"
)

type saveOCIOpts struct {
	images []string
	output string
}

func saveOCICommand(opts *saveOCIOpts) *cobra.Command {
	if opts == nil {
		opts = &saveOCIOpts{}
	}
	cmd := &cobra.Command{
		Use:   "saveoci [OPTIONS] IMAGE [IMAGE...]",
		Short: "Save one or more images to a tar archive in the OCI layout",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.images = args
			return runSaveOCI(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Write to a file")
	cmd.MarkFlagRequired("output")
	return cmd
}

func runSaveOCI(ctx context.Context, opts *saveOCIOpts) error {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"save"}, opts.images...)...)
	cmd.Stderr = os.Stderr
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer rc.Close()
	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Wait()

	wc, err := os.Create(opts.output)
	if err != nil {
		return err
	}
	defer wc.Close()

	return convert.DockerToOCI(rc, wc)
}
