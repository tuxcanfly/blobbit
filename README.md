Usage:
======

```bash
make build
```

Builds a binary called `blobbit` in the `build` directory.

```bash
Submit a blob to the Celestia blockchain.

Usage:
  blobbit submit [flags]

Flags:
      --core.grpc.port string   Set a custom gRPC port for the core node connection. The --core.ip flag must also be provided. (default "9090")
      --core.ip string          Indicates node to connect to the given core node. Example: <ip>, 127.0.0.1. <dns>, subdomain.domain.tld Assumes RPC port 26657 and gRPC port 9090 as default unless otherwise specified.
      --core.rpc.port string    Set a custom RPC port for the core node connection. The --core.ip flag must also be provided. (default "26657")
  -h, --help                    help for submit
      --node.store string       The path to root/home directory of your Celestia Node Store
```

```bash
./build/blobbit submit --core.ip grpc-mocha.pops.one --node.store ~/.celestia-light-mocha-4/keys
```

Output:

```
(*types.TxResponse)(0x140005340c0)(code: 0
codespace: ""
data: 122A0A282F63656C65737469612E626C6F622E76312E4D7367506179466F72426C6F6273526573706F6E7365
events:
...
txhash: 013CAD6670F7960E99E8A66D5869A54AA267ADE5E36D3A8F719F562431F3D6E9
)
```
