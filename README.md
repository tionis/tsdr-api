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
 - MODE = production 
 - REDIS_URL - important for central functions of the bot
 - DISCORD_ADMIN
 - MAIN_CHANNEL_ID

# ToDo
## Tasadar Network Specifics
### Overview
Add Handling of Pinning event and similar tn integrations like this:
https://api.tasadar.net/tn/pin/Hash_here/?name=NameofPinHere
Should connect to ipfs-cluster api and pin hashes. Save owner of hashes to key value store and ipfs-cluster
Requires Authentication -> Depends on oAuth Projects
### Authentication for this szenario:
 1. Check if user has valid session header
 2. If not forward to login route with callback to original link (with hash to pin) else go to step 5
 3. Authenticate user there
 4. Go to step 1
 5. Check if user is allowed to use this service and which limits shall apply
 6. check if user has enough capacity in his limits
 7. pin it via the ipfs-cluster rest api and set owner to user-id
 8. forward to pinned content

## Persistent storage
Get persistent storage
 - Make this modular so it can be switched to local storage backend or s3-compatible storage, maybe even directly use ipfs node
 - Make a list of use cases for various dynamic szenarios

## API Authentication
Strictly divide Code between Auth Code and API Code. Use different subdomains for both.
API could use goth for client implementation and hydra for server implementation.
API can be developed before server is implemented as external provide can be used.
This may even be the better solution user experience-wise.

## Glyph Discord Bot
### Construct Specifics
 - Get rule sets from wiki/git-provider and transform them into machine readable form
 - Query these rule sets when question occurr and during character creation(the second part may need some special syntax in the markdown files)