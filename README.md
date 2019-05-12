# Fyne OpenSnitch

An [OpenSnitch](https://github.com/evilsocket/opensnitch) UI in Go using [Fyne](https://fyne.io)

The application is running as a gRPC server on a unix socket and will interact with OpenSnitch daemon.

**THIS SOFTWARE IS WORK IN PROGRESS, DO NOT EXPECT IT TO BE BUG FREE AND DO NOT RELY ON IT FOR ANY TYPE OF SECURITY.**

## Requirements

- OpenSnitch [daemon](https://github.com/evilsocket/opensnitch#daemon) >= v1.0.0b
- Fyne [dependencies](https://github.com/fyne-io/fyne#prerequisites) to compile the application

## Running

Ensure the OpenSnitch daemon is configured and running.

    go build -o fyne-opensnitch && ./fyne-opensnitch

And you should see a main window containting the OpenSnitch Network Statistics like the following:

![OpenSnitch Network Statistics Screenshot](screenshot/network_stats.png)

and every time an action is required to add a new rule:

![OpenSnitch Ask Rule Screenshot](screenshot/ask_rule.png)


## Credits

- [OpenSnitch](https://github.com/evilsocket/opensnitch)
- [Fyne](https://github.com/fyne-io/fyne)
- [Statik](https://github.com/rakyll/statik) for the static assets embedding
- [GNOME Terminal](https://github.com/GNOME/gnome-terminal) for the [terminal icon](https://github.com/GNOME/gnome-terminal/blob/gnome-3-32/data/icons/hicolor_apps_scalable_org.gnome.Terminal.svg)