# sms-webhook

A simple sms webhook with plain authentication to get the SMS you receive on your android phone on IRC.</br>
I have a made blogpost about it [here](https://blog.terminaldweller.com/posts/how_to_get_your_sms_on_irc).</br>

The IRC bot supports SASL plain authentication.</br>
The webhook endpoint itself supports HTTP basic authentication.</br>
The webhook has [pocketbase](https://github.com/pocketbase/pocketbase) integrated so you can use that to create new users.</br>

Last but not least, you will need a forwarding agent that actually sends the SMS you get on your android device to the webhook endpoint.</br>
Currently [this](https://github.com/bogkonstantin/android_income_sms_gateway_webhook) is what I'm using to forward my SMS to the webhook. Also make sure the app settings on android are changed accordingly because the forwarder needs to run in the background so make sure android does not battery-optimize it out of existence.</br>

### Config
An example config file:

```toml
IrcServer = myirc.awesome.net
IrcPort = 6669
IrcNick = mynick
IrcSaslUser = mynick
IrcSaslPass = h4x0r1337p055w0rd
IrcChannel = 1337p17
```


### Deployment

A docker compose file is available for a quick setup:
```
version: "3.9"
services:
  sms-webhook:
    image: sms-webhook
    build:
      context: .
    deploy:
      resources:
        limits:
          memory: 256M
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
    networks:
      - smsnet
    restart: unless-stopped
    ports:
      - "127.0.0.1:8091:8090"
    depends_on:
      - nginx
    volumes:
      - pb-vault:/sms-webhook/pb_data
      - ./config.toml:/opt/smswebhook/config.toml
    cap_drop:
      - ALL
    dns:
      - 9.9.9.9
    environment:
      - SERVER_DEPLOYMENT_TYPE=deployment
    entrypoint: ["/sms-webhook/sms-webhook"]
    command: ["serve", "--http=0.0.0.0:8090"]
  nginx:
    deploy:
      resources:
        limits:
          memory: 128M
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
    image: nginx:stable
    ports:
      - "8090:443"
    networks:
      - smsnet
    restart: unless-stopped
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - DAC_OVERRIDE
      - SETGID
      - SETUID
      - NET_BIND_SERVICE
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - /etc/letsencrypt/live/sms.terminaldweller.com/fullchain.pem:/etc/letsencrypt/live/sms.terminaldweller.com/fullchain.pem:ro
      - /etc/letsencrypt/live/sms.terminaldweller.com/privkey.pem:/etc/letsencrypt/live/sms.terminaldweller.com/privkey.pem:ro
networks:
  smsnet:
    driver: bridge
volumes:
  sms-vault:
  pb-vault:
```

The setup uses nginx as a reverse proxy for TLS termination. The nginx config for that is also provided in the repo.</br>
