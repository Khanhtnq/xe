# testnet-0005

The published genesis bundle for the `testnet-0005` network. A node's identity
is its genesis, not its binary — point a node at this directory and it joins
this network:

```sh
xe node --genesis-dir ./genesis/testnet-0005 --data ./data \
  --dial /ip4/45.77.226.208/tcp/9000/p2p/12D3KooWJg4PQYGSfNCupBWZdEWbKj7pgdp5MmmUXbPdBcp6YDtT
```

| | |
|---|---|
| `network_id` | `testnet-0005` |
| ledger genesis hash | `3ff640410d7879821b544d09545dcfb8e96fe4fc717634e8bff4e53cacd1e150` |
| statechain genesis hash | `d01aabc1a29dc64bc6bbc30031a716cd2e0ceb867e5479953cd5ad22c8f296d6` |
| treasury | `76ffbdce7d7a0b0270fd111c096cbcd9df9dc1280825237aab73ed8a74609828` |
| genesis supply | 42,000,000 XE |

Check what you are about to join before you join it:

```sh
xe verify-genesis --genesis-dir ./genesis/testnet-0005 \
  --expect-network-id testnet-0005 \
  --expect-statechain-hash d01aabc1a29dc64bc6bbc30031a716cd2e0ceb867e5479953cd5ad22c8f296d6
```

Both files are public artifacts: signed blocks containing public keys, account
addresses and the governance keysets they establish. No private material.

**These files are retired by a testnet wipe.** When the protocol changes the
network is re-bootstrapped under a new id and a new bundle is published beside
this one; the old network stops existing. See
[`docs/run-a-node.md`](../../docs/run-a-node.md).
