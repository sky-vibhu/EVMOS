import json
import subprocess
from pathlib import Path
from typing import NamedTuple

from pystarport import ports

from .network import CosmosChain, Nexa, Hermes, setup_custom_nexa
from .utils import ADDRS, eth_to_bech32, wait_for_port

# Nexa IBC denom of aNEXB in crypto-org-chain
Nexa_IBC_DENOM = "ibc/8EAC8061F4499F03D2D1419A3E73D346289AE9DB89CAB1486B72539572B1915E"
RATIO = 10**10


class IBCNetwork(NamedTuple):
    nexa: Nexa
    other_chain: CosmosChain
    hermes: Hermes
    incentivized: bool


def prepare_network(tmp_path, file, other_chain_name, incentivized=False):
    file = f"configs/{file}.jsonnet"
    gen = setup_custom_nexa(tmp_path, 26700, Path(__file__).parent / file)
    nexa = next(gen)

    # set up another chain to connect to nexa
    if "chainmain" in other_chain_name:
        other_chain_name = "chainmain-1"
        other_chain = CosmosChain(
            nexa.base_dir.parent / other_chain_name, "chain-maind"
        )
        other_chain_denom = "basecro"
    if "stride" in other_chain_name:
        other_chain_name = "stride-1"
        other_chain = CosmosChain(nexa.base_dir.parent / other_chain_name, "strided")
        other_chain_denom = "ustrd"

    hermes = Hermes(nexa.base_dir.parent / "relayer.toml")
    # wait for grpc ready
    wait_for_port(ports.grpc_port(other_chain.base_port(0)))  # other_chain grpc
    wait_for_port(ports.grpc_port(nexa.base_port(0)))  # nexa grpc

    version = {"fee_version": "ics29-1", "app_version": "ics20-1"}
    incentivized_args = (
        [
            "--channel-version",
            json.dumps(version),
        ]
        if incentivized
        else []
    )

    # pystarport (used to start the setup), by default uses ethereum
    # hd-path to create the relayers keys on hermes.
    # If this is not needed (e.g. in Cosmos chains like Stride, Osmosis, etc.)
    # then overwrite the relayer key
    if "chainmain" not in other_chain_name:
        subprocess.run(
            [
                "hermes",
                "--config",
                hermes.configpath,
                "keys",
                "add",
                "--chain",
                other_chain_name,
                "--mnemonic-file",
                nexa.base_dir.parent / "relayer.env",
                "--overwrite",
            ],
            check=True,
        )

    subprocess.check_call(
        [
            "hermes",
            "--config",
            hermes.configpath,
            "create",
            "channel",
            "--a-port",
            "transfer",
            "--b-port",
            "transfer",
            "--a-chain",
            "nexa_9000-1",
            "--b-chain",
            other_chain_name,
            "--new-client-connection",
            "--yes",
        ]
        + incentivized_args
    )

    if incentivized:
        # register fee payee
        src_chain = nexa.cosmos_cli()
        dst_chain = other_chain.cosmos_cli()
        rsp = dst_chain.register_counterparty_payee(
            "transfer",
            "channel-0",
            dst_chain.address("relayer"),
            src_chain.address("signer1"),
            from_="relayer",
            fees=f"100000000{other_chain_denom}",
        )
        assert rsp["code"] == 0, rsp["raw_log"]

    nexa.supervisorctl("start", "relayer-demo")
    wait_for_port(hermes.port)
    yield IBCNetwork(nexa, other_chain, hermes, incentivized)


def assert_ready(ibc):
    # wait for hermes
    output = subprocess.getoutput(
        f"curl -s -X GET 'http://127.0.0.1:{ibc.hermes.port}/state' | jq"
    )
    assert json.loads(output)["status"] == "success"


def hermes_transfer(ibc, other_chain_name="chainmain-1", other_chain_denom="basecro"):
    assert_ready(ibc)
    # chainmain-1 -> nexa_9000-1
    my_ibc0 = other_chain_name
    my_ibc1 = "nexa_9000-1"
    my_channel = "channel-0"
    dst_addr = eth_to_bech32(ADDRS["signer2"])
    src_amount = 10
    src_denom = other_chain_denom
    # dstchainid srcchainid srcportid srchannelid
    cmd = (
        f"hermes --config {ibc.hermes.configpath} tx ft-transfer "
        f"--dst-chain {my_ibc1} --src-chain {my_ibc0} --src-port transfer "
        f"--src-channel {my_channel} --amount {src_amount} "
        f"--timeout-height-offset 1000 --number-msgs 1 "
        f"--denom {src_denom} --receiver {dst_addr} --key-name relayer"
    )
    subprocess.run(cmd, check=True, shell=True)
    return src_amount


def get_balance(chain, addr, denom):
    balance = chain.cosmos_cli().balance(addr, denom)
    print("balance", balance, addr, denom)
    return balance
