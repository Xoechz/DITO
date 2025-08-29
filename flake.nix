{
  description = "GO NixShell";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { nixpkgs, ... }: {
    devShells.x86_64-linux =
      let
        pkgs = nixpkgs.legacyPackages.x86_64-linux;
      in
      {
        default = pkgs.mkShell {
          name = "GO";
          nativeBuildInputs = with pkgs; [
            go
            gopls
            delve
            golangci-lint
          ];
        };
      };
  };
}
