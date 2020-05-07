### Description
A robust Phishing Framework with a full featured CLI interface. The project was born out necessity through of years of engagements with tools that just didn't do the job. Even though there are many projects out there, we were not able to find a suitable solution that gave us both easy of use and customizability.

Cartero is a modular project divided into commands that perform independent tasks (i.e. Mailer, Cloner, Listener, AdminConsole, etc...). In addition each sub-command has repeatable configuration options to configure and automate your work.

For example, if we wanted to clone gmail.com, we simply have to perform the following commands.

```shell
❯❯❯ ./cartero Cloner --url https://gmail.com --path /tmp --webserver gmail_com
❯❯❯ ./cartero Listener --webserver /tmp/gmail_com -p 80
Launching mongodb
Puma starting in single mode...
* Version 2.8.2 (ruby 2.1.1-p76), codename: Sir Edmund Percival Hillary
* Min threads: 4, max threads: 16
* Environment: production
* Listening on tcp://0.0.0.0:80
Use Ctrl-C to stop
```

Once we have a site up and running we can simply use the Mailer command to send templated emails to our victims:

```shell
❯❯❯ ./cartero Mailer --data victims.json --server gmail_com --subject "Internal Memo" --htmlbody email_html.html --attachment payload.pdf --from "John Doe <jdoe@company.com>"
Sending victim1@company.com
Sending victim2@company.com
Sending victim3@company.com

```

### Installation

##### Automated Installation

Using brew 2.1.5 ruby as default ruby library
```shell
❯❯❯ curl -L https://raw.githubusercontent.com/Section9Labs/Cartero/master/data/scripts/setup.sh | bash
```
Using RVM 2.1.5 ruby installation
```shell
❯❯❯ curl -L https://raw.githubusercontent.com/Section9Labs/Cartero/master/data/scripts/setup.sh | bash -s -- -r
```
#### Dependencies
##### Ruby
```shell
❯❯❯ \curl -sSL https://get.rvm.io | bash -s stable --ruby
```

##### MongoDB
On OSX:
```shell
❯❯❯ brew install mongodb
```

On Ubuntu / Kali / Debian
```shell
❯❯❯ apt-get install mongodb
```

On Arch Linux
```
❯❯❯ pacman -Syu mongodb
```

#### Framework
```shell
❯❯❯ git clone https://github.com/section9labs/Cartero
❯❯❯ cd Cartero
❯❯❯ gem install bundle
❯❯❯ bundle install
❯❯❯ cd bin
```

### Usage
### Commands
Cartero is a very powerful easy to use CLI.

```shell
❯❯❯ ./cartero
Usage: cartero [options]

List of Commands:
    AdminConsole, AdminWeb, Mailer, Cloner, Listener, Servers, Templates

Global options:
        --proxy [HOST:PORT]          Sets TCPSocket Proxy server
    -c, --config [CONFIG_FILE]       Provide a different cartero config file
    -v, --[no-]verbose               Run verbosely
    -p [PORT_1,PORT_2,..,PORT_N],    Global Flag fo Mailer and Webserver ports
        --ports
    -m, --mongodb [HOST:PORT]        Global Flag fo Mailer and Webserver ports
    -d, --debug                      Sets debug flag on/off
        --editor [EDITOR]            Edit Server


Common options:
    -h, --help [COMMAND]             Show this message
        --list-commands              Prints list of commands for bash completion
        --version                    Shows cartero CLI version
```

### Basic Commands

#### Mongo
This is a simple Wrapper for MongoDB that allows us to start stop the database with the corresponding commands and on the correct ~/.cartero path.

```shell
❯❯❯ ./cartero Mongo
Usage: Cartero Mongo [options]
    -s, --start                      Start MongoDB
    -k, --stop                       Stop MongoDB
    -r, --restart                    Restart MongoDB
    -b, --bind [HOST:PORT]           Set MongoDB bind_ip and port

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options
```

#### Cloner
A WebSite Cloner that allows us to download and convert a website into a Cartero WebServer application.
We can quickly and easily customize the website to Harvest Credentials, Server Payloads, or fully modify the site for any number of purposes.

