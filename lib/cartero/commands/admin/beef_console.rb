#encoding: utf-8
require 'command_line_reporter'

module Cartero
module Commands
# Documentation for AdminConsole < ::Cartero::Command
class BeefConsole < ::Cartero::Command
  include CommandLineReporter

  description(
    name: "Beef Administration Console",
    description: "Cartero Console based Admnistration Interface for Beef API. It allows users to interact with the captured data (i.e. hooks, logs, modules, commands, dns, etc.)",
    author: ['Matias P. Brutti <matias [©] section9labs.com>'],
    type: "Admin",
    license: "LGPL",
    references: [
                'https://section9labs.github.io/Cartero',
                'http://beefproject.com'
                ]
  )
  def initialize
    super do |opts|
      opts.separator ""
      opts.separator "Beef Commands:"

      opts.on("-H", "--hooks",
        "Display the list of Beef Hooked Browsers") do
        @options.hooks = true
      end

      opts.on("-L","--logs", String,
        "Display Beef logs for all or a specific hooked Session") do
        @options.logs = true
      end

      opts.on("-M","--modules",
        "Display Beef logs for all or a specific hooked Session") do
        @options.modules = true
      end

      opts.on("-C","--command",
        "Run a Command id agasint a hooked Browser") do
        @options.command = true
      end

      opts.on("-C","--command-results CID", String,
        "Run a Command id agasint a hooked Browser") do |c|
        @options.command_result = true
        @options.cmd_id = c
      end

      opts.on("-D","--dns-rules",
        "Display Beef logs for all or a specific hooked Session") do
        @options.dns_rules = true
      end

      opts.on("--dns-rule-remove",
        "Display Beef logs for all or a specific hooked Session") do
        @options.dns_rule_remove = true
      end

      opts.separator ""
      opts.separator "Beef parameter options:"

      opts.on("-i","--id ID", String,
        "Set session for module or dns") do |id|
        @options.id = id
      end

      opts.on("--mid ID1,ID2,ID3", Array,
        "Set multi-command hooked IDs") do |id|
        @options.mid = id
      end

      opts.on("-s","--session SESSION", String,
        "Set session for hooked browser") do |s|
        @options.session = s
      end

      opts.on("-p","--parameters JSONFILE", String,
        "Set parameters for hooked browser command") do |p|
        @options.parameters = p
      end
    end
  end

  def setup
    raise StandardError, "Missing {\"beef\" : {\"hook\" : \"VALUE\"} } in ~/.cartero/config"  if ::Cartero::GlobalConfig['beef']['hook'].nil? || ::Cartero::GlobalConfig['beef']['hook'] == ""
    setup_beef_rest_client

    # Work the Paramters into Hash object for RestClient
    # TODO: Implement msfvenon style options parser.
    #       I need to evaluate this better, it is not as simple as it sounds.
    return if @options.parameters.nil?
    if File.exist?(File.expand_path(@options.parameters))
      @options.parameters = JSON.parse(File.read(File.expand_path(@options.parameters)))
    else
      raise StandardError, "File does not exists"
    end
  end

  def run
    # Hooks
    run_hooks if @options.hooks
    # Logs
    run_logs if @options.logs
    # Modules
    run_modules if @options.modules
    # Commands
    run_commands if @options.command
    # Commands
    run_command_results if @options.command_result
    # DNS
    run_dns if @options.dns_rules
  end

  private

  # Run methods
  def run_hooks
    if @options.session
      info = @rest_client.hook(@options.session)
      display_hook(info)
    else
      hooked = @rest_client.hooks
      display_hooks(hooked['hooked-browsers']['online'], 'ONLINE')
      display_hooks(hooked['hooked-browsers']['offline'], 'OFFLINE')
    end
  end

  def run_logs
    if @options.session.nil?
      logs = @rest_client.logs
    else
      logs = @rest_client.log(@options.session)
    end
    display_logs(logs)
  end

  def run_modules
    if @options.id.nil?
      modules = @rest_client.modules
      display_modules(modules)
    else
      mod = @rest_client.module(@options.id)
      display_module(mod)
    end
  end

  def run_commands
    raise StandardError, "Missing hooked browser(s) session." if @options.session.nil? && @options.mid.nil?
    raise StandardError, "Missing module ID" if @options.id.nil?

    parameters = @rest_client.module(@options.id)['options']
    if !parameters.empty? && @options.parameters.nil?
      display_module_options(parameters)
      raise StandardError, "This module requires paramaters."
    end
    if @options.session && @options.id
      data = @rest_client.command(@options.session, @options.id, @options.parameters || {})
      if data['success'] == 'true'
        puts "Command ID: #{data['command_id']}"
      else
        puts "Command unsuccessful. Something went wrong."
      end
    elsif @options.mid && @options.id
      cmd_ids = @rest_client.multi_command(@options.mid, @options.id, @options.parameters || {})
      display_command_ids(cmd_ids)
    else
      $stderr.puts "Error: I should not be here"
    end
  end

  def run_command_results
    raise StandardError, "Missing Command ID" if @options.cmd_id.nil?
    raise StandardError, "Missing hooked browser(s) session." if @options.session.nil?
    raise StandardError, "Missing module ID" if @options.id.nil?

    results = @rest_client.command_result(@options.session, @options.id, @options.cmd_id)

    display_command_results(results)
  end

  def run_dns
    if @options.id.nil?
      dns_ruleset = @rest_client.dns_ruleset
      display_dns_ruleset(dns_ruleset)
    else
      rule = @rest_client.dns_rule(@options.id)
      display_dns_rule(rule)
    end
  end

  def run_dns_remove
    raise StandardError, "missing ID" if id.nil?
    dns = @rest_client.dns_rule_remove(id)
    display_dns_remove(dns)
  end

  # Display Methods
  def display_command_ids(results)
    return if results.empty?
    table() do
      row(:color => 'red', :header => true, :bold => true) do
        column('HOOK', :width => 4)
        column('CMD', :width => 3)
      end
      results.each do |k, v|
        row() do
          column(k, :color => 'blue', :width => 7)
          column(v, :color => 'blue', :width => 7)
        end
      end
    end
  end

  def display_command_results(results)
    return if results.empty?
    cols = Integer(`tput co`) / 2
    table() do
      row(:color => 'red', :header => true, :bold => true) do
        column('DATE', :width => 26)
        column('DATA', :width => cols)
      end
      results.each do |_k, r|
        row() do
          column(Time.at(r['date'].to_i), :color => 'blue', :width => 26)
          column(JSON.parse(r['data'])['data'], :color => 'blue', :width => cols)
        end
      end
    end
  end

  def display_dns_ruleset(ruleset)
    pp ruleset
  end

  def display_dns_rule(rule)
    pp rule
  end

  def display_dns_remove(rule)
    pp rule
  end

  def display_module(mod)
    return if mod.empty?
    val = Integer(`tput co`)
    val > 166 ? cols =  - 166 : cols = 100
    table() do
      row(:color => 'red', :header => true, :bold => true) do
        column('NAME', :width => 30)
        column('DESCRIPTION', :width => cols)
        column('CLASS', :width => 40)
        column('CATEGORY', :width => 40)
      end
      row(:color => 'blue') do
        column(mod['name'])
        column(mod['description'])
        column(mod['class'])
        column(mod['category'].is_a?(Array) ? mod['category'].join(', ') : mod['category'])
      end
    end
    display_module_options(mod['options'])
  end

  def display_module_options(options)
    if options.empty?
      puts 'No Options Available'
    else
      puts ""
      puts "OPTIONS"
      cols = (Integer(`tput co`) - 105) / 2
      table() do
        row(:color => 'red', :header => true, :bold => true) do
          column('LABEL', :width => 30)
          column('NAME', :width => 30)
          column('DESCRIPTION', :width => cols + 20)
          column('VALUE', :width => cols)
        end
        options.each do |x|
          row(:color => 'blue') do
            column(x['ui_label'])
            column(x['name'])
            column(x['description'])
            column(x['value'])
          end
        end
      end
    end
  end

  def display_modules(modules)
    return if modules.empty?
    table() do
      row(:color => 'red', :header => true, :bold => true) do
        column('ID', :width => 3)
        column('NAME', :width => 40)
        column('CLASS', :width => 40)
        column('CATEGORY', :width => 40)
      end
      modules.each do |_k,value|
        row(:color => 'blue') do
          column(value['id'].to_s)
          column(value['name'])
          column(value['class'])
          column(value['category'].is_a?(Array) ? value['category'].join(', ') : value['category'])
        end
      end
    end
  end

  def display_logs(logs)
    return if logs.empty?
    cols = Integer(`tput co`) - 48
    table() do
      row(:color => 'red', :header => true, :bold => true) do
        column('ID', :width => 3)
        column('DATE', :width => 25)
        column('TYPE', :width => 15)
        column('EVENT', :width => cols)
      end
      logs['logs'].each do |value|
        row(:color => 'blue') do
          column(value['id'].to_s)
          column(value['date'])
          column(value['type'])
          column(value['event'])
        end
      end
    end
  end

  def display_hooks(p,t)
    return if p.empty?
    puts t
    val = Integer(`tput co`)
    val > 182 ? cols =  val - 182  : cols =  100
    table() do
      row(:color => 'red', :header => true, :bold => true) do
        column('ID', :width => 3)
        column('IP', :width => 16)
        column('SESSION', :width => 80)
        column('NAME', :width => 15)
        column('VERSION', :width => 10)
        column('OS', :width => 15)
        column('PLATFORM', :width => 15)
        column('DOMAIN', :width => 10)
        column('PORT', :width => 6)
        column('PAGE_URI', :width => cols)
      end
      p.each do |_key,value|
        row(:color => 'blue') do
          column(value['id'])
          column(value['ip'])
          column(value['session'])
          column(value['name'])
          column(value['version'])
          column(value['os'])
          column(value['platform'])
          column(value['domain'])
          column(value['port'])
          column(value['page_uri'])
        end
      end
    end
  end

  def display_hook(data)
    return if data.empty?
    cols = Integer(`tput co`) / 2
    table() do
      data.each do |key,value|
        row() do
          column(key, :color => 'red', :bold => true, :width => 20)
          column(value, :color => 'blue', :width => cols)
        end
      end
    end
  end

  def setup_beef_rest_client
    require 'cartero/beef_api'
    @rest_client = ::Cartero::BeefApi.new(
      :server => ::Cartero::GlobalConfig['beef']['hook'],
      :username => @options.username || ::Cartero::GlobalConfig['beef']['username']  || "beef",
      :password => @options.password || ::Cartero::GlobalConfig['beef']['password']  || "beef"
    )
    @rest_client.login
    # Time to see if we were able to login.
    raise StandardError, "Something went wrong while connecting to Beef RESTful API" if @rest_client.token.nil?
  end
end
end
end
