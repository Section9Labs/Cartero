#encoding: utf-8
module Cartero
  # Documentation for module DB
  # Responsible for launching and stopping mongodb on Linux / MacOSX.
  module DB
    def self.start(ip=nil, port=nil)
      case RUBY_PLATFORM
      when /linux|darwin/
        mongod = "mongod --fork"
        #$stdout.puts "Launching mongodb"
        system( mongod + " --dbpath=" + ::Cartero::MongoDBDir + " --logpath=" + ::Cartero::LogsDir + '/mongodb.log' + " --bind_ip #{ip || 'localhost'} --port #{port || '27017'}" + " --logappend" + "> /dev/null")
        sleep(1)
      when /mingw|mswin/
        # TODO: Windows implementation.
        $stdout.puts "TODO: Manual Launch. !!! Make sure mongodb is running. !!!"
        sleep(1)
      end
    rescue => e
      $stderr.puts "Something went wrong starting db - #{e}"
      exit(0)
    end

    def self.stop
      #$stdout.puts "Stopping Mongodb"
      Mongoid.connection['admin'].command(:shutdown => 1)
      $stdout.puts "" if RUBY_PLATFORM =~ /mingw|mswin/
    rescue StandardError
      $stderr.puts "Looks like there is not an establish connection to shutdown."
      pid = File.read(::Cartero::MongoDBDir + '/mongod.lock').strip.to_i
      $stderr.puts "Killing Process #{pid}"
      Process.kill(2,pid)
    end
  end
end
