#encoding: utf-8

module Cartero
module Commands
# Documentation for Cloner < ::Cartero::Command
class Cloner < ::Cartero::Command

  description(
    name: "Web Application Cloner",
    description: "This command allows a user to clone a site using Cartero's webservers." +
                 "Additionally, it will automatically edit forms, catch traffic, block bots, and redirect" +
                 "to the original site, among many other things.",
    author: ["Matias P. Brutti <matias [©] section9labs.com>"],
    type: "Infrastructure",
    license: "LGPL",
    references: [
      "https://section9labs.github.io/Cartero",
      "https://section9labs.github.io/Cartero"
      ]
  )

  def initialize
    super do |opts|
      opts.on("-U", "--url URL_PATH", String,
        "Full Path of site to clone") do |url|
        @options.url = url
      end

      opts.on("-W", "--webserver SERVER_NAME", String,
        "Sets WebServer name to use") do |ws|
        @options.webserver = ws
      end

      opts.on("-p", "--path PATH", String,
        "Sets path to save webserver") do |path|
        @options.path = path
      end

      opts.on("--useragent UA_STRING", String,
        "Sets user agent for cloning") do |payload|
        @options.useragent = payload
      end

      opts.on("--wget", "Use wget to clone url") do
        @options.wget = true
      end

      opts.on("--apache", "Generate Apache Proxy conf") do
        @options.apache = true
      end

      opts.on("--ssl-verify-none", "Forces Cloner not to verify certificates") do
        @options.ssl_verify_none = true
      end

      opts.on("--reverse-proxy", "Generates clone reverse proxing original links") do
        @options.reverse_proxy = true
      end

      opts.on("-P", "--payload PAYLOAD_PATH", String,
        "Sets payload path") do |payload|
        @options.payload = payload
      end
    end
  end

  attr_accessor :url
  attr_accessor :url_route
  attr_accessor :path
  attr_accessor :webserver
  attr_accessor :forms_routes
  attr_accessor :payload
  attr_accessor :wget
  attr_accessor :apache
  attr_accessor :useragent
  attr_accessor :domain_info
  attr_accessor :reverse_proxy

  def setup
    require 'erb'
    require 'uri'
    require 'fileutils'

    if @options.url.nil?
      raise StandardError, "A url [--url] must be provided"
    end

    if @options.webserver.nil?
      raise StandardError, "A WebServer name [--webserver] must be provided"
    end

    if @options.path.nil?
      puts "Saving WebServer to [ #{Cartero::TemplatesWebServerDir} ]"
      @options.path = ::Cartero::TemplatesWebServerDir
    end

    if ( @options.msfpayload.nil? && ( !@options.msfoptions.nil? || !@options.msfname.nil? || !@options.msfarch.nil? ))
      raise StandardError, "Option requires a --msfpayload"
    elsif ( !@options.msfpayload.nil? )
      @msfpayload = true
    end

    @msfvenom		= @options.msfvenom
    @url 				= @options.url
    @url_route 	= URI.parse(@options.url).path
    @path 			= File.expand_path @options.path
    @webserver 	= @options.webserver
    @payload 		= @options.payload
    @wget 			= @options.wget
    @apache 		= @options.apache
    @reverse_proxy = @options.reverse_proxy
    @useragent 	= @options.useragent || "Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/37.0.2049.0 Safari/537.36"
  end

  def run
    puts "Cloning URL #{@url}"
    create_structure
    clone

    create_apache_conf if apache
  end

  private

  def create_structure
    name = webserver.underscore
    Dir.mkdir path + "/"  + name unless File.directory?(path + "/"  + name)
    Dir.mkdir path + "/"  + name + "/static" unless File.directory? path + "/"  + name + "/static"
    Dir.mkdir path + "/"  + name + "/static/js" unless File.directory? path + "/"  + name + "/static/js"
    Dir.mkdir path + "/"  + name + "/views" unless File.directory? path + "/"  + name + "/views"
  end

  def delete_structure
    name = webserver.underscore
    Dir.rmdir path + "/"  + name + "/static" if File.directory? path + "/"  + name + "/static"
    Dir.rmdir path + "/"  + name + "/views" if File.directory? path + "/"  + name + "/views"
    Dir.rmdir path + "/"  + name if File.directory?(path + "/"  + name)

  end


  def clone
    require 'mechanize'
    require 'uri'
    mechanize = Mechanize.new
    mechanize.user_agent = useragent
    mechanize.agent.http.verify_mode = OpenSSL::SSL::VERIFY_NONE if @options.ssl_verify_none == true
    begin
      page = mechanize.get(url)
    rescue Net::HTTPForbidden
      $stdout.puts "Unauthorized Response - We'll still clone the site."
    rescue OpenSSL::SSL::SSLError
      $stdout.puts "Invalid Certificate - If you still want to clone use --ssl-verify-none flag."
      delete_structure
      exit(1)
    end


    forms_routes = page.forms.map {|x| [x.method.downcase , URI.parse(x.action).path ] }

    @domain_info = {
      :url 					=> url,
      :url_route    => url_route,
      :path 				=> path,
      :webserver 		=> webserver,
      :forms_routes => forms_routes,
      :payload 			=> payload
    }

    @domain_info[:reverse_proxy] = [] if reverse_proxy

    # Add our custom Javascript
    add_javascript_inject(page)
    # generate javascript file.
    FileUtils.cp(
        File.dirname(__FILE__) + "/../../../templates/webserver/loader_template.js",
        @options.path + '/' + webserver.underscore + '/static/js/loader.js'
      )

    # Create Index.erb
    if wget
      system("wget --no-check-certificate -O \"#{@options.path + '/' + webserver.underscore + '/views/index.erb'}\" -c -k  \"#{url}\"")
    else
      create_index(page)
    end

    # Create templated Sinatra WebServer
    ws = File.new(@options.path + '/' + webserver.underscore + '/' + webserver.underscore + '.rb', "w")
    ws << ERB.new(File.read(File.dirname(__FILE__) + "/../../../templates/webserver/template.rb.erb")).result(domain_info.get_binding)
    ws.close
  end

  def process_reverse_proxy_urls(page,zurl,regexp)
    page.search(regexp).each do |href|
      link = URI.parse(href.value)
      if link.scheme != 'mailto' && link.scheme != 'javascript' && link.path != '/' && link.path != ''
        if link.host.nil?
          link.host = zurl.host
          link.scheme = zurl.scheme
          href.value = link.path + "#{"?#{link.query}" if link.query != nil }"
        else
          href.value = link.path + "#{"?#{link.query}" if link.query != nil }"
        end
        path = [
          link.path.split('/')[0..1].join("\\/"),  # regular expression
          link.to_s.split('?')[0].split('/')[0..3].join('/')     # full path
        ]
        @domain_info[:reverse_proxy] << path unless @domain_info[:reverse_proxy].include? path
      end
    end
  end

  def proccess_urls(page,zurl,regexp)
    page.search(regexp).each do |href|
      next if href.to_s.index("#") == 0
      begin
        link = URI.parse(href.value)
        if link.host.nil? && link.scheme != 'mailto' && link.scheme != 'javascript'
          link.host = zurl.host
          link.scheme = zurl.scheme
          if link.path[-1] != "/"
            link.path = "/" + link.path
          end
          href.value = link.to_s
        end
      rescue URI::InvalidURIError
        $stdout.puts "Cloner was unable to handle URL (#{href.value}), leaving as it is"
      end
    end
  end

  def add_javascript_inject(page)
    page.search("//script")[0].add_next_sibling("<script src=\"/js/loader.js\"></script>")
  end

  def create_index(page)
    zurl = URI.parse(url)

    if reverse_proxy
      process_reverse_proxy_urls(page,zurl,"//*/@href")
      process_reverse_proxy_urls(page,zurl,"//*/@src")

    else # normal flow no Reverse Proxy
      proccess_urls(page,zurl,"//*/@href")
      proccess_urls(page,zurl,"//*/@src")
    end

    page.search("//form/@action").each do |form|
      # TODO: This could break if using reverse_proxy if one of the proxied
      # path is the same as the one in this relative path.Possible workaround
      # is requesting domain name where phishing sitewill be hosted.
      path = URI.parse(form.value).path
      if reverse_proxy
        conflict_rules = @domain_info[:reverse_proxy].select {|x| x[0] == path.split('/')[0..1].join("\\/")}
        unless conflict_rules.empty?
          $stdout.puts "ISSUE: It looks like the form #{path} is in conflict with one or more"
          $stdout.puts "       of the --reverse-proxy rules and it requires your attention:"
          conflict_rules.each do |rule|
            $stdout.puts  "\treverse_proxy /^#{rule[0]}\/?(.*)$/, '#{rule[1]}/$1'"
          end
        end
      end
      form.value = path
    end


    f = File.new(@options.path + '/' + webserver.underscore + '/views/index.erb', "w")
    if page.encoding != ""
      f << page.parser.to_s.force_encoding(page.encoding).encode('utf-8')
    else
      f << page.parser.to_s
    end
    f.close
  end

  def create_apache_conf
    puts "Generating Apache mod_proxy config file"
    ws = File.new(@options.path + '/' + webserver.underscore + '/' + webserver.underscore + '_apache_module_.conf', "w")
    ws << ERB.new(File.read(File.dirname(__FILE__) + "/../../../templates/apache/apache_proxy.conf.erb")).result(domain_info.get_binding)
    ws.close
  end
end
end
end
