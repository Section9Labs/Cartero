#encoding: utf-8
require 'command_line_reporter'

module Cartero
module Commands
# Documentation for AdminConsole < ::Cartero::Command
class AdminConsole < ::Cartero::Command
  include CommandLineReporter

  description(
    name: "Administration Console",
    description: "Cartero Console based Admnistration Interface. It allows users to interact with the captured data (i.e. hits, persons, credentials.)",
    author: ["Matias P. Brutti <matias [©] section9labs.com>"],
    type: "Admin",
    license: "LGPL",
    references: ["https://section9labs.github.io/Cartero"]
  )
  
  def initialize
    super do |opts|
      opts.on("-p", "--persons [LATEST_N]", Integer,
        "Display the list of persons that responded") do |n|
        @options.persons = n || 50
      end

      opts.on("-i", "--hits [LATEST_N]", Integer,
        "Display the list of hits") do |n|
        @options.hits = n || 100
      end

      opts.on("-c", "--creds [LATEST_N]", Integer,
        "Display the list of Credentials") do |n|
        @options.credentials = n || 50
      end

      opts.on("-a", "--all",
        "Sets Email Payload Ports to scan") do
        @options.all = true
      end

      opts.on("-f", "--filter",
        "flag to search by parameters") do
        @options.filter = true
      end

      opts.on("--email [EMAIL]", String,
        "Display the list of hits") do |e|
        @options.email = e
      end

      opts.on("--campaign [CAMPAIGN]", String,
        "Display the list of hits") do |c|
        @options.campaign = c
      end

      opts.on("--ip [IP_ADDRESS]", String,
        "Display the list of hits") do |ip|
        @options.ip = ip
      end
    end
  end

  attr_accessor :persons
  attr_accessor :hits
  attr_accessor :credentials
  attr_accessor :hooks
  attr_accessor :all
  attr_accessor :email
  attr_accessor :campaign
  attr_accessor :ip
  attr_accessor :filter

  def setup
    require 'cartero/models'

    @persons 			= @options.persons
    @hits 				= @options.hits
    @credentials 	= @options.credentials
    @hooks        = @options.hooks
    @email        = @options.email
    @campaign     = @options.campaign
    @ip           = @options.ip
    @filter       = @options.filter

    ::Cartero::DB.start

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



  def run
    run_persons if persons || all
    run_hits if hits || all
    run_credentials if credentials || all
  end

  private

  def run_persons
    return display_persons(Person.sort(:updated_at.desc).limit(persons || 50).all.reverse) if !filter
    p_email    = Person.where(:email => /#{email}/).all if !email.nil?
    p_ip       = Person.all.select { |x| x.responded.to_s =~ /#{ip}/ } if !ip.nil?
    p_campaign = Person.all.select { |x| x.campaigns.to_s =~ /#{campaign}/i } if !campaign.nil?
    pe = []
    pe.concat(p_campaign) if !p_campaign.nil?
    pe.concat(p_email) if !p_email.nil?
    pe.concat(p_ip) if !p_ip.nil?
    pe.uniq!
    display_persons(pe)
  end

  def run_hits
    return display_hits(Hit.sort(:created_at.desc).limit(hits || 100).all.reverse) if !filter
    h_email    = Hit.where.all.select {|x| x.data['email'] =~ /#{email}/i }if !email.nil?
    h_campaign = Hit.where.all.select {|x| x.data['subject'] =~ /#{campaign}/i } if !campaign.nil?
    h_ip       = Hit.where(:ip => /#{ip}/i).all if !ip.nil?
    h = []
    h.concat(h_campaign) if !h_campaign.nil?
    h.concat(h_email) if !h_email.nil?
    h.concat(h_ip) if !h_ip.nil?
    h.uniq!
    display_hits(h)
  end

  def run_credentials
    return display_credentials(Credential.sort(:created_at.desc).limit(credentials || 50).all.reverse) if !filter
    c_email    = Credential.where(:username => /#{email}/i).all if !email.nil?
    c_campaign = Credential.where(:domain =~ /#{campaign}/i).all if !campaign.nil?
    c_ip       = Credential.where(:ip => /#{ip}/i).all if !ip.nil?
    c = []
    c.concat(c_campaign) if !c_campaign.nil?
    c.concat(c_email) if !c_email.nil?
    c.concat(c_ip) if !c_ip.nil?
    c.uniq!
    display_credentials(c)
  end

  def display_persons(p)
    return if p.empty?

    table() do
      row(:color => 'red', :header => true, :bold => true) do
        column('ID', :width => 3)
        column('EMAIL', :width => 27)
        column('LAST SEEN', :width => 20)
        column('LAST OS', :width => 20)
        column('HITS', :width => 4)
        column('CREATED', :width => 20)
        column('UPDATED', :width => 20)
      end
      p.each_with_index do |person,idx|
        row(:color => 'blue') do
          column(idx + 1)
          column(person.email)
          column(person.responded.last)
          column(person.hits.last.ua_os)
          column(person.responded.size)
          column(person.created_at.strftime("%m/%d/%Y-%T"))
          column(person.updated_at.strftime("%m/%d/%Y-%T"))
        end
      end
    end
  end

  def display_hits(h)
    return if h.empty?

    table() do
      row(:color => 'red', :header => true, :bold => true) do
        column('ID', 			:width => 3)
        column('EMAIL',   :width => 20)
        column('CAMPAIGN',:width => 40)
        column('IP', 			:width => 16)
        column('PORT', 		:width => 6)
        column('DOMAIN', 	:width => 30)
        column('GEOLOCATION', 	:width => 30)
        column('PATH', 		:width => 20)
        column('OS', 			:width => 15)
        column('BROWSER', :width => 20)
        column('ENGINE', 	:width => 20)
        column('PLATFORM',:width => 22)
        column('CREATED', :width => 20)
      end
      h.each_with_index do |hit,idx|
        row(:color => 'blue') do
          column(idx + 1)
          if !hit.data.nil?
            column(hit.data['email'])
            column(hit.data['subject'])
          else
            column("")
            column("")
          end
          column(hit.ip)
          column(hit.port)
          column(hit.domain)
          column(hit.location['city'] + ' - ' + hit.location['country_name'])
          column(hit.path)
          column(hit.ua_os)
          column(hit.ua_browser)
          column(hit.ua_engine)
          column(hit.ua_platform)
          column(hit.created_at.strftime("%m/%d/%Y-%T"))
        end
      end
    end
  end

  def display_credentials(c)
    return if c.empty?

    table() do
      row(:color => 'red', :header => true, :bold => true) do
        column('ID', 			:width => 3)
        column('USERNAME',:width => 20)
        column('PASSWORD',:width => 20)
        column('IP',			:width => 18)
        column('PORT', 		:width => 6)
        column('DOMAIN', 	:width => 20)
        column('GEOLOCATION', 	:width => 30)
        column('PATH', 		:width => 20)
        column('OS', 			:width => 15)
        column('BROWSER', :width => 20)
        column('ENGINE', 	:width => 20)
        column('PLATFORM',:width => 22)
        column('CREATED', :width => 20)
      end
      c.each_with_index do |cred,idx|
        row(:color => 'blue') do
          column(idx + 1)
          column(cred.username)
          column(cred.password)
          column(cred.ip)
          column(cred.port)
          column(cred.domain)
          column(cred.location['city'] + ' - ' + cred.location['country_name'])
          column(cred.path)
          column(cred.ua_os)
          column(cred.ua_browser)
          column(cred.ua_engine)
          column(cred.ua_platform)
          column(cred.created_at.strftime("%m/%d/%Y-%T"))
        end
      end
    end
  end
end
end
end
