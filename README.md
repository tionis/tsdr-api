# Tasadar-Heroku
[![DeepSource](https://static.deepsource.io/deepsource-badge-light-mini.svg)](https://deepsource.io/gl/tionis/tasadar-api/?ref=repository-badge)
[![Gitpod Ready-to-Code](https://img.shields.io/badge/Gitpod-Ready--to--Code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/tionis/tasadar-heroku) 

Tasadar API and Bot network, designed to be run on Heroku Platform, but should run on any Linux Platform, where an https load balancer is loaded, the dns record of api.tasadar.net points to the load balancer and the load balancers addressed port is specified on $PORT together with the tokens.

# Things needed for Operation
This API needs Tokens to use its bot bindings and a RESP compatible Database defined by REDIS_URL.
This database could for example be ardb or simply redis.
Below are listed all requesits that are needed

## Environment Variables needed for this application
 - PORT
 - DISCORD_TOKEN
 - TELEGRAM_TOKEN
 - UNIPASSAUBOT_TOKEN
 - QUOTATOR_TOKEN
 - MODE = production 
 - REDIS_URL - important for central functions of the bot
 - DATABASE_URL