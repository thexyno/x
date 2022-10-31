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

    in
    {

      nixosModule = { config, options, lib, pkgs, ... }:
        let
          pk = self.packages.${pkgs.system};
        in
        with lib;
        {
          options.services.mailmover = {
            enable = mkEnableOption "Enables xynos mailmover service";
            schedule = mkOption {
              type = types.str;
              default = "*-*-* *:*:10";
            };
            configFile = mkOption {
              type = types.str;
              default = "/etc/mailmover.dhall";
            };
          };
          config = mkIf config.services.mailmover.enable {
            systemd.timers.mailmover = {
              wantedBy = [ "multi-user.target" ];
              description = "xynos mailmover service";
              timerConfig.OnCalendar = config.services.mailmover.schedule;
            };
            systemd.services.mailmover = {
              description = "xynos mailmover service";
              after = [ "network.target" ];
              wantedBy = [ "multi-user.target" ];
              serviceConfig = {
                DynamicUser = true;
                PrivateTmp = "true";
                PrivateDevices = "true";
                ProtectHome = "true";
                ProtectSystem = "strict";
                LoadCredential = "dhall:${config.services.mailmover.configFile}";
              };
              script = ''
                export HOME=/tmp
                exec ${pk.mailmover}/bin/mailmover ''${CREDENTIALS_DIRECTORY}/dhall
              '';
            };

          };
        };

      packages = forAllSystems (system:
        let pkgs = nixpkgsFor.${system}; in
        {
          mailmover = pkgs.buildGo118Module {
            pname = "mailmover";
            inherit version;
            src = ./mailmover;
            vendorSha256 = "sha256-0K/hgFbbstZI/I8cHCFy8Pn+fnT5bf8w+VfbxKT4DGo=";
          };
        }
      );

      devShell = forAllSystems (system:
        let pkgs = nixpkgsFor.${system}; in
        pkgs.mkShell {
          buildInputs = [ pkgs.go_1_18 pkgs.nixpkgs-fmt pkgs.lefthook pkgs.gopls ];
        }
      );

    };
}
