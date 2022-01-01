upstream nchan_redis {
  nchan_redis_server XXXXXXX;
}

upstream alpha.kite.com {
  server alpha.kite.com:443 max_fails=0;
  keepalive 256;
}

nchan_max_reserved_memory 1024M;

server {
  listen 9094 default_server;
  listen [::]:9094 default_server;

  server_name rc.kite.com;

  proxy_http_version 1.1;
  proxy_set_header Connection "";

  nchan_subscriber_timeout 3600;

  location = /internal/auth {
    resolver 8.8.8.8;
    proxy_pass https://alpha.kite.com/require-auth?ID=$nchan_channel_id;
    proxy_pass_request_body off;
  }

  location ~ ^/receive/([0-9]+)$ {
    nchan_subscriber websocket;

    nchan_channel_group "user";
    nchan_authorize_request /internal/auth;
    nchan_channel_id $1;

    nchan_redis_pass nchan_redis;
  }

  location ~ ^/send/([0-9]+)$ {
    auth_basic "send message";
    auth_basic_user_file /etc/nginx/htpasswd;

    nchan_publisher http;

    nchan_channel_group "user";
    nchan_channel_id $1;

    nchan_redis_pass nchan_redis;
  }

  location ~ ^/receive/([\w-]+)$ {
    nchan_subscriber websocket;

    nchan_channel_group "install";
    nchan_channel_id $1;

    nchan_redis_pass nchan_redis;
  }

  location ~ ^/send/([\w-]+)$ {
    auth_basic "send message";
    auth_basic_user_file /etc/nginx/htpasswd;

    nchan_publisher http;

    nchan_channel_group "install";
    nchan_channel_id $1;

    nchan_redis_pass nchan_redis;
  }

  location ~ ^/cohort/([\w-]+)$ {
    proxy_pass http://127.0.0.1:9000;
  }
  location ~ ^/convcohort {
    proxy_pass http://127.0.0.1:9000;
  }

  location = /nginx_status {
    # Enable Nginx stats
    stub_status on;
    # Only allow access from your IP e.g 1.1.1.1 or localhost #
    allow 127.0.0.1;
    # Other request should be denied
    deny all;
  }

  location /.ping {
    return 200 "pong";
  }
}

server {
  listen 80 default_server;
  listen [::]:80 default_server;

  server_name rc.kite.com;

  location /.ping {
    return 200 "pong";
  }
}
