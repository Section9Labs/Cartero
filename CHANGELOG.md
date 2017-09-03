## CHANGELOG

#### 12-00-14:
- Added metasploit support. Will not commit until metasploit accepts changes.
  UPDATE: 12/13/14 - Changes accepted #4142.

#### 2-10-15:
- added rubocop & fixed a bunch of things. Still plenty to go.
- Added Cartero::SinatraHelpers helpers to make webserver code cleaner and smaller.

#### 2-14-15:
- added Cartero::CrawlerBlock Register to block and redirect bots

#### 2-18-15
- Added first version of Cloner --reverse-proxy

#### 2-19-15
- Pushing version 0.4
- Uploaded non-stable changes to Github

#### 2-22-15
- Added _CarteroAutoComplete script for basic Bash AutoComplete.
- Fixed OptsParser language around [non]mandatory arguments.
- Fixed minor issues on install script and added autocomplete logic.
- Added --list-short-options to all commands.
- Added beef logic to AdminWeb & AdminWebServer
- Pushing version 0.4.1

#### 2-28-15
- CryptoBox AES support Implemented. ~/.cartero/config now has new option crypto which allows use to pick encryption methods.

#### 3-1-15
- Several Fixes for rubocop
- Code refactoring.
- Starting Code documentation. I need to remember how things work.

#### 3-3-15
- Commands::information for better understanding of commands.
  + Data for now will be --but not limited to-- name, description, author, type, license & references
  + Added option (--details) to commands that would correctly display information on commands.
  + Although the system supports all type of information fields. We do limit it the types of objects to Arrays of Strings or Strings.
- beef hook generation command

#### 3-10-15
- Beef & BeefConsole
- AES Implementation
- Bug Fixes
- Command & Payloads information sytem added. (--details)
- Pushed 0.5 BigBife mayor version.

#### 3-11-15
- Reducing timeouts on BeefApi
- Bug Fixes
- GeoCoder on WebServers by default

#### 3-12-15
- Adding GeoLocation to AdminWeb
- GeoLocation to AdminConsole
- AdminWeb & README.md

#### 3-15-15
- Adding first-run auto bundler installer and bundler requirements.
- Rubocop minor fixes
- CHANGELOG & TODO updates.
- Error on first run only due to lack of proper execution order of first-run methods.
- Fixing first-run auto-bundler to correctly contemplate Gemfile
- Bug Fixes & first-setup enhancements.

#### 3-16-15
- Adding API /api/* to AdminWeb

#### 3-18-15
- Testing Slack Notifications

#### 3-19-15
- default is to be verbose
- Added simple log() & log_debug() methods :-)
- Fixed templates and sinatra_helpers to reflect changes.
- Updating jquery to 2.1.3
- Updating bootstrap to 3.3.4

#### 3-23-15
- Adding some text surrounding MongoDB. This fixed #38 since we realized MongoMapper does require an updated version of MongoDB.

#### 3-24-15
- XMPP Commands
- Adding jabber server option.
- fixing a few bugs found on JSON templates :=)
- removed subject option from xmpp since it makes no sense any longer to have it.
- First Alpha version of Xmpp Command time to play with it.
- It subscribes users and sends them a template message.
- Follows same delivery methodology as any other sender.
- Fixing a few things that were wrong with the AES implementation to be fully backward compatible with old crypto model
- Changed version - 0.5.1 farfullero

#### 10-9-15
- smbrelayx.py support allowing remote shell attacks and replay attacks using new French Kiss Attack.

#### 9-4-17
- Migrating to mongoid, since mongomapper no longer works. 
- general small fixes. 

