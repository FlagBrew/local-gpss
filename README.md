# Local GPSS

A local hostable version of the services that [PKSM](https://github.com/FlagBrew/PKSM) uses.

## How to Use
See the [Setup Guide](https://github.com/FlagBrew/local-gpss/wiki/Server-Setup-Guide)

## Docker

Images for `linux/amd64` and `linux/arm64` (Raspberry Pi 4/5 and other 64-bit Arm boards) are published to
`ghcr.io/flagbrew/local-gpss`. Use the `-nonroot` tags (`latest-nonroot`, `<version>-nonroot`): they run as an
unprivileged user and work with random UIDs such as OpenShift's. The plain tags (`latest`, `<version>`) run as root.

The interactive setup isn't available in the container, so it needs a `config.json` mounted in. To run it with Docker Compose:

```sh
cp config.example.json config.json   # adjust if needed
docker compose up -d
```

[`docker-compose.yml`](docker-compose.yml) uses the nonroot image, mounts `config.json` and keeps the sqlite database in
the `/data` volume, which is what [`config.example.json`](config.example.json) points at. Files and volumes you mount must
be readable by the container user, and writable for the database: uid 10001 or group 0. To use Postgres or MySQL instead,
change the `database` section of the config.

The container only writes to `/data` and `/tmp`, so the compose file runs it with a read-only root filesystem.

## Updating Auto Legality
This is a pain to do, and is one of the reasons why Auto Legality never really stayed up to date.

Essentially you'll need to clone https://github.com/santacrab2/PKHeX-Plugins and hope that the developer has it properly compiling
then you'll need to build the PKHeX.Core.AutoMod.dll and pair it with a version matching PKHeX.Core.dll

For your sanity, I have included the November 2024 dlls so that you can at-least compile and have something working out of the box.

## Some context

Back in 2019, I had an idea one random night about a sharing service for PKSM, something like the GTS but outside a
game, and that's
exactly what I ended up building that April and within a month or so had it ready and deployed it.

Along with GPSS, cloud legality was also introduced, it was often requested feature to add auto legalization, but
clearly the 3DS couldn't do it on its own, so with
GPSS being introduced, why not also introduce auto legality? Well, September 8th 2019, PKSM 7.0.0 was released to the
public
with both GPSS and Auto Legality.

In 2025, it'll have been online for 6+ years, almost 100,000 Pokémon uploaded and in total downloaded well-over 100,000
times.

### So why create this?

While it's been cool to have something like this and definitely a good learning experience, I just don't have time to
maintain it anymore
and PKSM likely won't see too many updates for the foreseeable future.

Rather than just completely shut down and leave everyone in the dark, I threw this together, it'll essentially act as a
local backend for PKSM,
you definitely should not expose this to the public internet, it isn't built with security in mind nor is it really
designed to be hit by many users at once.
