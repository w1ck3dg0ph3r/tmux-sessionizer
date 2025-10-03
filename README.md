# tmux-sessionizer

Simple tmux session manager that fuzzy selects accross git repositories and explicitly configured directories.

## Requirements

- [tmux](https://github.com/tmux/tmux) (duh)
- [fzf](https://github.com/junegunn/fzf)
- [pgrep](https://man7.org/linux/man-pages/man1/pgrep.1.html)

## Configuration

Configuration is read from `~/.config/tmux-sessionizer`:

```yaml
# Directories to scan for git repositories:
scan:
- {path: ~/projects, depth: 3}

# Additional directories:
dirs:
- ~/.local/bin
- ~/projects/not-a-git-repo
```

## Integrations

### bash keybinding

```bash
type -P tmux-sessionizer &>/dev/null && bind '"\C-f":"tmux-sessionizer\n"'
```

### tmux keybinding

```tmux
bind-key -r f run-shell "tmux neww ~/.local/bin/tmux-sessionizer"
```
