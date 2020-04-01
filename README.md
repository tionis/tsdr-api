# Tasadar-Heroku
[![DeepSource](https://static.deepsource.io/deepsource-badge-light-mini.svg)](https://deepsource.io/gh/tionis/tasadar-heroku/?ref=repository-badge) 
[![CircleCI](https://circleci.com/gh/tionis/tasadar-heroku/tree/master.svg?style=svg&circle-token=1600ea2fac5bfc9bad7867288f6ec2712adf7563)](https://circleci.com/gh/tionis/tasadar-heroku/tree/master)
[![Gitpod Ready-to-Code](https://img.shields.io/badge/Gitpod-Ready--to--Code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/tionis/tasadar-heroku) 

Tasadar API and Bot network, designed to be run on Heroku Platform, but should run on any Linux Platform, where an https load balancer is loaded, the dns record of api.tasadar.net points to the load balancer and the load balancers addressed port is specified on $PORT together with the tokens.

# Environment Variables needed for this application
 - PORT
 - DISCORD_TOKEN
 - TELEGRAM_TOKEN
 - UNIPASSAUBOT_TOKEN
 - DATABASE_URL - may be removed in the future
 - GIN_MODE = release
 - RCON_PASS - shall be removed in the future
 - REDIS_URL - important for central functions of the bot
 - SMTP_PASSWORD - for sending of emails over mailgun

# ToDo
## Tasadar Network Specifics
Add Handling of Pinning event and similar tn integrations like this:
https://api.tasadar.net/tn/pin/Hash_here/?name=NameofPinHere
Should connect to ipfs-cluster api and pin hashes. Save owner of hashes to key value store and ipfs-cluster
Requires Authentication -> Depends on oAuth Projects

## oAuth
oAuth Client Implementation based on work from goth library (has to be adopted -> maybe add Pull Request)

## S3 For persistent storage
Use S3 (Region eu-west-1 for heroku eu) for persistent storage
--> Make this modular so it can bbe switched to local storage backed or s3-compatible storage
Maybe even write a ipns backed version over tn-gateway -> for better availability 