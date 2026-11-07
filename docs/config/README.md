# Configuration reference

Managed files: `desktop.yaml`, `plugins.yaml`, `services.yaml`, `shortcuts.yaml`.

## desktop.yaml

| Key | Type | Description |
|---|---|---|
| version | int | Schema version |
| session.name | string | Session name |
| session.compositor | string | Compositor binary |
| paths.config_dir | string | Config directory |
| paths.data_dir | string | Data directory |
| paths.state_dir | string | State directory |
| paths.cache_dir | string | Cache directory |
| logging.level | string | Log level |
| logging.format | string | Log format |
| ui.shell | string | Shell name |
| ui.theme | string | Theme name |

## plugins.yaml

| Key | Type | Description |
|---|---|---|
| version | int | Schema version |
| plugins[].name | string | Plugin name |
| plugins[].enabled | bool | Enabled |

## services.yaml

| Key | Type | Description |
|---|---|---|
| version | int | Schema version |
| services.<name>.enabled | bool | Enabled |

## shortcuts.yaml

| Key | Type | Description |
|---|---|---|
| version | int | Schema version |
| shortcuts[].name | string | Shortcut name |
| shortcuts[].binding | string | Key binding |
| shortcuts[].command | string | Command |
