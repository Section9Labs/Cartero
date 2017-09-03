#encoding: utf-8
# Documentation for Hit
class Hit
  include Mongoid::Document
  include Mongoid::Timestamps

  field :ip, 					type: String
  field :location,    type: Hash
  field :port, 				type: String
  field :domain,			type: String
  field :path,				type: String
  field :user_agent, 	type: String
  field :ua_comp,			type: String
  field :ua_os, 			type: String
  field :ua_browser, 	type: String
  field :ua_engine,		type: String
  field :ua_platform,	type: String
  field :ua_lang,			type: String
  field :forwarded, 	type: Boolean
  field :data, 				type: Hash

  belongs_to :user
end
