# Faucet

This faucet app allows anyone to request tokens for a Cosmos account address. This app needs to be deployed on a Cosmos SDK based full node, because it relies on using the binary to send tokens.

**Note**: This faucet works only for Cosmos SDK based application whose version is greater than or equal to `v0.40`.

## Prerequisites

### Checkout Code

Requires Go and the `dep` dependency tool to be installed. 

```
go get git@github.com:vitwit/faucet-curl
```

### Install Network Binary

Make sure to install network binary and add faucet account key with keyring-backend as `test` in instance where you are going to run this app.

## Setup

### Production

First, set the environment variables for the app, using `.env` as a template:

```
cd $GOPATH/src/github.com/vitwit/faucet-curl
cp .env .env.local
vi .env.local
```

Then build the app.

```
dep ensure
go build faucet.go
```

The following executable will run the faucet on `FAUCET_PUBLIC_URL` value configured in .env.local 

```
./faucet
```

**WARNING**: It's highly recommended to run a reverse proxy with rate limiting in front of this app. Included in this repo is an example `Caddyfile` that lets you run an TLS secured faucet that is rate limited to 1 claim per IP per day.

For now, rate-limit is enabled for `/faucet/<address>` endpoint. User( based on IP) by default will have 20 requests and request will be increased for every 15 seconds. To change this rate-limit, please go through `getVisitor()` in faucet.go.

### Development

Run `GO111MODULE=auto go run faucet.go` to start faucet server.

### Testing

Once after running faucet, run below command in other shell:

```
curl http://localhost:8080/faucet/<address>
```

Here, `http://localhost:8080` is assumed as configured `FAUCET_PUBLIC_URL` value. Replace `<address>` with your account address.
