#encoding: utf-8
module Cartero
module Commands
# Documentation for Mailer < ::Cartero::Commands
class Mailer < Command

  description(
    name: "Customized Mass Email Command",
    description: "Mailer is responsible for crafting and sending emails from a simple txt based email all the way to complicated email templates that can individualize each email as if it was being written by a person.",
    author: ["Matias P. Brutti <matias [©] section9labs.com>"],
    type: "Delivery",
    license: "LGPL",
    references: ["https://section9labs.github.io/Cartero"]
  )

  def initialize
    super do |opts|
      opts.on("-D", "--data DATA_FILE", String,
        "File containing template data sets") do |data|
        @options.data = data
      end

      opts.on("-S", "--server SERVER_NAME", String,
        "Sets Email server to use") do |server|
        @options.server = server
      end

      opts.on("-s", "--subject EMAIL_SUBJECT", String,
        "Sets Email subject") do |subject|
        @options.subject = subject
      end

      opts.on("-f", "--from EMAIL_FROM", String,
        "Sets Email from") do |from|
        @options.from = from
      end

      opts.on("-r", "--reply-to EMAIL_REPLY_TO", String,
        "Sets Email reply-to") do |reply_to|
        @options.reply_to = reply_to
      end

      opts.on("-b", "--body FILE_PATH", String,
        "Sets Email Text Body") do |body|
        @options.body = body
      end

      opts.on("-B", "--htmlbody FILE_PATH", String,
        "Sets Email HTML Body") do |body|
        @options.html_body = body
      end

      opts.on("-c", "--charset CHARSET", String,
        "Sets Email charset") do |charset|
        @options.charset = charset
      end

      opts.on("-C", "--content-type [CONTENT_TYPE]", String,
        "Sets Email content type") do |charset|
        @options.charset = charset
      end

      opts.on("-a", "--attachment FILE_1,FILE_2,..,FILE_N", String,
        "Sets Email Attachments") do |attach|
        @options.attachments = attach.split(",")
      end

      opts.on("-p", "--ports PORT_1,PORT_2,..,PORT_N", String,
        "Sets Email Payload Ports to scan") do |p|
        @options.ports = p.split(",").map(&:to_i)
      end
    end
  end

  def setup
    require 'erb'
    require 'pony'

    if @options.data.nil?
      raise StandardError, "A data set [--data] must be provided"
    end

    if @options.body.nil? && @options.html_body.nil?
      raise StandardError, "At least a body [--body] and/or html_body [--htmlbody] must be provided"
    end

    if @options.server.nil?
      @server = {:name => "default", :type => :sendmail }
    elsif !Cartero::Commands::Servers.exists?(@options.server)
      raise StandardError, "Server with name #{@options.server} does not exist."
    else
      s = ::Cartero::Commands::Servers.server(@options.server)
      @server = JSON.parse(File.read(s),{:symbolize_names => true})
    end

    @data					= @options.data
    @from 				= @options.from
    @reply_to 		= @options.reply_to
    @subject 			= @options.subject

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

    unless @options.html_body.nil?
      if ::Cartero::Commands::Templates.exists?(@options.html_body)
        @html_body = File.read("#{Cartero::TemplatesDir}/#{@options.html_body}.erb")
      else
        if File.exist?(File.expand_path @options.html_body)
          @html_body = File.read(File.expand_path @options.html_body)
        else
          raise StandardError, "HTML Body Template (#{File.expand_path @options.html_body}) does not exists"
        end
      end
    end

    # Content-Types, Charsets and Extra Headers
    @content_type = @options.content_type
    @charset 			= @options.charset
    @headers 			= @options.headers

    @attachments 	= {}
    unless @options.attachments.nil?
      @options.attachments.each do |attach|
        @attachments[File.basename(attach)] = File.read(File.expand_path attach)
      end
    end
    @data = JSON.parse(File.read(File.expand_path @options.data),{:symbolize_names => true})
    @ports = @options.ports || []
  end

  attr_reader :attachments
  attr_reader :data
  attr_reader :server
  attr_reader :to
  attr_reader :from
  attr_reader :reply_to
  attr_reader :subject
  attr_reader :content_type
  attr_reader :charset
  attr_reader :headers
  attr_reader :body
  attr_reader :html_body
  attr_reader :ports

  def send
    data.each do |entity|
      if !entity[:email].nil?
        yield entity if block_given?
        create_email(entity)
      else
        Cartelo::Log.error "Entity #{entity} does not contain an :email key."
      end
    end
  end

  def run
    send do |s|
      puts "Sending #{s[:email]}"
    end
  end

  def create_email(entity)
    mail = {}
    # Set server configuration
    mail[:via] 				 = server[:type].to_sym
    mail[:via_options] = server[:options] if server[:options]

    # set TO, FROM and Subject
    mail[:to] 			= entity[:email]
    mail[:from]			= entity[:from] 		|| from

    if entity[:reply_to] || reply_to
      mail[:reply_to] = entity[:reply_to] || reply_to
    end

    mail[:subject]	= entity[:subject] 	|| subject

    # Set Content-Type, if provided
    unless content_type.nil?
      mail[:content_type] = content_type
    end

    # Set Charset, if provided
    unless charset.nil?
      mail[:charset] = charset
    end

    unless ports.empty?
      entity[:ports] = ports
    end

		entity[:subject] ||= subject
    entity[:from] ||= from
    entity[:payload] = ::Cartero::CryptoBox.encrypt(entity.to_json)

    # Add Text body if was provided.
    unless body.nil?
      mail[:body] = ERB.new(body).result(entity.get_binding)
      mail[:body_part_header] = { content_disposition: "inline" }
    end

    # Add HTML Body if was provided.
    unless html_body.nil?
      mail[:html_body] = ERB.new(html_body).result(entity.get_binding)
      mail[:html_body_part_header] = { content_disposition: "inline" }
    end

    # Add Attachment/s if was/were provided.
    unless attachments.empty?
      mail[:attachments] = attachments
    end
    # Actually send email using Pony Library.
    Pony.mail(mail)
  end
end
end
end
