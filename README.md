# turtlectl

A Go CLI tool to manage and run [Turtle WoW](https://turtle-wow.org/) on Linux.

Works on both **X11** and **Wayland** with automatic GPU detection (AMD/NVIDIA/Intel).

## Install

```bash
# From source
go install github.com/bnema/turtlectl@latest

# Arch Linux (AUR)
yay -S turtlectl-git
```

## Usage

```bash
turtlectl install    # Download AppImage + create desktop entry
turtlectl launch     # Start the game
turtlectl update     # Update AppImage only
turtlectl clean      # Remove config/cache (keeps game files)
turtlectl clean -a   # Full purge including game files
```

## Directories

| Type | Path |
|------|------|
| Data | `~/.local/share/turtle-wow` |
| Cache | `~/.cache/turtle-wow` |
| Game | `~/Games/turtle-wow` |

Override game directory: `TURTLE_WOW_GAME_DIR=/path/to/game turtlectl launch`

## License

MIT
