#!/bin/bash

KEY="dev0"
CHAINID="nexa_9000-1"
MONIKER="mymoniker"
DATA_DIR=$(mktemp -d -t nexa-datadir.XXXXX)

echo "create and add new keys"
./nexad keys add $KEY --home $DATA_DIR --no-backup --chain-id $CHAINID --algo "eth_secp256k1" --keyring-backend test
echo "init Nexa with moniker=$MONIKER and chain-id=$CHAINID"
./nexad init $MONIKER --chain-id $CHAINID --home $DATA_DIR
echo "prepare genesis: Allocate genesis accounts"
./nexad add-genesis-account \
"$(./nexad keys show $KEY -a --home $DATA_DIR --keyring-backend test)" 1000000000000000000aNEXB,1000000000000000000stake \
--home $DATA_DIR --keyring-backend test
echo "prepare genesis: Sign genesis transaction"
./nexad gentx $KEY 1000000000000000000stake --keyring-backend test --home $DATA_DIR --keyring-backend test --chain-id $CHAINID
echo "prepare genesis: Collect genesis tx"
./nexad collect-gentxs --home $DATA_DIR
echo "prepare genesis: Run validate-genesis to ensure everything worked and that the genesis file is setup correctly"
./nexad validate-genesis --home $DATA_DIR

echo "starting nexa node $i in background ..."
./nexad start --pruning=nothing --rpc.unsafe \
--keyring-backend test --home $DATA_DIR \
>$DATA_DIR/node.log 2>&1 & disown

echo "started nexa node"
tail -f /dev/null