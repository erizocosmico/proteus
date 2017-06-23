package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/src-d/proteus.v1"
	"gopkg.in/src-d/proteus.v1/protobuf"
	"gopkg.in/src-d/proteus.v1/report"

	"gopkg.in/urfave/cli.v1"
)

var (
	packages cli.StringSlice
	path     string
	verbose  bool
)

func main() {
	app := cli.NewApp()
	app.Name = "proteus"
	app.Description = "Proteus generates code and protobuffer 3 proto files while keeping your Go source code as the source of truth."
	app.Version = "1.0.0"

	baseFlags := []cli.Flag{
		cli.StringSliceFlag{
			Name:  "pkg, p",
			Usage: "Use `PACKAGE` as input for the generation. You can use this flag multiple times to specify more than one package.",
			Value: &packages,
		},
		cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Print all warnings and info messages.",
			Destination: &verbose,
		},
	}

	folderFlag := cli.StringFlag{
		Name:        "folder, f",
		Usage:       "All generated .proto files will be written to `FOLDER`.",
		Destination: &path,
	}

	app.Flags = append(baseFlags, folderFlag)
	app.Commands = []cli.Command{
		{
			Name:        "proto",
			Description: "Generates .proto files from your Go source code.",
			Usage:       "Generates .proto files from Go packages",
			Action:      initCmd(genProtos),
			Flags:       append(baseFlags, folderFlag),
		},
		{
			Name:        "rpc",
			Description: "Generates the gRPC implementation of the gRPC server interface defined by your Go source code.",
			Usage:       "Generates gRPC server implementation",
			Action:      initCmd(genRPCServer),
			Flags:       baseFlags,
		},
	}
	app.Action = initCmd(genAll)

	app.Run(os.Args)
}

type action func(c *cli.Context) error

func initCmd(next action) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		if len(packages) == 0 {
			return errors.New("no package provided, there is nothing to generate")
		}

		if !verbose {
			report.Silent()
		}

		return next(c)
	}
}

func genProtos(c *cli.Context) error {
	if path == "" {
		return errors.New("destination path cannot be empty")
	}

	if err := checkFolder(path); err != nil {
		return err
	}

	return proteus.GenerateProtos(proteus.Options{
		BasePath: path,
		Packages: packages,
	})
}

func genRPCServer(c *cli.Context) error {
	return proteus.GenerateRPCServer(packages)
}

var (
	goSrc       = filepath.Join(os.Getenv("GOPATH"), "src")
	protobufSrc = filepath.Join(goSrc, "github.com", "gogo", "protobuf")
	grpcGateway = filepath.Join(goSrc, "github.com", "grpc-ecosystem", "grpc-gateway", "third_party", "googleapis")
)

func genAll(c *cli.Context) error {
	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		return fmt.Errorf("protoc is not installed: %s", err)
	}

	if err := checkFolder(protobufSrc); err != nil {
		return fmt.Errorf("github.com/gogo/protobuf is not installed")
	}

	if err := genProtos(c); err != nil {
		return err
	}

	for _, p := range packages {
		outPath := goSrc
		proto := filepath.Join(path, p, "generated.proto")

		if err := protocExec(protocPath, p, outPath, proto); err != nil {
			return fmt.Errorf("error generating Go files from %q: %s", proto, err)
		}

		dest := filepath.Join(outPath, p)
		err := mvGlob(
			filepath.Join(path, p, "*.pb.go"),
			dest,
		)
		if err != nil {
			return err
		}

		if err := grpcGatewayExec(protocPath, p, outPath, proto); err != nil {
			return fmt.Errorf("error generating GRPC Gateway from %q: %s", proto, err)
		}

		err = mvGlob(
			filepath.Join(path, p, "*pb.gw.go"),
			dest,
		)
		if err != nil {
			return err
		}
	}

	return genRPCServer(c)
}

func mvGlob(glob, dest string) error {
	matches, err := filepath.Glob(glob)
	if err != nil {
		return fmt.Errorf("error moving Go files: %s", err)
	}

	for _, s := range matches {
		if err := mv(s, dest); err != nil {
			return err
		}
	}

	return nil
}

func protocExec(protocPath, pkg, outPath, protoFile string) error {
	protocArgs := fmt.Sprintf(
		"--proto_path=%s:%s:%s:%s:%s:.",
		goSrc,
		path,
		grpcGateway,
		filepath.Join(protobufSrc, "protobuf"),
		filepath.Join(path, pkg),
	)

	report.Info("executing protoc: %s %s", protocPath, protocArgs)

	cmd := exec.Command(
		protocPath,
		protocArgs,
		genAllGoFastOutOption(outPath),
		protoFile,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func grpcGatewayExec(protocPath, pkg, outPath, protoFile string) error {
	protocArgs := fmt.Sprintf(
		"--proto_path=%s:%s:%s:%s:%s:.",
		goSrc,
		path,
		grpcGateway,
		filepath.Join(protobufSrc, "protobuf"),
		filepath.Join(path, pkg),
	)

	report.Info("generating grpc gateway: %s %s", protocPath, protocArgs)

	cmd := exec.Command(
		protocPath,
		protocArgs,
		genAllGRPCGatewayOutOption(outPath),
		protoFile,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func mv(from, to string) error {
	cmd := exec.Command("mv", from, to)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func genAllGoFastOutOption(outPath string) string {
	return optionWithImportsAndPath("--gofast_out=plugins=grpc", outPath)
}

func genAllGRPCGatewayOutOption(outPath string) string {
	return optionWithImportsAndPath("--grpc-gateway_out=logtostderr=true", outPath)
}

func optionWithImportsAndPath(opt string, outPath string) string {
	importMappings := protobuf.DefaultMappings.ToGoOutPath()

	if importMappings != "" {
		opt += fmt.Sprintf(",%s", importMappings)
	}

	opt += fmt.Sprintf(":%s", outPath)

	return opt
}

func checkFolder(p string) error {
	fi, err := os.Stat(p)
	switch {
	case os.IsNotExist(err):
		return errors.New("folder does not exist, please create it first")
	case err != nil:
		return err
	case !fi.IsDir():
		return fmt.Errorf("folder is not directory: %s", p)
	}
	return nil
}
