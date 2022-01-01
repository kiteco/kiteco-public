FROM nginx:1.19.5

RUN apt-get update && apt-get install -y \
  wget \
  gcc \
  libpcre3-dev \
  zlib1g-dev \
  build-essential \
  libssl-dev \
  libxslt-dev \
  libgd-dev \
  libgeoip-dev \
  # nginx=1.14.0-0ubuntu1.7
  && rm -rf /var/lib/apt/lists/*

RUN wget http://nginx.org/download/nginx-1.19.5.tar.gz
RUN tar -zxvf nginx-1.19.5.tar.gz

RUN wget https://github.com/slact/nchan/archive/v1.2.7.tar.gz
RUN tar -zxvf v1.2.7.tar.gz

# Run configure with same arguments as pre-built nginx. See `nginx -V`.
RUN cd nginx-1.19.5 && ./configure \
 --add-dynamic-module=../nchan-1.2.7 \
 --prefix=/etc/nginx \
 --sbin-path=/usr/sbin/nginx \
 --modules-path=/usr/lib/nginx/modules \
 --conf-path=/etc/nginx/nginx.conf \
 --error-log-path=/var/log/nginx/error.log \
 --http-log-path=/var/log/nginx/access.log \
 --pid-path=/var/run/nginx.pid \
 --lock-path=/var/run/nginx.lock \
 --http-client-body-temp-path=/var/cache/nginx/client_temp \
 --http-proxy-temp-path=/var/cache/nginx/proxy_temp \
 --http-fastcgi-temp-path=/var/cache/nginx/fastcgi_temp \
 --http-uwsgi-temp-path=/var/cache/nginx/uwsgi_temp \
 --http-scgi-temp-path=/var/cache/nginx/scgi_temp \
 --user=nginx \
 --group=nginx \
 --with-compat \
 --with-file-aio \
 --with-threads \
 --with-http_addition_module \
 --with-http_auth_request_module \
 --with-http_dav_module \
 --with-http_flv_module \
 --with-http_gunzip_module \
 --with-http_gzip_static_module \
 --with-http_mp4_module \
 --with-http_random_index_module \
 --with-http_realip_module \
 --with-http_secure_link_module \
 --with-http_slice_module \
 --with-http_ssl_module \
 --with-http_stub_status_module \
 --with-http_sub_module \
 --with-http_v2_module \
 --with-mail \
 --with-mail_ssl_module \
 --with-stream \
 --with-stream_realip_module \
 --with-stream_ssl_module \
 --with-stream_ssl_preread_module \
 --with-cc-opt='-g -O2 -fdebug-prefix-map=/data/builder/debuild/nginx-1.19.5/debian/debuild-base/nginx-1.19.5=. -fstack-protector-strong -Wformat -Werror=format-security -Wp,-D_FORTIFY_SOURCE=2 -fPIC' \
 --with-ld-opt='-Wl,-z,relro -Wl,-z,now -Wl,--as-needed -pie'

RUN cd nginx-1.19.5 && make modules
RUN cp nginx-1.19.5/objs/ngx_nchan_module.so /usr/lib/nginx/modules/

# RUN apt-get update && apt-get install -y \
#   libnginx-mod-nchan=1.14.2-2+deb10u3 \
#   && rm -rf /var/lib/apt/lists/*

COPY conf/nginx.conf /etc/nginx/nginx.conf
COPY conf/sites-enabled /etc/nginx/sites-enabled
