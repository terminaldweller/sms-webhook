#!/usr/bin/env sh

curl -k -u "ulk2nygabzhf5f4:gorgoroth" -X POST https://127.0.0.1:8090/hook -H 'Content-Type: application/json' -d '{"sender":"kismet-wifi","title":"alert","description":"hello hello hello"}'