```shell
❯❯❯ ./cartero Cloner
Usage: Cartero Cloner [options]
    -U, --url [URL_PATH]             Full Path of site to clone
    -W, --webserver [SERVER_NAME]    Sets WebServer name to use
    -p, --path [PATH]                Sets path to save webserver
    -P, --payload [PAYLOAD_PATH]     Sets payload path
        --useragent [UA_STRING]      Sets user agent for cloning
        --wget                       Use wget to clone url
        --apache                     Generate Apache Proxy conf

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options
```
By default the command uses our Ruby implementation to download and convert links to render, but we also support a *--wget* option that will use the local wget system command.

#### Listener
The listener is responsible for running the WebServer created through Cloner or a manually created site. By default we present a very simple website if none is provided.

```shell
❯❯❯ ./cartero Listener
Usage: Cartero Listener [options]
    -i, --ip [1.1.1.1]               Sets IP interface, default is 0.0.0.0
    -p [PORT_1,PORT_2,..,PORT_N],    Sets Email Payload Ports to scan
        --ports
    -s, --ssl                        Run over SSL. [this also requires --sslcert and --sslkey]
    -C, --sslcert [CERT_PATH]        Sets Email Payload Ports to scan
    -K, --sslkey [KEY_PATH]          Sets SSL key to use for Listener.
    -V, --views [VIEWS_FOLDER]       Sets SSL Certificate to use for Listener.
    -P, --public [PUBLIC_FOLDER]     Sets a Sinatra public_folder
    -W [WEBSERVER_FOLDER],           Sets the sinatra full path from cloner.
        --webserver
        --payload [PAYLOAD]          Sets a payload download to serve on /download
        --customapp [CUSTOM_SINATRA] Sets a custom Sinatra::Base WebApp. Important, WebApp name should be camelized of filename

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options

```
The WebServers support ssl keys and virtual hosts across multiple IP, Hostnames, and Ports.

#### Servers
In order to send emails campaigns we need to setup email servers and this command allows Cartero to create, store and list servers. All data is stored in the ~/.cartero configuration directory.

```shell
./cartero Servers
Usage: Cartero Servers [options]
    -a, --add [NAME]                 Add Server
    -e, --edit [NAME]                Edit Server
    -d, --delete [NAME]              Edit Server
    -l, --list                       List servers

Configuration options:
    -T, --type [TYPE]                Set the type
    -U, --url [DOMAIN]               Set the Mail or WebMail url/address
    -M, --method [METHOD]            Sets the WebMail Request Method to use [GET|POST]
        --api-access [API_KEY]       Sets the Linkedin API Access Key
        --api-secret [API_SECRET]    Sets the Linkedin API Secret Key
        --oauth-token [OAUTH_TOKEN]  Sets the Linkedin OAuth Token Key
        --oauth-secret [OAUTH_SECRET]
                                     Sets the Linkedin OAuth Secret Key

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options
```

#### Templates
Just like Servers, email campaigns also need a pre-defined Template for sending content to the victims. This module allows the attacker to keep track, create, list, and edit templates being used in their campaign.

Note: Setting templates here is not necessary and Mailer accepts a direct path to emails templates from the CLI.

```shell
❯❯❯ ./cartero Templates
Usage: Cartero Templates [options]
    -a, --add [NAME]                 Add Template
    -e, --edit [NAME]                Edit Template
    -d, --delete [NAME]              Edit Template
    -l, --list                       List Templates
    -h, --help                       Show this message
```
#### Mailer
THe main command and component in the Cartero Framework -- It allows Cartero to send custom templated emails to one or more email addresses.

Each email can be customized using the powerful erb Template engine, allowing users to create complex programmatic rules within the templates to send massive amounts of very targeted emails.

For more information on how to build custom templates, please refer to our Examples.

