#encoding: utf-8
module Cartero
# Documentation for CryptoBox - AES Implementation
module CryptoBox
  ALGORITHM = "aes-256-cbc"
  @@key = nil
  @@iv = nil
  @@secret_box = nil
  @@nonce = nil

  def self.init
    if !key.nil? && !iv.nil? && !secret_box.nil?
      return true
    elsif File.exist?(Cartero::SecretMaterial)
      # Read data from JSON File
      m = JSON.parse(File.read(::Cartero::SecretMaterial),{:symbolize_names => true})
      self.secret_box = OpenSSL::Cipher.new(ALGORITHM)
      self.key        = Base64.decode64(m[:key])
      self.nonce 			= Base64.decode64(m[:nonce]).strip
    else
      @@secret_box = OpenSSL::Cipher.new(ALGORITHM)
      @@key = Digest::SHA256.hexdigest(@@secret_box.random_key)
      @@iv=  @@secret_box.random_iv
      # Storing keys on file.
      f = File.new(::Cartero::SecretMaterial, "w+")
      f.puts({
        :key => Base64.strict_encode64(@@key).strip,
        :nonce => Base64.strict_encode64(@@iv).strip
        }.to_json
      )
      f.close
    end
    true
  end

  def self.reinit
    File.delete(::Cartero::SecretMaterial)
    init
  end

  def self.encrypt(m)
    secret_box.encrypt
    secret_box.key = key
    secret_box.iv = iv
    e = secret_box.update(m)
    e << secret_box.final
    x = Base64.urlsafe_encode64(Base64.strict_encode64(e).strip).strip#.gsub("+","-").gsub("/","_")
    x
  end

  def self.decrypt(e)
    secret_box.decrypt
    secret_box.key = key
    secret_box.iv = iv
    d = secret_box.update(
      Base64.strict_decode64(
        Base64.urlsafe_decode64(e)#.gsub("-","+").gsub("_","/")
      ).strip
    )
    d << secret_box.final
    d
  end

  def self.close
    self.key = nil
    self.nonce = nil
    self.secret_box = nil
  end

  class << self
    private

    def key
      @@key
    end

    def iv
      @@iv
    end

    def nonce
      iv
    end

    def secret_box
      @@secret_box
    end

    def key=(k)
      @@key = k
    end

    def iv=(n)
      @@iv = n
    end

    def nonce=(n)
      @@iv = n
			@@nonce = n
    end

    def secret_box=(s)
      @@secret_box = s
    end
  end
end
end
