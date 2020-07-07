# Tasadar-API
Tasadar API and Bot network, designed to be run on Heroku Platform, but should run on any Linux Platform, where a https load balancer is loaded, the dns record of api.tasadar.net points to the load balancer, and the load balancers addressed port is specified on $PORT together with the tokens.
The needed localffmpeg can be installed in many ways, more coming soon.

# Things needed for Operation
This API needs Tokens to use its bot bindings, and a RESP compatible Database defined by REDIS_URL.
This database could for example be ardb or simply redis. A localffmpeg install is needed
Listed below you'll find all requisites that are needed.

## Environment Variables needed for this application
 - PORT
 - DISCORD_TOKEN
 - TELEGRAM_TOKEN
 - UNIPASSAUBOT_TOKEN
 - QUOTATOR_TOKEN
 - MODE = production 
 - REDIS_URL - important for central functions of the bot
 - DATABASE_URL
