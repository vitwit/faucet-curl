# Cosmos Testnet Faucet

This faucet app allows anyone who passes a captcha to request tokens for a Cosmos account address. This app needs to be deployed on a Cosmos SDK based full node, because it relies on using the binary to send tokens.

**Note**: This faucet backend works only for Cosmos SDK based application whose version is greater than or equal to `v0.40`.

## Prerequisites

### reCAPTCHA

If you don't have a reCAPTCHA site setup for the faucet, now is the time to get one. Go to the [Google reCAPTCHA Admin](https://www.google.com/recaptcha/admin) and create a new reCAPTCHA site. For the version of captcha, choose `reCAPTCHA v2`.

### Checkout Code

The backend requires Go and the `dep` dependency tool to be installed. For the frontend, you also need to have node.js and the `yarn` dependency tool installed. 

```
go get git@github.com:vitwit/faucet-curl
```

### Install Network Binary

Make sure to install network binary and add faucet account key with keyring-backed as `test` in instance where you are going to run faucet backend.

## Backend Setup

### Production

First, set the environment variables for the backend, using `./backend/.env` as a template:

```
cd $GOPATH/src/github.com/vitwit/faucet-curl/backend
cp .env .env.local
vi .env.local
```

Then build the backend.

```
dep ensure
go build faucet.go
```

The following executable will run the faucet on port 8080. 

```
./faucet
```

**WARNING**: It's highly recommended to run a reverse proxy with rate limiting in front of this app. Included in this repo is an example `Caddyfile` that lets you run an TLS secured faucet that is rate limited to 1 claim per IP per day.

For now, rate-limit is enabled for `/faucet/<address>` endpoint. User( based on IP) by default will have 20 requests and request will be increased for every 15 seconds. To change this rate-limit, please go through `getVisitor()` in faucet.go.

### Development

Run `go run faucet.go` in the `backend` directory to serve the backend.

## Frontend Setup

### Production

First, set the environment variables for the frontend, using `./frontend/.env` as a template:

```
cd $GOPATH/src/github.com/vitwit/faucet-curl/frontend
cp .env .env.local
vi .env.local
```

Then build the frontend.

```
yarn
yarn build
```

Lastly, serve the `./frontend/dist` directory with the web server of your choice.

### Development

Run `yarn serve` in the `frontend` directory to serve the frontend with hot reload.
