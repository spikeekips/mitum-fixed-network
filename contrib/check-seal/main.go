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
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/contrib"
	"github.com/spikeekips/mitum/util/encoder"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

var Version string = "v0.1-proto3"

type mainFlags struct {
	Version         bool   `help:"print version"`
	Input           string `arg:"" help:"seal input file; '-' is stdin" optional:""`
	NetworkIDString string `arg:"" name:"network-id" help:"network-id" optional:""`
	networkID       []byte
	encoder         encoder.Encoder
	content         []byte
	// TODO set encoder
	// TODO set hint
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

	extraHinters := []hint.Hinter{
		contestlib.ContestAddress(""), // contest-address
	}

	if encs, err := contrib.LoadEncoder(extraHinters...); err != nil {
		return xerrors.Errorf("failed to load encoders: %w", err)
	} else {
		if e, err := encs.Encoder(jsonencoder.JSONType, ""); err != nil {
			return xerrors.Errorf("json encoder needs for quic-network: %w", err)
		} else {
			flags.encoder = e
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

	return nil
}
