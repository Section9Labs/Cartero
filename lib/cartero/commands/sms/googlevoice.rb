#encoding: utf-8
module Cartero
module Commands
# Documentation for GoogleVoice < ::Cartero::Command
class GoogleVoice < ::Cartero::Command

  description(
    name: "Google Voice SMS/MMS Text Messages Command",
    description: "Using Google Voice as a delivery method, an attacker can send multiple individually crafted text messages.",
    author: ["Matias P. Brutti <matias [©] section9labs.com>"],
    type: "Delivery",
    license: "LGPL",
    references: [
      "https://section9labs.github.io/Cartero",
      "http://voice.google.com"
      ]
  )

  def initialize
    super do |opts|
      opts.on("-D", "--data DATA_FILE", String,
        "File containing template data sets") do |data|
        @options.data = data
      end

      opts.on("-S", "--server SERVER_NAME", String,
        "Sets SMS server to use") do |server|
        @options.server = server
      end

      opts.on("-b", "--body FILE_PATH", String,
        "Sets SMS Text Body") do |body|
        @options.body = body
      end

      opts.on("-m", "--message MESSAGE", String,
        "Sets SMS message") do |message|
        @options.message = message
      end

      opts.on("-u", "--username USER", String,
        "Sets Google Voice Username") do |u|
        @options.username = u
      end

      opts.on("-p", "--password PWD", String,
        "Sets Google Voice password") do |p|
        @options.password = p
      end
    end
  end

  def setup
    Puts "[!] - Temporareily disabled until a new ruby gem is found. googlevoiceapi no longer exists."
    exit(1)
    require 'erb'
    require 'googlevoiceapi'

    if @options.data.nil?
      raise StandardError, "A data set [--data] must be provided"
    end

    if @options.body.nil? && @options.message.nil?
      raise StandardError, "At least a body [--body] or a straight message [--message] must be provided"
    end

    if @options.server.nil? && @options.username.nil? && @options.password.nil?
      raise StandardError, "A Google Voice Server Credentials should be provided."
    elsif !Cartero::Commands::Servers.exists?(@options.server)
      raise StandardError, "Server with name #{@options.server} does not exist."
    else
      s = ::Cartero::Commands::Servers.server(@options.server)
      @server = JSON.parse(File.read(s),{:symbolize_names => true})

      if @server[:type].downcase != "gvoice"
        raise StandardError, "Server with name #{@options.server} is not gvoice type."
      end
    end

    unless @options.data.nil?
      @data = JSON.parse(File.read(File.expand_path @options.data),{:symbolize_names => true})
    end

    @username 		= @options.username
    @password 		= @options.password

    @gapi = ::GoogleVoice::Api.new(username || server[:options][:username], password || server[:options][:password])

    unless @options.body.nil?
      if ::Cartero::Commands::Templates.exists?(@options.body)
        @body = File.read("#{Cartero::TemplatesDir}/#{@options.body}.erb")
      else
        if File.exist?(File.expand_path @options.body)
          @body = File.read(File.expand_path @options.body)
        else
          raise StandardError, "Text Body Template (#{File.expand_path @options.body}) does not exists"
        end
      end
    end

    @message = @options.message unless @options.message.nil?
  end

  attr_reader :data
  attr_reader :server
  attr_reader :username
  attr_reader :password
  attr_reader :message
  attr_reader :to
  attr_reader :body
  attr_reader :ports
  attr_reader :gapi

  def send
    data.each do |entity|
      if !entity[:phone].nil?
        yield entity if block_given?
        create_sms(entity)
      else
        Cartelo::Log.error "Entity #{entity} does not contain an :phone key."
      end
    end
  end

  def run
    send do |s|
      puts "Sending #{s[:phone]}"
    end
  end

  def create_sms(entity)
    sms = {}
    # set TO
    sms[:to] = entity[:phone]

    entity[:payload] = ::Cartero::CryptoBox.encrypt(entity.to_json)

    # Add Text body if was provided.
    if !body.nil?
      sms[:message] = ERB.new(body).result(entity.get_binding)
    elsif !message.nil?
      sms[:message] = message
    else
      raise StandardError, "Entity #{entity} does not contain a body or a message key."
    end

    gapi.sms(sms[:to], sms[:message])
  end
end
end
end