```shell
❯❯❯ ./cartero Mailer
Usage: Cartero Mailer [options]
    -D, --data [DATA_FILE]           File containing template data sets
    -S, --server [SERVER_NAME]       Sets Email server to use
    -s, --subject [EMAIL_SUBJECT]    Sets Email subject
    -f, --from [EMAIL_FROM]          Sets Email from
    -r, --reply-to [EMAIL_REPLY_TO]  Sets Email from
    -b, --body [FILE_PATH]           Sets Email Text Body
    -B, --htmlbody [FILE_PATH]       Sets Email HTML Body
    -c, --charset [CHARSET]          Sets Email charset
    -C [CONTENT_TYPE],               Sets Email content type
        --content-type
    -a [FILE_1,FILE_2,..,FILE_N],    Sets Email Attachments
        --attachment
    -p [PORT_1,PORT_2,..,PORT_N],    Sets Email Payload Ports to scan
        --ports

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options
```
#### WebMailer
This command supports an alternative to SMTP / IMAP servers through send messages using known vulnerable or anonymous webmail services via web requests.

```shell
❯❯❯ ./cartero WebMailer
Usage: Cartero WebMailer [options]
    -R, --raw [RAW_REQUEST_FILE]     Sets WebMail Raw Request
    -S, --server [SERVER_NAME]       Sets WebMail server to use
    -U, --url [URL:PORT]             Sets WebMail server url to use
    -H [HEADER:VAL\nHEADER:VAL],     Sets WebMail Headers to use
        --headers
    -C, --cookies [COOKIES]          Sets WebMail Cookies to use
    -D, --data [DATA_FILE]           File containing template data sets
    -s, --subject [EMAIL_SUBJECT]    Sets Email subject
    -f, --from [EMAIL_FROM]          Sets Email from
    -r, --reply-to [EMAIL_REPLY_TO]  Sets Email reply-to
    -b, --body [REQUEST_FILE_PATH]   Sets Email Text request query Body
    -p [PORT_1,PORT_2,..,PORT_N],    Sets Email Payload Ports to scan
        --ports

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options

```
The Command can be used in two main ways. One using a raw command such as the one we get from Intercepting traffic with a web proxy (i.e. Burp Proxy) and or using the servers command available on Cartero.

#####Example of webmail server for send-mail.org
```json
{
    "name": "send-email",
    "type": "webmail",
    "options": {
        "url": "http://send-email.org/send",
        "method": "POST",
        "cookies": "",
        "headers": {
            "Host": "send-email.org",
            "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:29.0) Gecko/20100101 Firefox/29.0",
            "Accept": "application/json, text/javascript, */*; q=0.01",
            "Accept-Language": "en-US,en;q=0.5",
            "Accept-Encoding": "gzip, deflate",
            "Content-Type": "application/x-www-form-urlencoded; charset=UTF-8",
            "X-Requested-With": "XMLHttpRequest",
            "Referer": "http://send-email.org/",
            "Content-Length": "126",
            "Connection": "keep-alive",
            "Pragma": "no-cache"
        }
    },
    "confirmation" : "Your message was sent!"
}
```

Using this pre-configured request, we can easily send message using the same datasets for Mailer and using the same type of templates. An example is available in /templates/mail/sample.web

#####Sample Command:
```shell
❯❯❯ ./cartero WebMailer -S webmail -D ~/sample.json -b ../templates/mail/sample.web -r cartero@gmail.com
```
#### LinkedIn
The LinkedIn command is the first Social Network addition to the Cartero Framework. This plugin allows attackers to use the social platform to send messages and attack users all from within LinkedIn.
```shell
❯❯❯ ./cartero LinkedIn
Usage: Cartero LinkedIn [options]
    -D, --data [DATA_FILE]           File containing template data sets
    -S, --server [SERVER_NAME]       Sets Email server to use
    -s, --subject [MESSAGE_SUBJECT]  Sets LinkedIn Message subject
    -b, --body [FILE_PATH]           Sets LinkedIn Message Body
    -l, --list [CONNECTIONS|GROUPS]  List json of (connections or groups)
        --send [MESSAGE|GROUP_UPDATE]
                                     Send one or more (message/s or group/s updates)
    -o, --save [FILE_PATH]           Sets LinkedIn Message Body

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options
```

The command requires a developer API and oauth key on the attackers profile. This can be easily obtained in https://www.linkedin.com/secure/developer. and a new server template can be created with the Server command.

#### IMessage
Allows Cartero, on OS X, to send iMessages to victims addresses just like emails, but these will show up on every iDevice they have registered with apple.

