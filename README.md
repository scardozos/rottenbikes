# Rotten Bikes

An open-source platform for community reviews, ratings, and condition tracking of urban bike-share fleets.

> **Disclaimer:** Rotten Bikes is an independent, community-driven open-source project. It is not affiliated, associated, authorized, endorsed by, or in any way officially connected with Bicing, Barcelona de Serveis Municipals (B:SM), Smou, the Ajuntament de Barcelona, or any official public transit operator. 
>
> All product names, trademarks, and registered trademarks mentioned herein (such as "Bicing") are the property of their respective owners. Any reference to third-party services or marks is strictly for identification, reference, and descriptive nominative fair use purposes.

## Overview

Rotten Bikes allows commuters and riders to crowdsource fleet quality data for public and shared bicycles (such as Barcelona's Bicing system). Users can scan vehicle QR/Barcodes to leave and view ratings across key physical components (brakes, pedals, electric power, seat comfort) to help fellow riders avoid faulty equipment and report maintenance needs.

- **Passwordless login:** magic links by email, with a login code for logging in another device (request on your laptop, read the email on your phone).
- **Bike scanning:** scan a bike's QR code to see its ratings, or add it if it's new.
- **Reviews:** rate brakes, seat, sturdiness, power and pedals; see how a bike has been doing over the last weeks.
- **Moderation:** admin tools to remove abusive posters and bikes, with an audit log.

Backend in Go with PostgreSQL; app in React Native (Expo) for iOS, Android and web.

## Quickstart

Requires Go, Docker, Node.js, [golang-migrate](https://github.com/golang-migrate/migrate) and Make (details in [development](docs/development.md#prerequisites)).

```sh
make run
```

This starts PostgreSQL, applies migrations and seed data, and runs the API on `http://localhost:8080` and the app on `http://localhost:8081` (or in Expo Go).

## Documentation

| Doc | What's in it |
| :--- | :--- |
| [Development](docs/development.md) | Local setup, running the UI, database tasks, how to add endpoints, queries and migrations. |
| [API reference](docs/api.md) | Every endpoint, auth, errors and rate limits. |
| [Architecture](docs/architecture.md) | Components, code layout, data model, how login, scanning, reviews and moderation work. |
| [Configuration](docs/configuration.md) | Environment variables, env files, Kubernetes config and secrets. |
| [Testing](docs/testing.md) | Unit tests (Go and UI), the E2E suite, and running it against local, dev and prod. |
| [Operations](docs/operations.md) | Environments, deployment, admin roles, metrics, logs and health checks. |
| [Contributing](docs/contributing.md) | Workflow, PR checklist, testing expectations and code conventions. |
