{
  description = "A very basic flake";
  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "nixpkgs/nixos-22.05";
  outputs = { self, nixpkgs }:
    let

      # to work with older version of flakes
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";

      # Generate a user-friendly version number.
      version = builtins.substring 0 8 lastModifiedDate;

      # System types to support.
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

    in {

      packages = forAllSystems(system:
        let pkgs = nixpkgsFor.${system}; in {

              mailmover = pkgs.buildGoModule {
                inherit version;
                src = ./mailmover;
                vendorSha256 = pkgs.lib.fakeSha256;
              };
            }
      );

      devShell = forAllSystems (system: let pkgs = nixpkgsFor.${system}; in pkgs.mkShell {
        buildInputs = [ pkgs.go_1_18 pkgs.nixpkgs-fmt pkgs.lefthook pkgs.gopls ];
      }
      );

  };
}