Background: After reading some news on how Chinese spammers are abusing the fact that iMessage messages will be displayed on all devices linked to the account/s, a quick command was developer to allow Cartero users to also have this feature available to the Framework.

_Important: This will only work on OSX, for now_

```shell
❯❯❯ ./cartero IMessage
Usage: Cartero IMessage [options]
IMPORTANT: This command only works on OSX

    -D, --data [DATA_FILE]           File containing template data sets
    -A, --attachment [ATTACHMENT]    Sets iMessage file path to send
    -b, --body [BODY_FILE]           Sets iMessage message
    -m, --message [MESSAGE]          Sets iMessage message

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options
```
iMessage does not allow sending a single message containing both text and attachments, but when both are provided the message will be split and sent as two individual messages.
#####Sample command
```shell
❯❯❯ ./cartero IMessage --data /Users/cartero/Desktop/test.json -b ../templates/mail/sample.imsg -a /Users/cartero/Downloads/jon.jpg
```
#### GoogleVoice
If you have a GoogleVoice account and want to automatically send SMS this might be a nice way to do it. It follows the same infrastructure as all other commands.

````shell
❯❯❯ ./cartero GoogleVoice
Usage: Cartero GoogleVoice [options]
    -D, --data [DATA_FILE]           File containing template data sets
    -S, --server [SERVER_NAME]       Sets SMS server to use
    -b, --body [FILE_PATH]           Sets SMS Text Body
    -m, --message [MESSAGE]          Sets SMS message
    -u, --username [USER]            Sets Google Voice Username
    -p, --password [PWD]             Sets Google Voice password

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options

````

#### Twilio
If you have a Twilio account and want to automatically send SMS this might yet another way to send SMS. It is worth noting this is a paid service and it requires a valid access token (sid) and secret (secret_token). For more information please refer to Twilio's website at https://www.twilio.com/sms/api

````shell
❯❯❯ ./cartero Twilio
Usage: Cartero Twilio [options]
    -D, --data [DATA_FILE]           File containing template data sets
    -S, --server [SERVER_NAME]       Sets SMS server to use
    -f, --from [NUMBER]              Sets SMS from number to use
    -b, --body [FILE_PATH]           Sets SMS Text Body
    -m, --message [MESSAGE]          Sets SMS message
    -u, --sid [SID]                  Sets Twilio Username
    -p, --token [TOKEN]              Sets Twilio password
    -A, --attachment [PATH_1||PATH_2||PATH_3]    Sets Twilio MMS URL image paths to send

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options

````


#### AdminWeb
The Admin Web interface is a simple Web-Application that allows the attacker to retrieve information about all Campaigns.

```shell
❯❯❯ ./cartero AdminConsole
Usage: Cartero AdminConsole [options]
    -i, --ip [1.1.1.1]               Sets IP interface, default is 0.0.0.0
    -p [PORT_1,PORT_2,..,PORT_N],    Sets Email Payload Ports to scan
        --ports
    -s, --ssl                        Run over SSL. [this also requires --sslcert and --sslkey]
    -C, --sslcert [CERT_PATH]        Sets Email Payload Ports to scan
    -K, --sslkey [KEY_PATH]          Sets Email Payload Ports to scan

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options
```

#### AdminConsole
The Admin Console is CLI tool that allows to list information about Persons, Campaigns, Hits and Credentials gathered.
```shell
❯❯❯ ./cartero AdminConsole
Usage: Cartero AdminConsole [options]
    -p, --persons [LATEST_N]         Display the list of persons that responded
    -i, --hits [LATEST_N]            Display the list of hits
    -c, --creds [LATEST_N]           Display the list of Credentials
    -a, --all                        Sets Email Payload Ports to scan
    -f, --filter                     flag to search by parameters
        --email [EMAIL]              Display the list of hits
        --campaign [CAMPAIGN]        Display the list of hits
        --ip [IP_ADDRESS]            Display the list of hits

Common options:
    -h, --help                       Show this message
        --list-options               Show list of available options

```

### Building Commands

