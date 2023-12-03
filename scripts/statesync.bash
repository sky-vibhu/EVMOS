#!/bin/bash
# microtick and bitcanna contributed significantly here.
# Pebbledb state sync script.
# invoke like: bash scripts/ss.bash



## USAGE RUNDOWN
# Not for use on live nodes
# For use when testing.
# Assumes that ~/.nexad doesn't exist
# can be modified to suit your purposes if ~/.nexad does already exist


set -uxe

# Set Golang environment variables.
export GOPATH=~/go
export PATH=$PATH:~/go/bin

# Install with pebbledb 
# go mod edit -replace github.com/tendermint/tm-db=github.com/baabeetaa/tm-db@pebble
# go mod tidy
# go install -ldflags '-w -s -X github.com/cosmos/cosmos-sdk/types.DBBackend=pebbledb -X github.com/tendermint/tm-db.ForceSync=1' -tags pebbledb ./...

go install ./...

# NOTE: ABOVE YOU CAN USE ALTERNATIVE DATABASES, HERE ARE THE EXACT COMMANDS
# go install -ldflags '-w -s -X github.com/cosmos/cosmos-sdk/types.DBBackend=rocksdb' -tags rocksdb ./...
# go install -ldflags '-w -s -X github.com/cosmos/cosmos-sdk/types.DBBackend=badgerdb' -tags badgerdb ./...
# go install -ldflags '-w -s -X github.com/cosmos/cosmos-sdk/types.DBBackend=boltdb' -tags boltdb ./...

# Initialize chain.
nexad init test --chain-id nexa_9000-1

# Get Genesis
wget https://archive.nexa.org/mainnet/genesis.json
mv genesis.json ~/.nexad/config/


# Get "trust_hash" and "trust_height".
INTERVAL=1000
LATEST_HEIGHT=$(curl -s https://nexa-rpc.polkachu.com/block | jq -r .result.block.header.height)
BLOCK_HEIGHT=$(($LATEST_HEIGHT-$INTERVAL)) 
TRUST_HASH=$(curl -s "https://nexa-rpc.polkachu.com/block?height=$BLOCK_HEIGHT" | jq -r .result.block_id.hash)

# Print out block and transaction hash from which to sync state.
echo "trust_height: $BLOCK_HEIGHT"
echo "trust_hash: $TRUST_HASH"

# Export state sync variables.
export NEXAD_STATESYNC_ENABLE=true
export NEXAD_P2P_MAX_NUM_OUTBOUND_PEERS=200
export NEXAD_STATESYNC_RPC_SERVERS="https://rpc.nexa.interbloc.org:443,https://nexa-rpc.polkachu.com:443,https://tendermint.bd.nexa.org:26657,https://rpc.nexa.posthuman.digital:443,https://rpc.nexa.testnet.run:443,https://rpc.nexa.bh.rocks:443"
export NEXAD_STATESYNC_TRUST_HEIGHT=$BLOCK_HEIGHT
export NEXAD_STATESYNC_TRUST_HASH=$TRUST_HASH

# Fetch and set list of seeds from chain registry.
export NEXAD_P2P_SEEDS=$(curl -s https://raw.githubusercontent.com/cosmos/chain-registry/master/nexa/chain.json | jq -r '[foreach .peers.seeds[] as $item (""; "\($item.id)@\($item.address)")] | join(",")')

# Start chain.
nexad start --x-crisis-skip-assert-invariants 
