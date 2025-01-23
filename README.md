## Minstrel

A GUI app (SmartSDR-like thingy) for the FLEX-6000 and FLEX-8000 series
radios that runs on Linux.

Currently very early alpha.

### Building

Install deps. For Ubuntu:

```sh
apt install build-essential golang-go libopusfile-dev libasound2-dev \
    libglfw3-dev libxcursor-dev libxinerama-dev libxi-dev libxxf86vm-dev
```

Build:

```sh
go build
```

Run:

```sh
./minstrel
```

## Tablet Mode

Besides desktops, Minstrel is designed to run on a Raspberry Pi (3 or above) or
similar single-board computer with a MIPI DSI touchscreen display. The minimum
resolution required is 720x480, and a size of at least 5 inches is recommended,
or touch targets will be too small. 7 inches is better.

Minstrel can run without any desktop environment, X server, or anything else, by
launching it under the [cage](https://github.com/cage-kiosk/cage) Wayland
compositor, and it's possible to boot directly into it by setting it as the
greeter program of [greetd](https://sr.ht/~kennylevinsen/greetd/) or similar.
