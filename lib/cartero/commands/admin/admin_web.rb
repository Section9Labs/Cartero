#encoding: utf-8
# Documentation for Cartero
module Cartero
module Commands
# Documentation for AdminWeb < ::Cartero::Command
class AdminWeb < ::Cartero::Command

  description(
    name: "Cartero Administration Web Application",
    description: "Cartero WebApp based Admnistration Interface. It allows users to interact with the captured data (i.e. hits, persons, credentials.) using a friendly Web Application.",
    author: ["Matias P. Brutti <matias [©] section9labs.com>"],
    type: "Admin",
    license: "LGPL",
    references: ["https://section9labs.github.io/Cartero"]
  )
  
  def initialize
    super do |opts|
      opts.on("-i", "--ip 1.1.1.1", String,
        "Sets IP interface, default is 0.0.0.0") do |ip|
        @options.ip = ip
      end

      opts.on("-p", "--ports [PORT_1,PORT_2,..,PORT_N]", String,
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

      opts.on("-B", "--beef URL", String,
       "Sets Beef Framework UI URL") do |url|
        @options.beef = url
      end
    end
  end

  attr_accessor :ip
  attr_accessor :ports
  attr_accessor :ssl_key_path
  attr_accessor :ssl_cert_path
  attr_accessor :web_server

  def setup
    require 'cartero/models'
    require 'cartero/puma_cartero_cli'

    @puma = Puma::CarteroCLI.new([])
    @puma.options[:environment] = 'production'
    @puma.options[:min_threads] = 4
    @puma.options[:max_threads] = 16
    @puma.options[:quiet] = true
    #@puma.options[:workers] = 4
    # Set Another Listening IP interface.
    @ip = @options.ip || "127.0.0.1"

    require "ipaddress"
    if !IPAddress.valid? @ip
      raise StandardError, "IP provided is not a valid address."
    end

    @web_server = WebAdmin

    @web_server.set :server, :puma
    @web_server.set :views, File.expand_path("../../../../../data/web/admin/views", __FILE__)
    @web_server.set :public_folder, File.expand_path("../../../../../data/web/admin/static", __FILE__)

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

    @web_server.set :beef, @options.beef

    if @options.beef
      Rack::ReverseProxy.class_eval("def beef_url; \"#{@options.beef}\"; end")
      @web_server.use Rack::ReverseProxy do
        reverse_proxy /^\/stats\/beef\/?(.*)$/, "#{beef_url}/$1"
      end

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

    binds = []
    @options.ports.each do |p|
      if !@options.ssl.nil?
        binds << "ssl://#{@ip}:#{p}?key=#{@ssl_key_path}&cert=#{@ssl_cert_path}"
      else
        binds << "tcp://#{@ip}:#{p}"
      end
    end
    @puma.options[:binds] = binds
  end

  def run
    @puma.run
  end
end
end

require 'rack'
require 'sinatra'
require 'sinatra/json'
require 'csv'
require 'rack/reverse_proxy'

# Documentation for WebAdmin < Sinatra::Base
class WebAdmin < Sinatra::Base
  helpers Sinatra::JSON
  helpers do
    def h(text)
      Rack::Utils.escape_html(text)
    end

    def beef_enabled?
      settings.beef != nil
    end
  end

  get "/" do
    @persons = Person.all.size
    erb :index
  end

  get "/help" do
    erb :help
  end

  get "/stats/persons" do
    @persons = Person.all
    erb :stats
  end

  get "/stats/credentials" do
    @hits = Credential.all
    erb :stats_creds
  end

  get "/stats/hits" do
    @paths = Hit.distinct("path")
    @hits = Hit.all
    erb :stats_hits
  end

  get "/stats/hits.csv" do
    content_type "text/csv"
    @hits = Hit.all
    csv_string = CSV.generate do |csv|
      csv << ["Email", "Campaign", "IP", "Port", "Path", "Forwarded", "OS", "Browser", "Engine", "Platform", "Created"]
      @hits.each do |hit|
        csv << [hit.data['email'], hit.data['subject'],hit.ip ,hit.port, hit.path, hit.ua_os, hit.ua_browser, hit.ua_engine, hit.ua_platform, hit.created_at.strftime("%m/%d/%Y-%T")]
      end
    end
    return csv_string
  end

  post "/stats/search/:type" do
    case params[:type]
    when /persons/ then
      @persons = Person.all.select {|x| x.to_json.to_s =~ /#{params[:searchfield]}/i }
      erb :stats
    when /credentials/ then
      @hits = Credential.all.select {|x| x.to_json.to_s =~ /#{params[:searchfield]}/i }
      erb :stats_creds
    when /hits/ then
      @paths = Hit.distinct("path")
      @hits = Hit.all.select {|x| x.to_json.to_s =~ /#{params[:searchfield]}/i }
      erb :stats_hits
    else
      redirect back
    end
  end

  get "/stats/hits/campaign/:subject" do
    @paths = Hit.distinct("path")
    @hits = Hit.where.all.select {|x| x.data['subject'] =~ /#{params[:subject]}/i }
    erb :stats_hits
  end

  get "/stats/hits/email/:subject" do
    @paths = Hit.distinct("path")
    @hits = Hit.where(:email => params[:email]).first
    erb :stats_hits
  end

  get "/stats/person/:email" do
    @person = Person.where(:email => params[:email]).first
    erb :stats_person
  end

  #API Calls
  get "/api/persons" do
    json(Person.all, :encoder => :to_json, :content_type => :js)
  end

  get "/api/credentials" do
    json(Credential.all, :encoder => :to_json, :content_type => :js)
  end

  get "/api/hits" do
    json(Hit.all, :encoder => :to_json, :content_type => :js)
  end

  get "/api/hits/campaign/:subject" do
    json(Hit.where.all.select {|x| x.data['subject'] =~ /#{params[:subject]}/i }, :encoder => :to_json, :content_type => :js)
  end

  get "/api/hits/email/:subject" do
    json(Hit.where(:email => params[:email]).first, :encoder => :to_json, :content_type => :js)
  end

  get "/api/person/:email" do
    json(Person.where(:email => params[:email]).first, :encoder => :to_json, :content_type => :js)
  end
end
end
