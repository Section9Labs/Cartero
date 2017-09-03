#encoding: utf-8
# Documentation for Person
class Person
  include Mongoid::Document
  include Mongoid::Timestamps

  field :email, type: String
  field :campaigns, type: Array
  field :responded, type: Array
  field :credentials, type: Array

  validates_uniqueness_of :email
    
  has_many :hits
end
