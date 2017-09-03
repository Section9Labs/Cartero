#encoding: utf-8
require 'active_model/serializers'

begin
  require 'mongoid'
rescue
  require 'active_support'
  require 'mongoid'
end

$LOAD_PATH.unshift(File.dirname(__FILE__) + '/models')

require 'person'
require 'hit'
require 'credential'
