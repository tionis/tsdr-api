# Postgres Migration
## Simple Redis Values
### Old Values
Following Redis Values have to be replaced:
 - dg|status -> AppData
 - dg|:discord-id|:username (for account linking) -> 3PID
 - mc|IsRunning -> AppData
 - mc|LastPlayerOnline -> AppData
 - TOTP-Secret|... -> TOTP-Secrets
 - tg|:telegram-id|:username (for account linking) -> 3PID
 - auth|:username|hash -> user
 - auth|:username|groups -> user_rel_groups
 - auth|:username|name -> user
 - auth|:username|email -> user
 - auth|:username|mc -> 3PID
 - token|:token|IFTTT -> Tokens
 - token|:token|alpha -> Tokens
 - mc|token|:token -> Tokens
 - IOT-States (iot|:home|:service) to on/off -> iot
 

 ### New Values
 This will be the Data Structure
 - User-Table
   - Username
   - Password Hash
   - Email
   - First Name
   - Last Name
   - MainGroup
 - Groups-Table
   - Group name
   - Parent
 - UserXGroup - Table
   - Username
   - Group
 - 3PID
   - Username 
   - Telegram ID
   - Discord ID
   - MC Username
   - WhatsApp ID
   - More Stuff...
 - AppData
   - Key
   - Value
 - TOTP-Secrets(encrypt!)
   - Owner
   - ID
   - Friendly Name
   - Secret
 - Tokens
   - Token-ID
   - Designated Service
 - iot
   - Owner
   - Home
   - Service
   - Status