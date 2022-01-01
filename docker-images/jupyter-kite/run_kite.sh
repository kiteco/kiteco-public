#!/bin/bash
(
cd ~

test -f /usr/share/kite/kite-credentials && source /usr/share/kite/kite-credentials

/usr/share/kite/current/kited&

if [[ -z "$KITE_USER" ]] || [[ -z "$KITE_PASSWORD" ]]
then
      echo "KITE_USER and KITE_PASSWORD env variable are not defined, staying as anonymous user (Kite Free)"
else
      echo "Login to Kite with the user $KITE_USER"
      sleep 5
      status_code=$( curl --write-out %{http_code} --output /usr/share/kite/login_logs.txt -X POST -F "email=$KITE_USER" -F "password=$KITE_PASSWORD" http://localhost:46624/clientapi/login)
      if [[ "$status_code" -ne 200 ]] ; then
        echo "Error while logging in, please check your credentials"
        cat /usr/share/kite/login_logs.txt
        echo ""
        exit 1
      else
        echo "Login to Kite successful with user $KITE_USER"
      fi
fi

mkdir -p /home/jovyan/.local/share/kite/current/
cp /usr/share/kite/current/kite-lsp /home/jovyan/.local/share/kite/current/kite-lsp
cp /usr/share/kite/current/kited /home/jovyan/.local/share/kite/kited
) </dev/null > /usr/share/kite/run_logs.txt 2>&1 &

if [[ -z "$JUPYTERHUB_USER" ]]
then
  cd ~
  mkdir -p notebooks
  export NOTEBOOK_DIR_ARG="--notebook-dir=/home/$NB_USER/notebooks"
  echo "Setting notebook dir arg : $NOTEBOOK_DIR_ARG"
else
  cd
fi

echo "Command executed : start-notebook.sh $NOTEBOOK_DIR_ARG $@"
start-notebook.sh $NOTEBOOK_DIR_ARG "$@"
