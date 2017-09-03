#encoding: utf-8
# Documentation goes here.
module Cartero
module Commands
# Documentation for Listener < ::Cartero::Command
class Listener < ::Cartero::Command

  description(
    name: "Cartero Customizeable Web Server",
    description: "Listener is the command that serves our customized WebServers " +
                 "hosting pages on one or more ports at any given time.",
    author: ["Matias P. Brutti <matias [©] section9labs.com>"],
    type: "Infrastructure",
    license: "LGPL",
    references: ["https://section9labs.github.io/Cartero"]
  )

  def initialize
    super do |opts|
      opts.on("-i", "--ip 1.1.1.1", String,
        "Sets IP interface, default is 0.0.0.0") do |ip|
        @options.ip = ip
      end

      opts.on("-p", "--ports PORT_1,PORT_2,..,PORT_N", String,
        "Sets Email Payload Ports to scan") do |p|
        @options.ports = p.split(",").map(&:to_i)
      end

      opts.on("-s", "--ssl", "Run over SSL. [this also requires --sslcert and --sslkey]") do
        @options.ssl = true
      end

      opts.on("-C", "--sslcert CERT_PATH", String,
        "Sets SSL cert to use for Listener") do |cert|
        @options.sslcert = cert
      end

      opts.on("-K", "--sslkey KEY_PATH", String,
        "Sets SSL key to use for Listener.") do |key|
        @options.sslkey = key
      end

      opts.on("-V", "--views VIEWS_FOLDER", String,
        "Sets SSL Certificate to use for Listener.") do |file|
        @options.views = file
      end

      opts.on("-P", "--public PUBLIC_FOLDER", String,
        "Sets a Sinatra public_folder") do |file|
        @options.public_folder = file
      end

      opts.on("-W", "--webserver WEBSERVER_FOLDER", String,
        "Sets the sinatra full path from cloner.") do |path|
        app = path.split("/")[-1]
        @options.customwebserver = path + "/" + app + ".rb"
        @options.views = path + "/views"
        @options.public_folder = path + "/static"
      end

      opts.on("--payload PAYLOAD", String,
        "Sets a payload download to serve on /download") do |file|
        @options.payload = file
      end

      opts.on("-m","--metasploit", "Enable metasploit integration for hosts,creds,clients") do
        @options.metasploit = true
      end

      opts.on("--customapp CUSTOM_SINATRA", String,
        "Sets a custom Sinatra::Base WebApp. Important, WebApp name should be camelized of filename") do |app|
        @options.customwebserver = app
      end
    end
  end

  def setup
    require 'cartero/models'
    require 'cartero/user_agent_parser'
    require 'cartero/puma_cartero_cli'
    require 'cartero/sinatra_helpers'
    require 'geocoder'

    if @options.ports.nil?
      raise StandardError, "No Port Provided [--ports]. Need to provide at least one binding port."
    end

    @puma = Puma::CarteroCLI.new([])
		@puma.options[:environment] = 'production'
    @puma.options[:min_threads] = 4
    @puma.options[:max_threads] = 16
    @puma.options[:quiet] = true
    #@puma.options[:workers] = 4
    # Set Another Listening IP interface.
    @ip = @options.ip || "0.0.0.0"

    require "ipaddress"
    if !IPAddress.valid? @ip
      raise StandardError, "IP provided is not a valid address."
    end

    # Setting Default / Custom Sinatra Server.
    if @options.customwebserver.nil?
      @web_server = WebServer
    else
      if File.exist?(File.expand_path(@options.customwebserver))
        # Load library from file.
        load "#{File.expand_path(@options.customwebserver)}"
        # It is key that the name of the file reflect an uncamelize version of the library.
        # i.e. test_web_server.rb should be class TestWebServer
        @web_server = Module.const_get(File.basename(@options.customwebserver).split(".")[0..-2].join(".").camelize)
      else
        raise StandardError, "Custom WebServer file does not exists."
      end
    end

    # This is a constant given we use PUMA. So far no other option.
    @web_server.set :server, :puma
    @web_server.configure do
      @options.mongodb.nil? ? m = "localhost:27017" : m = @options.mongodb
      Mongoid.configure do |config|
        config.sessions = { 
          :default => {
            :hosts => [m], 
            :database => "Cartero"
          }
        }
      end
    end

    # set webserver verbosity and/or debug state
    @web_server.set :verbose, @options.verbose
    @web_server.set :debug, @options.debug

    # Allow to pass a Views Render Folder Path
    unless @options.views.nil?
      if File.directory?(File.expand_path(@options.views))
        @web_server.set :views, File.expand_path(@options.views)
      else
        raise StandardError, "View Folder does not exists."
      end
    end

    # Allow to pass a Public Folder Path
    unless @options.public_folder.nil?
      if File.directory?(File.expand_path(@options.public_folder))
        @web_server.set :public_folder, File.expand_path(@options.public_folder)
      else
        raise StandardError, "Public_folder Folder does not exists."
      end
    end

    # Allow to pass a payload for /download?key=XXXXXX
    unless @options.payload.nil?
      if File.exist?(File.expand_path(@options.payload))
        @web_server.set :payload_path, File.expand_path(@options.payload)
      else
        raise StandardError, " #{File.expand_path(@options.payload)} Payload does not exists."
      end
    end

    if !@options.metasploit.nil?
      require 'cartero/metasploit'
      msf = ::Cartero::Metasploit.new(::Cartero::GlobalConfig["metasploit"])
      l = msf.login
      if msf.token && l == true
        puts "Successfully Connected to msfrpcd"
      else
        raise StandardError, "Cannot connect to Metasploit Check your msfrpcd server or configuration."
      end
      if msf.db_connect["result"] == "success"
        puts "Successfully Connected to metasploit database"
      else
        raise StandardError, "Cannot connect to Metasploit database"
      end
      @web_server.set :metasploit, msf
    else
      @web_server.set :metasploit, nil
    end

    # Passing PUMA the Sinatra WebApp we will be using.
    @puma.options[:app] = @web_server

    # Handling SSL Options in Advance.
    # Handling Also port inside here, to ensure that
    # if none provided 443 || 80 are correctly provided.
    if @options.ssl.nil?
      @ports = @options.ports || [80]
    else
      @ports = @options.ports || [443]
      raise StandardError, "WebServer on SSL mode needs a cert path [ --sslcert ]." if @options.sslcert.nil?
      @ssl_cert_path = File.expand_path(@options.sslcert)
      raise StandardError, "WebServer on SSL mode needs a key path.[ --sslkey ]" if @options.sslkey.nil?
      @ssl_key_path = File.expand_path(@options.sslkey)
    end

    # Generating Bind/s for each provided port.
    binds = []
    @options.ports.each do |p|
      if !@options.ssl.nil?
        binds << "ssl://#{@ip}:#{p}?key=#{@ssl_key_path}&cert=#{@ssl_cert_path}"
      else
        binds << "tcp://#{@ip}:#{p}"
      end
    end
    @puma.options[:binds] = binds

    Geocoder.configure(:timeout => 6)
  end

  attr_accessor :ip
  attr_accessor :ports
  attr_accessor :ssl_key_path
  attr_accessor :ssl_cert_path
  attr_accessor :web_server

  def run
    @puma.run
  end
end
end

require 'rack'
require 'sinatra'
require 'sinatra/cookies'
require 'cartero/sinatra_helpers'

# Documentation for WebServer < Sinatra::Base
class WebServer < Sinatra::Base
  helpers Sinatra::Cookies
  helpers ::Cartero::SinatraHelpers
  register ::Cartero::CrawlerBlock

  block_and_redirect("http://example.com")

  not_found do
    puts "#{Time.now} - IP #{request.ip} PORT #{request.port} - #{request.forwarded?} - #{request.user_agent}"
    erb :error
  end

  get "/image" do
    process_info(params,request)
    return_img
  end

  get "/download" do
    process_info(params, request)
    return_payload
  end

  get "/click" do
    process_info(params,request)
    return_img
  end
end
end
