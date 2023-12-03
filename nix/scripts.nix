{ pkgs
, config
, nexa ? (import ../. { inherit pkgs; })
}: rec {
  start-nexa = pkgs.writeShellScriptBin "start-nexa" ''
    # rely on environment to provide nexad
    export PATH=${pkgs.test-env}/bin:$PATH
    ${../scripts/start-nexa.sh} ${config.nexa-config} ${config.dotenv} $@
  '';
  start-geth = pkgs.writeShellScriptBin "start-geth" ''
    export PATH=${pkgs.test-env}/bin:${pkgs.go-ethereum}/bin:$PATH
    source ${config.dotenv}
    ${../scripts/start-geth.sh} ${config.geth-genesis} $@
  '';
  start-scripts = pkgs.symlinkJoin {
    name = "start-scripts";
    paths = [ start-nexa start-geth ];
  };
}
