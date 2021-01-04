# Tasadar-API
[![DeepSource](https://deepsource.io/gh/tionis/api.svg/?label=active+issues&show_trend=true&token=zajw9kzTw_hnN54R-UBD4pjP)](https://deepsource.io/gh/tionis/api/?ref=repository-badge)
Tasadar API and Bot network, designed to be run on Heroku Platform, but should run on any Linux Platform, where a https load balancer is loaded, the dns record of api.tasadar.net points to the load balancer, and the load balancers addressed port is specified on $PORT together with the tokens.

# Things needed for Operation
This API needs Tokens to use its bot bindings and a postgres database.
Listed below you'll find all requisites that are needed.

## Environment Variables needed for this application
 - PORT - Set port of http endpoint
 - DISCORD_TOKEN - Glyph Bot Discord Token
 - TELEGRAM_TOKEN - Glyph Bot Telegram Token
 - UNIPASSAUBOT_TOKEN - Uni Passau Bot Telegram token
 - MODE = production - Set mode to production
 - DATABASE_URL - URL for Postgres Database
 - MATRIX_HOMSERVER_URL - URL to reach homeserver at
 - MATRIX_USERNAME - Username of the bot account (only localpart)
 - MATRIX_PASSWORD - Password for the bot account
