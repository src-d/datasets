package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/containers/image/transports"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type inspectOptions struct {
	global *globalOptions
	image  *imageOptions
	raw    bool // Output the raw manifest instead of parsing information about the image
	config bool // Output the raw config blob instead of parsing information about the image
}

func inspectCmd(global *globalOptions) cli.Command {
	sharedFlags, sharedOpts := sharedImageFlags()
	imageFlags, imageOpts := imageFlags(global, sharedOpts, "", "")
	opts := inspectOptions{
		global: global,
		image:  imageOpts,
	}
	return cli.Command{
		Name:  "inspect",
		Usage: "Inspect image IMAGE-NAME",
		Description: fmt.Sprintf(`
    Return low-level information about "IMAGE-NAME" in a registry/transport

    Supported transports:
    %s

    See skopeo(1) section "IMAGE NAMES" for the expected format
    `, strings.Join(transports.ListNames(), ", ")),
		ArgsUsage: "IMAGE-NAME",
		Flags: append(append([]cli.Flag{
			cli.BoolFlag{
				Name:        "raw",
				Usage:       "output raw manifest or configuration",
				Destination: &opts.raw,
			},
			cli.BoolFlag{
				Name:        "config",
				Usage:       "output configuration",
				Destination: &opts.config,
			},
		}, sharedFlags...), imageFlags...),
		Action: commandAction(opts.run),
	}
}

const batchSize = 1024
const maxAttempts = 50

func errorIsRecoverable(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Temporary failure in name resolution") ||
		strings.Contains(err.Error(), "too many open files") ||
		strings.Contains(err.Error(), "TLS handshake timeout"))
}

func (opts *inspectOptions) run(args []string, stdout io.Writer) error {
	ctx, cancel := opts.global.commandTimeoutContext()
	defer cancel()

	var imageNames []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		imageNames = append(imageNames, "docker://docker.io/"+scanner.Text())
	}

	for batch := 0; batch < len(imageNames)/batchSize+1; batch++ {
		log.Println(">>> batch", batch+1)
		endIndex := (batch + 1) * batchSize
		if endIndex > len(imageNames) {
			endIndex = len(imageNames)
		}
		imageBatch := imageNames[batch*batchSize : endIndex]

		if err := reexecIfNecessaryForImages(imageBatch...); err != nil {
			return err
		}
		wg := &sync.WaitGroup{}
		wg.Add(len(imageBatch))
		for _, imageName := range imageBatch {
			go func(imageName string) (imageErr error) {
				defer wg.Done()

				fileNameSeed := imageName[len("docker://docker.io/"):]
				fileNameManifest := filepath.Join("manifests", fileNameSeed[:2], fileNameSeed+".json")
				fileNameConfig := filepath.Join("configs", fileNameSeed[:2], fileNameSeed+".json")
				if _, err := os.Stat(fileNameManifest); err == nil {
					if _, err := os.Stat(fileNameConfig); err == nil {
						return nil
					}
				}

				img, err := parseImage(ctx, opts.image, imageName)
				defer func() {
					if img != nil {
						if err := img.Close(); err != nil {
							imageErr = errors.Wrapf(imageErr, fmt.Sprintf("(could not close image: %v) ", err))
						}
					}
					if imageErr != nil {
						log.Println("error!", imageName, imageErr)
					}
				}()
				if err != nil {
					imageErr = err
					return
				}

				var rawManifest []byte
				for i := 0; i < maxAttempts; i++ {
					rawManifest, _, err = img.Manifest(ctx)
					if errorIsRecoverable(err) {
						time.Sleep(50 * time.Millisecond)
						continue
					}
					break
				}
				if err != nil {
					imageErr = err
					return
				}
				os.MkdirAll(filepath.Dir(fileNameManifest), os.ModePerm)
				manfile, err := os.Create(fileNameManifest)
				if err != nil {
					imageErr = fmt.Errorf("Error creating %s: %v", fileNameManifest, err)
					return
				}
				defer manfile.Close()
				_, err = manfile.Write(rawManifest)
				if err != nil {
					imageErr = fmt.Errorf("Error writing manifest to %s: %v", fileNameManifest, err)
					return
				}

				var configBlob []byte
				for i := 0; i < maxAttempts; i++ {
					configBlob, err = img.ConfigBlob(ctx)
					if errorIsRecoverable(err) {
						time.Sleep(50 * time.Millisecond)
						continue
					}
					break
				}
				if err != nil {
					imageErr = fmt.Errorf("Error reading configuration blob: %v", err)
					return
				}

				os.MkdirAll(filepath.Dir(fileNameConfig), os.ModePerm)
				cfgfile, err := os.Create(fileNameConfig)
				if err != nil {
					imageErr = fmt.Errorf("Error creating %s: %v", fileNameConfig, err)
					return
				}
				defer cfgfile.Close()
				_, err = cfgfile.Write(configBlob)
				if err != nil {
					imageErr = fmt.Errorf("Error writing configuration blob to %s: %v", fileNameConfig, err)
					return
				}

				log.Println(fileNameSeed)
				return nil
			}(imageName)
		}
		wg.Wait()
	}
	return nil
}
