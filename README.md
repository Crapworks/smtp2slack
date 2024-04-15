# SMTP2Slack

This is small go program that runs an STMP server, forwarding all incoming messages to Slack.

> [!CAUTION]
> This is a personal playground project. Maybe someone finds the code useful and can implement it's own use case with it. This is the first time I wrote anything in Go and it sure shows.

## My use case

I am running a couple of servers at home and many tools still want an SMTP server to send notifications. But E-Mail is hard and most mail providers suck. Since I already get many notifications from Grafana, TrueNAS, etc. via Slack, I wrote this tool to run in my k8s cluster and provide STMP for services that need it.

I am using this as well for systems like Authelia and Vaultwarden, who wants to send mails for verification purposes. Since those are sensible information and I do not want them hangign around in Slack, one can specify a list of senders that need encryption and provide a PGP public key for that.

## Example

```bash
$ ./smtp2slack -channel "#mail" -token "slack-bot-auth-token" -auth 'htpasswd-line-for-smtpauth' -encrypted_senders important@secret.com -pubkey ~/public.pgp
```
