{
  description = "Kartoza Video Processor - Screen recording tool for Wayland";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "0.1.0";

        # Helper function for cross-compilation
        mkPackage = { pkgs, system, GOOS, GOARCH }:
          pkgs.buildGoModule {
            pname = "kartoza-video-processor";
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
                mv kartoza-video-processor kartoza-video-processor.exe
              fi

              # Create release tarball
              mkdir -p $out/release
              if [ "${GOOS}" = "windows" ]; then
                tar -czf $out/release/kartoza-video-processor-${GOOS}-${GOARCH}.tar.gz kartoza-video-processor.exe
              else
                tar -czf $out/release/kartoza-video-processor-${GOOS}-${GOARCH}.tar.gz kartoza-video-processor
              fi

              # Install desktop file (Linux only)
              if [ "${GOOS}" = "linux" ]; then
                mkdir -p $out/share/applications
                cat > $out/share/applications/kartoza-video-processor.desktop << EOF
              [Desktop Entry]
              Name=Kartoza Video Processor
              Comment=Screen recording tool for Wayland
              Exec=kartoza-video-processor
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
              homepage = "https://github.com/kartoza/kartoza-video-processor";
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

          kartoza-video-processor = self.packages.${system}.default;

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
            name = "kartoza-video-processor-all-releases";
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
            alias gob='go build -o bin/kartoza-video-processor .'
            alias gom='go mod tidy'
            alias gol='golangci-lint run'

            echo ""
            echo "🎬 Kartoza Video Processor Development Environment"
            echo ""
            echo "Available commands:"
            echo "  gor  - Run the application"
            echo "  got  - Run tests"
            echo "  gob  - Build binary"
            echo "  gom  - Tidy go modules"
            echo "  gol  - Run linter"
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
            program = "${self.packages.${system}.default}/bin/kartoza-video-processor";
          };

          setup = {
            type = "app";
            program = toString (pkgs.writeShellScript "setup" ''
              echo "Initializing kartoza-video-processor..."
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
