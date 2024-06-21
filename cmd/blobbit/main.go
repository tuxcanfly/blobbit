package main

import (
	"context"
	"fmt"
	"os"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/pkg/appconsts"
	"github.com/celestiaorg/celestia-app/pkg/namespace"
	"github.com/celestiaorg/celestia-app/pkg/user"
	blobtypes "github.com/celestiaorg/celestia-app/x/blob/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	rootCmd.AddCommand(Submit(NodeFlags()))
	rootCmd.SetHelpCommand(&cobra.Command{})
}

func main() {
	err := run()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	return rootCmd.ExecuteContext(context.Background())
}

func PersistentPreRunEnv(cmd *cobra.Command, _ []string) error {
	var (
		ctx = cmd.Context()
		err error
	)

	// loads existing config into the environment
	ctx, err = ParseFlags(ctx, cmd)
	if err != nil {
		return err
	}

	cmd.SetContext(ctx)
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "blobbit [subcommand]",
	Short: "A CLI for interacting with the Celestia blockchain",
	Args:  cobra.NoArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: false,
	},
}

type (
	storePathKey  struct{}
	coreConfigKey struct{}
)

const (
	nodeStoreFlag     = "node.store"
	coreFlag          = "core.ip"
	coreRPCFlag       = "core.rpc.port"
	coreGRPCFlag      = "core.grpc.port"
	blobNamespaceFlag = "blob.namespace"
	blobDataFlag      = "blob.data"
)

// CoreConfig combines all configuration fields for managing the relationship with a Core node.
type CoreConfig struct {
	IP       string
	RPCPort  string
	GRPCPort string
}

// NodeFlags gives a set of hardcoded Node package flags.
func NodeFlags() *flag.FlagSet {
	flags := &flag.FlagSet{}

	flags.String(
		nodeStoreFlag,
		"",
		"The path to root/home directory of your Celestia Node Store",
	)
	flags.String(
		coreFlag,
		"",
		"Indicates node to connect to the given core node. "+
			"Example: <ip>, 127.0.0.1. <dns>, subdomain.domain.tld "+
			"Assumes RPC port 26657 and gRPC port 9090 as default unless otherwise specified.",
	)
	flags.String(
		coreRPCFlag,
		"26657",
		"Set a custom RPC port for the core node connection. The --core.ip flag must also be provided.",
	)
	flags.String(
		coreGRPCFlag,
		"9090",
		"Set a custom gRPC port for the core node connection. The --core.ip flag must also be provided.",
	)
	flag.String(
		blobNamespaceFlag,
		"1234567890",
		"The namespace of the blob to submit",
	)
	flag.String(
		blobDataFlag,
		"some data",
		"The blob data to submit",
	)

	return flags
}

// StorePath reads the store path from the context.
func StorePath(ctx context.Context) string {
	return ctx.Value(storePathKey{}).(string)
}

// WithStorePath sets Store Path in the given context.
func WithStorePath(ctx context.Context, storePath string) context.Context {
	return context.WithValue(ctx, storePathKey{}, storePath)
}

// ParseCoreConfig reads the store path from the context.
func ParseCoreConfig(ctx context.Context) CoreConfig {
	return ctx.Value(coreConfigKey{}).(CoreConfig)
}

// WithCoreConfig sets the node config in the Env.
func WithCoreConfig(ctx context.Context, config *CoreConfig) context.Context {
	return context.WithValue(ctx, coreConfigKey{}, *config)
}

// ParseFlags parses Node flags from the given cmd and applies values to Env.
func ParseFlags(ctx context.Context, cmd *cobra.Command) (context.Context, error) {
	store := cmd.Flag(nodeStoreFlag).Value.String()
	if store == "" {
		return nil, fmt.Errorf("must specify a node store path")
	}
	ctx = WithStorePath(ctx, store)
	coreIP := cmd.Flag(coreFlag).Value.String()
	if coreIP == "" {
		if cmd.Flag(coreGRPCFlag).Changed || cmd.Flag(coreRPCFlag).Changed {
			return nil, fmt.Errorf("cannot specify RPC/gRPC ports without specifying an IP address for --core.ip")
		}
		return ctx, nil
	}

	rpc := cmd.Flag(coreRPCFlag).Value.String()
	grpc := cmd.Flag(coreGRPCFlag).Value.String()
	var cfg CoreConfig

	cfg.IP = coreIP
	cfg.RPCPort = rpc
	cfg.GRPCPort = grpc

	ctx = WithCoreConfig(ctx, &cfg)
	return ctx, nil
}

// Submit is a demo function that shows how to use the signer to submit data
func Submit(fsets ...*flag.FlagSet) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit a blob to the Celestia blockchain",
		Args:  cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return PersistentPreRunEnv(cmd, args)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			encConf := encoding.MakeConfig(app.ModuleEncodingRegisters...)
			ring, err := keyring.New(app.Name, keyring.BackendTest, StorePath(ctx), os.Stdin, encConf.Codec)
			if err != nil {
				panic(err)
			}
			cfg := ParseCoreConfig(ctx)
			ns := namespace.MustNewV0([]byte(cmd.Flag(blobNamespaceFlag).Value.String()))
			blob, err := blobtypes.NewBlob(ns, []byte(cmd.Flag(blobDataFlag).Value.String()), appconsts.ShareVersionZero)
			if err != nil {
				panic(err)
			}
			return DemoSubmitData(fmt.Sprintf("%s:%s", cfg.IP, cfg.GRPCPort), ring, ns, blob)
		},
	}
	for _, set := range fsets {
		cmd.Flags().AddFlagSet(set)
	}
	return cmd
}

// SubmitData is a demo function that shows how to use the signer to submit data
// to the blockchain directly via a celestia node. We can manage this keyring
// using the `celestia-appd keys` or `celestia keys` sub commands and load this
// keyring from a file and use it to programmatically sign transactions.
func DemoSubmitData(grpcAddr string, kr keyring.Keyring, ns namespace.Namespace, blob *tmproto.Blob) error {
	// create an encoding config that can decode and encode all celestia-app
	// data structures.
	ecfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

	// create a connection to the grpc server on the consensus node.
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	// get the address of the account we want to use to sign transactions.
	rec, err := kr.Key("my_celes_key")
	if err != nil {
		return err
	}

	addr, err := rec.GetAddress()
	if err != nil {
		return err
	}

	// Setup the signer. This function will automatically query the relevant
	// account information such as sequence (nonce) and account number.
	signer, err := user.SetupSigner(context.TODO(), kr, conn, addr, ecfg)
	if err != nil {
		return err
	}

	gasLimit := blobtypes.DefaultEstimateGas([]uint32{uint32(len(blob.Data))})

	options := []user.TxOption{
		// here we're setting estimating the gas limit from the above estimated
		// function, and then setting the gas price to 0.1utia per unit of gas.
		user.SetGasLimitAndFee(gasLimit, 0.1),
	}

	// this function will submit the transaction and block until a timeout is
	// reached or the transaction is committed.
	resp, err := signer.SubmitPayForBlob(context.TODO(), []*tmproto.Blob{blob}, options...)
	if err != nil {
		return err
	}

	// check the response code to see if the transaction was successful.
	if resp.Code != 0 {
		// handle code
		fmt.Println(resp.Code, resp.Codespace, resp.RawLog)
	}

	spew.Dump(resp)

	return err
}