Commands have a fairly simple framework. Example Framework commands are stored in _Cartero/lib/cartero/commands/*.rb_ and _~/.cartero/commands/*.rb_.

```ruby
module Cartero
module Commands
class CommandName < ::Cartero::Command
	def initialize
		super(
          name: "Long Command Name Here",
          description: "Lorem ipsum dolor sit amet, consectetur adipisicing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
          author: ["Author Name <noname [©] cartero.com>"],
          type:"General",
          license: "LGPL",
          references: [
            "https://section9labs.github.io/Cartero",
            "https://section9labs.github.io/Cartero"
            ]

    ) do |opts|
      #OptionsParser options available here.
			opts.on("-x", "--xoption [DATA_FILE]", String,
    		"Description of command option") do |data|
      	@options.xoption = data
    	end
      opts.on("-y", "--yoption [DATA_FILE]", String,
        "Description of command option") do |data|
        @options.xoption = data
      end
      # help() option already provided.
      # --list-options for auto-complete automatic.
    end
  end

  def setup
    # This will be hooked and run before run().
    # It is meant as a method so the commands can run everything before that.
  end

  def run
    # Everything that will run.
  end

  def after
    # This is the place to run clean-up code.
  end
end
end
end
```


### Mailer Templates
Emails are simple txt files or limited HTML files, depending on the format being used. Cartero allows complex templating through the erb ruby library and a CLI interface to help build and template both formats.

Files can be extensively customized for the campaign through erb which provides dynamic substitution and programmatic decisions at run time.

Another important feature is the encrypted self[:payload] that should be added in each email template which allows Cartero to identify the source entity regardless of multiple forwards or clicks. This small payload is encrypted using a randomly generated key allowing attackers to keep their source data secure and away from detection.

**SAMPLE DATA FILE**
```json
[{
    "email": "johndoe@gmail.com",
    "name": "John Doe"
}, {
    "email": "gh@gmail.com",
    "name": "Gas Hill"
}, {
    "email": "johndoe@hotmail.com",
    "name": "John Doe 2"
}, {
    "email": "janedoe@hotmail.com",
    "name": "Jane Doe - Hotmail",
    "subject": "Hotmail Test 123"
}]
```
**HTML SAMPLE TEMPLATE**
```html
<html>
<body>
<h3> Hello Spear Phishing World <%= self[:name] %>,</h3>
<p> This is an automated email to your email <%= self[:email] %>.<p>
<% if self[:ports] %>
	<% self[:ports].each do |port| %>
	    <img alt="" width="1" height="1" border="0" style="height:1px !important;width:1px !important;border-width:0 !important;margin-top:0 !important;margin-bottom:0 !important;margin-right:0 !important;margin-left:0 !important;padding-top:0 !important;padding-bottom:0 !important;padding-right:0 !important;padding-left:0 !important;" src="http://localhost:<%= port %>/image?key=<%= self[:payload] %>">
	<% end %>
<% end %>
</body>
</html>

```

**TEXT SAMPLE TEMPLATE**
```txt
Hola <%= self[:name] %>,
This email needs to be displayed as HTML.
This is an automated email to your email<%= self[:email] %>.
In addition, this email can also be displayed securely
on http://192.168.1.216:8080/click?key=<%= self[:payload] %>

cheers,

<%= self[:from_name] %>

```

### Servers
Servers can be managed using the Servers command, which provides the ability to add, edit and delete servers.
Note: Servers can be manually edited in _~/.cartero/servers/*.json_

####smtp
```json
{
    "name": "gmail",
    "type": "smtp",
    "options": {
        "address": "smtp.yourserver.com",
        "port": 25,
        "user_name": "user",
        "password": "password",
        "authentication": "plain",
        "domain": "localhost.localdomain"
    }
}
```

####linkedin
```json
{
  "name": "linkedin",
  "type": "linkedin",
	"options": {
	  "api_access": "api_access",
	  "api_secret": "api_secret",
	  "oauth_token": "oauth_token",
	  "oauth_secret": "oauth_secret"
	}
}
```

####webmail
```json
{
  "name": "webmail-sample",
  "type": "webmail",
  "options": {
    "url": "http://www.send-email.com/data/send/email",
    "method": "POST",
    "cookies": "sdajsda09s7das923i3j2l131;21381903810",
    "headers": {
      "x-forward": "asdadadasad"
    }
  },
  "confirmation" : null
}
```