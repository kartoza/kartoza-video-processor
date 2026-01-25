{
  description = "Kartoza Screencaster - Screen recording tool for Wayland";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "0.7.1";

        # MkDocs with Material theme for documentation
        mkdocsEnv = pkgs.python3.withPackages (ps: with ps; [
          mkdocs
          mkdocs-material
          mkdocs-minify-plugin
          pygments
          pymdown-extensions
        ]);

        # Helper function for cross-compilation
        mkPackage = { pkgs, system, GOOS, GOARCH }:
          pkgs.buildGoModule {
            pname = "kartoza-screencaster";
            inherit version;
            src = ./.;

            vendorHash = null;

            CGO_ENABLED = 0;
            inherit GOOS GOARCH;

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
            ];

            tags = [ "release" ];

            # Platform-specific binary name
            postInstall = ''
              cd $out/bin
              if [ "${GOOS}" = "windows" ]; then
                mv kartoza-screencaster kartoza-screencaster.exe
              fi

              # Create release tarball
              mkdir -p $out/release
              if [ "${GOOS}" = "windows" ]; then
                tar -czf $out/release/kartoza-screencaster-${GOOS}-${GOARCH}.tar.gz kartoza-screencaster.exe
              else
                tar -czf $out/release/kartoza-screencaster-${GOOS}-${GOARCH}.tar.gz kartoza-screencaster
              fi

              # Install desktop file (Linux only)
              if [ "${GOOS}" = "linux" ]; then
                mkdir -p $out/share/applications
                cat > $out/share/applications/kartoza-screencaster.desktop << EOF
              [Desktop Entry]
              Name=Kartoza Screencaster
              Comment=Screen recording tool for Wayland
              Exec=kartoza-screencaster
              Icon=video-x-generic
              Terminal=true
              Type=Application
              Categories=AudioVideo;Video;Recorder;
              Keywords=screen;recording;video;wayland;
              EOF
              fi
            '';

            meta = with pkgs.lib; {
              description = "Screen recording tool for Wayland with audio processing";
              homepage = "https://github.com/kartoza/kartoza-screencaster";
              license = licenses.mit;
              maintainers = [ ];
              platforms = platforms.unix ++ platforms.windows;
            };
          };

      in
      {
        packages = {
          default = mkPackage {
            inherit pkgs system;
            GOOS = if pkgs.stdenv.isDarwin then "darwin" else if pkgs.stdenv.isLinux then "linux" else "linux";
            GOARCH = if pkgs.stdenv.hostPlatform.isAarch64 then "arm64" else "amd64";
          };

          kartoza-screencaster = self.packages.${system}.default;

          # Cross-compiled packages
          linux-amd64 = mkPackage {
            inherit pkgs system;
            GOOS = "linux";
            GOARCH = "amd64";
          };

          linux-arm64 = mkPackage {
            inherit pkgs system;
            GOOS = "linux";
            GOARCH = "arm64";
          };

          darwin-amd64 = mkPackage {
            inherit pkgs system;
            GOOS = "darwin";
            GOARCH = "amd64";
          };

          darwin-arm64 = mkPackage {
            inherit pkgs system;
            GOOS = "darwin";
            GOARCH = "arm64";
          };

          windows-amd64 = mkPackage {
            inherit pkgs system;
            GOOS = "windows";
            GOARCH = "amd64";
          };

          # All releases combined
          all-releases = pkgs.symlinkJoin {
            name = "kartoza-screencaster-all-releases";
            paths = [
              self.packages.${system}.linux-amd64
              self.packages.${system}.linux-arm64
              self.packages.${system}.darwin-amd64
              self.packages.${system}.darwin-arm64
              self.packages.${system}.windows-amd64
            ];
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go
            gopls
            golangci-lint
            gomodifytags
            gotests
            impl
            delve
            go-tools

            # Build tools
            gnumake
            gcc
            pkg-config

            # CLI utilities
            ripgrep
            fd
            eza
            bat
            fzf
            tree
            jq
            yq

            # Recording dependencies (for testing)
            wl-screenrec
            ffmpeg
            pipewire
            libnotify

            # Documentation
            mkdocsEnv

            # Nix tools
            nil
            nixpkgs-fmt
            nixfmt-classic

            # Git
            git
            gh

            # Security
            trivy
          ];

          shellHook = ''
            export EDITOR=nvim
            export GOPATH="$PWD/.go"
            export GOCACHE="$PWD/.go/cache"
            export GOMODCACHE="$PWD/.go/pkg/mod"
            export PATH="$GOPATH/bin:$PATH"

            # Helpful aliases
            alias gor='go run .'
            alias got='go test -v ./...'
            alias gob='go build -o bin/kartoza-screencaster .'
            alias gom='go mod tidy'
            alias gol='golangci-lint run'

            # Documentation aliases
            alias docs='mkdocs serve'
            alias docs-build='mkdocs build'

            echo ""
            echo "🎬 Kartoza Screencaster Development Environment"
            echo ""
            echo "Available commands:"
            echo "  gor  - Run the application"
            echo "  got  - Run tests"
            echo "  gob  - Build binary"
            echo "  gom  - Tidy go modules"
            echo "  gol  - Run linter"
            echo ""
            echo "Documentation:"
            echo "  docs       - Serve docs locally (http://localhost:8000)"
            echo "  docs-build - Build static docs site"
            echo ""
            echo "Make targets:"
            echo "  make build    - Build binary"
            echo "  make test     - Run tests"
            echo "  make lint     - Run linter"
            echo "  make release  - Build all platforms"
            echo ""
          '';
        };

        apps = {
          default = {
            type = "app";
            program = "${self.packages.${system}.default}/bin/kartoza-screencaster";
          };

          setup = {
            type = "app";
            program = toString (pkgs.writeShellScript "setup" ''
              echo "Initializing kartoza-screencaster..."
              go mod download
              go mod tidy
              echo "Setup complete!"
            '');
          };

          release = {
            type = "app";
            program = toString (pkgs.writeShellScript "release" ''
              echo "Building all release binaries..."
              nix build .#all-releases
              mkdir -p release
              cp -r result/release/* release/
              echo "Release binaries created in ./release/"
            '');
          };

          release-upload = {
            type = "app";
            program = toString (pkgs.writeShellScript "release-upload" ''
              TAG="$1"
              if [ -z "$TAG" ]; then
                echo "Usage: nix run .#release-upload -- vX.Y.Z"
                exit 1
              fi

              echo "Building and uploading release $TAG..."
              nix build .#all-releases
              mkdir -p release
              cp -r result/release/* release/

              # Generate checksums
              cd release
              sha256sum *.tar.gz > checksums.txt
              cd ..

              # Upload to GitHub
              gh release upload "$TAG" release/*.tar.gz release/checksums.txt --clobber

              echo "Release $TAG uploaded successfully!"
            '');
          };
        };
      }
    );
}
