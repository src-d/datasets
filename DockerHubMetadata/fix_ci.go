package main

import (
	"context"
	"io"
	
	"github.com/containers/image/types"
	"github.com/urfave/cli"
)

type globalOptions struct{}

func (global *globalOptions) commandTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

type imageOptions struct{}

func imageFlags(global *globalOptions, shared interface{}, a, b string) ([]cli.Flag, *imageOptions) {
	return nil, &imageOptions{}
}

func sharedImageFlags() ([]cli.Flag, interface{}) {
	return nil, nil
}

func commandAction(handler func(args []string, stdout io.Writer) error) cli.ActionFunc {
	return func(c *cli.Context) error { return nil }
}

func reexecIfNecessaryForImages(s... string) error {
    return nil
}

func parseImage(ctx context.Context, imageOpts *imageOptions, imageName string) (types.ImageCloser, error) {
    return nil, nil
}

func main() {}
