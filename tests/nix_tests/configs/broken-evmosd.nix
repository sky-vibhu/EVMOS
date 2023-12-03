{ pkgs ? import ../../../nix { } }:
let nexad = (pkgs.callPackage ../../../. { });
in
nexad.overrideAttrs (oldAttrs: {
  patches = oldAttrs.patches or [ ] ++ [
    ./broken-nexad.patch
  ];
})
