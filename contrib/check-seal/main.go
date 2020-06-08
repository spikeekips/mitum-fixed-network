package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/contrib"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

var Version string = "v0.1-proto3"

type mainFlags struct {
	Version         bool   `help:"print version"`
	Input           string `arg:"" help:"seal input file; '-' is stdin" optional:""`
	NetworkIDString string ` name:"network-id" help:"network-id"`
	networkID       []byte
	Encoder         string `name:"encoder" help:"encoder type, {json, bson} default:json"`
	encoder         encoder.Encoder
	content         []byte
}

func (cmd *mainFlags) Run() error {
	var sl seal.Seal
	if hinter, err := cmd.encoder.DecodeByHint(cmd.content); err != nil {
		return err
	} else if i, ok := hinter.(seal.Seal); !ok {
		return hint.InvalidTypeError.Errorf("not seal.Seal; type=%T", i)
	} else {
		sl = i
	}

	if err := sl.IsValid(cmd.networkID); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "looks good")

	return nil
}

func main() {
	flags := &mainFlags{}
	ctx := kong.Parse(
		flags,
		kong.Name(os.Args[0]),
		kong.Description("mitum seal checker"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			NoAppSummary: false,
			Compact:      false,
			Summary:      true,
			Tree:         true,
		}),
		kong.Vars{},
	)

	if flags.Version {
		_, _ = fmt.Fprintln(os.Stdout, Version)

		os.Exit(0)
	}

	ctx.FatalIfErrorf(parse(flags))
	ctx.FatalIfErrorf(flags.Run())

	os.Exit(0)
}

func parse(flags *mainFlags) error {
	if s := strings.TrimSpace(flags.NetworkIDString); len(s) > 0 {
		flags.networkID = []byte(s)
	}

	var input io.Reader
	switch s := strings.TrimSpace(flags.Input); {
	case len(s) < 1:
		return xerrors.Errorf("missing input file")
	case s == "-":
		input = bufio.NewReader(os.Stdin)
	default:
		if i, err := os.Stat(s); err != nil {
			return err
		} else if i.IsDir() {
			return xerrors.Errorf("directory found")
		}

		if f, err := os.Open(filepath.Clean(s)); err != nil {
			return err
		} else {
			defer func() {
				_ = (interface{})(f).(io.ReadCloser).Close()
			}()

			input = f
		}
	}

	var content []byte
	reader := bufio.NewReader(input)
	b := make([]byte, 1024)
	for {
		n, err := reader.Read(b)
		content = append(content, b[:n]...)

		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}
	}
	flags.content = content

	if err := getEncoder(flags); err != nil {
		return err
	}

	return nil
}

func getEncoder(flags *mainFlags) error {
	if len(flags.Encoder) < 1 {
		flags.Encoder = "json"
	}

	var enc encoder.Encoder
	switch flags.Encoder {
	case "json":
		enc = jsonenc.NewEncoder()
	case "bson":
		enc = bsonenc.NewEncoder()
	default:
		return xerrors.Errorf("invalid encoder, %s", flags.Encoder)
	}

	if _, err := encoder.LoadEncoders([]encoder.Encoder{enc}, contrib.Hinters...); err != nil {
		return xerrors.Errorf("failed to load encoders: %w", err)
	}

	flags.encoder = enc

	return nil
}
