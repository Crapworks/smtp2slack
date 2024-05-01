# SMTP2Slack

This is small go program that runs an STMP server, forwarding all incoming messages to Slack.

## What for?

I am running a couple of servers at home and many tools still want an SMTP server to send notifications. But E-Mail is hard and most mail providers suck. Since I already get many notifications from Grafana, TrueNAS, etc. via Slack, I wrote this tool to run in my k8s cluster and provide STMP for services that need it.

I am using this as well for systems like Authelia and Vaultwarden, who wants to send mails for verification purposes. Since those are sensible information and I do not want them hangign around in Slack, one can specify a list of senders that need encryption and provide a PGP public key for that.

## Configuration

You can configure `smtp2slack` either via command line switches or via environment variables. Command line switches will override environment variables if both are used:

| Command line       | Environment   | Description |
| -------------------| ------------- | ----------- |
| --addr ADDR        | LISTEN_ADDR   | address string to listen on (default: 0.0.0.0:2525)  |
| --channel CHANNEL  | CHANNEL       | channel to forward the mails to |
| --token TOKEN      | TOKEN         | slack authentication token |
| --auth AUTH        | AUTH          | user:passwd combination for authentication |
| --encryptedsenders ENCRYPTEDSENDERS | ENCRYPTED_SENDERS | sender addresses which mails should be encrypted |
| --pubkey PUBKEY    | PUBKEY        | path to a file that contains the public key for encryption |
| --tlscert TLSCERT  | TLSCERT       | path to tls certificate |
| --tlskey TLSKEY    | TLSKEY        | path to tls key |
| --watchsecret WATCHSECRET | WATCHSECRET | watch secret and restart if it is changed. format is namespace:secretname |

## Quick Start

The minimum amount of options to run `smtp2slack`:

```bash
$ ./smtp2slack -channel "#mail" -token "slack-bot-auth-token"
```

This will listen on `0.0.0.0:2525` for incoming mails, sending them to the slack channel `#mail` using the specified slack token. No authentication is required, no encryption on the transport layer is performed. This should obviously not be exposed to any unprotected network.

## Authentication

To enable SMTP Authentication, you have to provide credentials using the htpasswd format. You can add multiple users/passwords by separating them with a space.
For example, to have a user "test" with password "test" using BCrypt hashing:

```bash
$ htpasswd -n -B test
New password:
Re-type new password:
test:$2y$05$nXjCLZVD6/q9XHcksX7LOOPLocw6zqJCIkq4PDq5lFDKZOu28aZSy
```

You can then pass the credentials to `smtp2slack`:

```bash
$ ./smpt2slack -channel "#mail" -token "slack-bot-auth-token" -auth 'test:$2y$05$nXjCLZVD6/q9XHcksX7LOOPLocw6zqJCIkq4PDq5lFDKZOu28aZSy'
```

## TLS

If you are using SMTP authentication, you want to make sure that your passwords are not sent in clear test. To do this, `smpt2slack` supports the usage of `STARTTLS`. This should be supported by pretty much every SMTP client. To enable it, you just have to specify the path to the certificate and private key:

```bash
./smpt2slack -channel "#mail" -token "slack-bot-auth-token" -tlscert ./my.cert -tlskey ./my.key
```

## TLS with Cert-Manager on Kubernetes

I run `smtp2slack` in my kubernetes cluster together with [cert-manager](https://cert-manager.io/), which issues and renews letsencrypt certificates for me. In order to use a new certificate once it has been updated by `cert-manager`, `smtp2slack` needs to restart in order to use it. To automate that, you can tell `smtp2slack` that it runs in a kubernetes cluster and what secret it should watch. Once that secret changes, `smtp2slack` will restart and use it. The format of the parameter is `namespace`:`secret`. So to watch a secret called `smtpcert` in namespace `smtp2slack` the command is:

```bash
./smpt2slack -channel "#mail" -token "slack-bot-auth-token" -watchsecret "smtp2slack:smtpcert"
```

## Encryption

I also use `smtp2auth` to forward sensitive information to slack, like for example password recovery links. Since I don't want them to end up in Slack unecrypted, there is an option to speficy a pgp public key and a list of sender addresses that should be encryped with it:

```bash
./smpt2slack -channel "#mail" -token "slack-bot-auth-token" -pubkey ./pgp.pub -encryptedsenders "authelia@mydomain.org"
```