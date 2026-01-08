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

## Addon Registry

Browse **880+ addons** from the [Turtle WoW Wiki](https://turtle-wow.fandom.com/wiki/Addons), enriched with GitHub metadata (stars, last commit, author).

**Web browser:** https://bnema.github.io/turtlectl/

**CLI:**
```bash
turtlectl addons explore          # Interactive TUI browser
turtlectl addons explore -l       # Table output
turtlectl addons explore --json   # JSON output
turtlectl addons explore -r       # Force refresh from GitHub
```

The registry is updated daily via GitHub Actions.

## Directories

| Type | Path |
|------|------|
| Data | `~/.local/share/turtle-wow` |
| Cache | `~/.cache/turtle-wow` |
| Game | `~/Games/turtle-wow` |

Override game directory: `TURTLE_WOW_GAME_DIR=/path/to/game turtlectl launch`

## License

MIT
