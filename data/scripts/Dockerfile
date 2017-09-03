FROM alpine:latest


# Dependencies for getting/building
RUN apk update && \
    apk upgrade && \
    apk add --no-cache gcc gnupg curl wget ruby ruby-dev bash procps musl-dev make \
        zlib zlib-dev openssl openssl-dev libssl1.0 \
        mongodb postfix git openssh build-base libffi-dev \
        libxslt-dev libxml2-dev libxml2 libxml2-dev libxslt-dev && \
    gem install rake bundler -N


# Installing Cartero
RUN mkdir -p /usr/local/share/Cartero && \
    git clone https://github.com/section9labs/Cartero /usr/local/share/Cartero && \
    echo "export EDITOR=vim" >> ~/.bash_profile && \
    cd /usr/local/share/Cartero && \
    bundle install && \
    echo -e "#\0041/bin/bash\n/usr/local/share/Cartero/bin/cartero \"\$@\"" > /usr/local/bin/cartero && \
    chmod +x /usr/local/bin/cartero && \
    echo "[[ -s /usr/local/share/Cartero/data/scripts/CarteroComplete.sh ]] && source /usr/local/share/Cartero/data/scripts/CarteroComplete.sh" >> ~/.bash_profile

