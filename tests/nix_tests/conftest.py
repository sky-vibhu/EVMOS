import pytest

from .network import setup_nexa, setup_geth


@pytest.fixture(scope="session")
def nexa(tmp_path_factory):
    path = tmp_path_factory.mktemp("nexa")
    yield from setup_nexa(path, 26650)


@pytest.fixture(scope="session")
def geth(tmp_path_factory):
    path = tmp_path_factory.mktemp("geth")
    yield from setup_geth(path, 8545)


@pytest.fixture(scope="session", params=["nexa", "nexa-ws"])
def nexa_rpc_ws(request, nexa):
    """
    run on both nexa and nexa websocket
    """
    provider = request.param
    if provider == "nexa":
        yield nexa
    elif provider == "nexa-ws":
        nexa_ws = nexa.copy()
        nexa_ws.use_websocket()
        yield nexa_ws
    else:
        raise NotImplementedError


@pytest.fixture(scope="module", params=["nexa", "nexa-ws", "geth"])
def cluster(request, nexa, geth):
    """
    run on nexa, nexa websocket and geth
    """
    provider = request.param
    if provider == "nexa":
        yield nexa
    elif provider == "nexa-ws":
        nexa_ws = nexa.copy()
        nexa_ws.use_websocket()
        yield nexa_ws
    elif provider == "geth":
        yield geth
    else:
        raise NotImplementedError
