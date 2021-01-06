<a name="unreleased"></a>
## [Unreleased]

### Feat
- implemented part of the message send functionality
- switched to non blocking operations
- better abstraction for the web server
- reorganized repo for better abstraction
- implemented user db interaction logic
- added more todo tags and comments
- finished implementing quotator functionality
- starting to actually use better error handling
- better internal error handling
- finished implementation of quotator logic
- finished writing nwe telegram bot interface
- added more and better abstractions, but code not working yet
- refactor partly working
- started total refactor

### Fix
- fixed an issue with double waiting group adds
- some more fixes for [#11](https://github.com/tionis/tsdr-api/issues/11)
- fixed some issues for [#11](https://github.com/tionis/tsdr-api/issues/11)
- removed unneeded div from index.html
- now ports should be handled correctly
- fixed various larger bugs across the application but its not production ready yet
- correct mentioning in telegram should now work
- fixed the concurreny issue with the cache

### Refactor
- improved cyclo in some parts
- reduced cyclo by a bit especially in rollhelper


<a name="v0.0.1"></a>
## v0.0.1 - 2021-01-04
### Feat
- successfully dockerized application
- added forwarding to today's log entry
- added dm handling
- removed landing page handling page handling
- added handling of test site hosting
- added tasadar.net landing page handling
- removed last minecraft help test
- minor improvement of dice throwing
- removed music integration completly
- removed discord voice features
- added correct dockerfile and build system
- improved logging by using logging framework
- improved volume managment
- improved volume managment
- added final queue feature adjustments
- added queue concept
- fixed play command parsing
- added music stopping and echo timer
- removed redis sets without timer to one
- discord bot now joins channel of the user
- added first concept for voice functionality
- added git-crypt support
- added help message to cors proxy
- added cors proxy
- removed minecraft server functionality
- **GlyphDiscordBot:** :sparkles: Added first gm dice bot function

### Fix
- fixed various tiny bugs
- fixed little logging bugs
- fixed some minor race conditions
- fixed package name
- added response headers to cors proxy
- **GlyphDiscordBot:** :bug: corrected dice roll mechanic

### Refactor
- improved readability
- removed unneeded code
- Started refactoring as specified in [#5](https://github.com/tionis/tsdr-api/issues/5) [#6](https://github.com/tionis/tsdr-api/issues/6) and [#3](https://github.com/tionis/tsdr-api/issues/3)
- refactored cache and removed redis dependency
- refactored code for more explicit error handling and more
- removed unneeded code
- improved readability
- some general repo cleanup
- updated dependencies
- **GlyphDiscordBot:** improved logging code
- **UniPassauBot:** imported uni passau bot as dependency
- **UniPassauBot:** :construction: improved some of the logic behind the bot


[Unreleased]: https://github.com/tionis/tsdr-api/compare/v0.0.1...HEAD
