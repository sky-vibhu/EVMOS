#!/bin/bash

KEYS="alice"
CHAINID="nexa_9016-1"
MONIKER="nexanode"
KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
# Set dedicated home directory for the streakkd instance
HOMEDIR="/data/nexad"

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json

# validate dependencies are installed
command -v jq >/dev/null 2>&1 || {
	echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"
	exit 1
}

# used to exit on first error
set -e

# Reinstall daemon
make build

# User prompt if an existing local node configuration is found.
if [ -d "$HOMEDIR" ]; then
	printf "\nAn existing folder at '%s' was found. You can choose to delete this folder and start a new local node with new keys from genesis. When declined, the existing local node is started. \n" "$HOMEDIR"
	echo "Overwrite the existing configuration and start a new local node? [y/n]"
	read -r overwrite
else
	overwrite="Y"
fi

# Setup local node if overwrite is set to Yes, otherwise skip setup
if [[ $overwrite == "y" || $overwrite == "Y" ]]; then
	# Remove the previous folder
	rm -rf "$HOMEDIR"

	# Set client config
	./build/nexad config keyring-backend $KEYRING --home "$HOMEDIR"
	./build/nexad config chain-id $CHAINID --home "$HOMEDIR"

	./build/nexad keys add $KEYS --keyring-backend $KEYRING --algo $KEYALGO --home "$HOMEDIR"

	./build/nexad init $MONIKER -o --chain-id $CHAINID --home "$HOMEDIR"

	# Change parameter token denominations to aNEXB
	jq '.app_state["staking"]["params"]["bond_denom"]="aNEXB"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["crisis"]["constant_fee"]["denom"]="aNEXB"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="aNEXB"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["evm"]["params"]["evm_denom"]="aNEXB"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# jq '.app_state["inflation"]["params"]["mint_denom"]="aNEXB"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
    jq '.app_state["mint"]["params"]["mint_denom"]="aNEXB"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# Set gas limit in genesis
	jq '.consensus_params["block"]["max_gas"]="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set claims decays
	jq '.app_state["claims"]["params"]["duration_of_decay"]="200s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["claims"]["params"]["duration_until_decay"]="200s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["deposit_params"]["max_deposit_period"]="200s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["voting_params"]["voting_period"]="200s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["recovery"]["params"]["packet_timeout_duration"]="140s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["staking"]["params"]["unbonding_time"]="200s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	

	#changes status in app,config files
        sed -i 's/timeout_propose = "3s"/timeout_propose = "3s"/g' "$CONFIG"
        sed -i 's/timeout_commit = "5s"/timeout_commit = "2s"/g' "$CONFIG"
        sed -i 's/seeds = ""/seeds = ""/g' "$CONFIG"
        sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
        sed -i 's/prometheus-retention-time  = "0"/prometheus-retention-time  = "1000000000000"/g' "$APP_TOML"
        sed -i 's/enabled = false/enabled = true/g' "$APP_TOML"
        sed -i 's/enable = false/enable = true/g' "$APP_TOML"
        sed -i 's/swagger = false/swagger = true/g' "$APP_TOML"


	# Allocate genesis accounts (cosmos formatted addresses)
	./build/nexad add-genesis-account $KEYS 10000000000000000000000000000000000000000000000aNEXB --keyring-backend $KEYRING --home "$HOMEDIR"

	# Sign genesis transaction
	./build/nexad gentx ${KEYS} 1000000000000000000000aNEXB --keyring-backend $KEYRING --chain-id $CHAINID --home "$HOMEDIR"
	
	# Collect genesis tx
	./build/nexad collect-gentxs --home "$HOMEDIR"

	# Run this to ensure everything worked and that the genesis file is setup correctly
	./build/nexad validate-genesis --home "$HOMEDIR"

fi

# Start the node
./build/nexad start --home "$HOMEDIR"


