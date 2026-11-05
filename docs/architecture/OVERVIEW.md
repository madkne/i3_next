# i3-next Architecture

## Overview

i3_next is a modular Wayland Desktop Environment
built around Sway.

## Main Components

### Compositor

Sway is responsible for:

- Wayland compositor
- Window management
- Workspace management
- Keyboard input
- Output management

### Desktop Core

The Go backend is responsible for:

- Service lifecycle
- Configuration
- Event management
- Sway integration
- System integration
- Plugin management

### Desktop Shell

Quickshell is responsible for:

- Panels
- Widgets
- Notification UI
- Control Center
- Application Launcher
- Desktop Settings

## Architecture Principles

1. Modular monolith for the initial release.
2. Core must not depend on UI.
3. UI communicates through stable APIs.
4. Sway communication happens through an adapter.
5. Services must support graceful shutdown.
6. External processes must be supervised.
7. Configuration changes must be validated.