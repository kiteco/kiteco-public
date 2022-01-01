#!/usr/bin/env bash
set -e

URL="https://linux.kite.com/linux/current/kite-installer"

# Exit codes:
#  1 - unknown/generic error
# 10 - OS unsupported
# 12 - no AVX support
# 15 - missing dependencies
# 20 - root user unsupported
# 30 - systemctl not found
# 40 - MS WSL unsupported
# 50 - wget and curl unavailable

function checkPrerequisites(){
    if ! uname -a | grep -i "x86_64" | grep -qi "Linux"; then
        echo >&2 "Sorry! This installer is only compatible with Linux on x86_64. Exiting now."
        exit 10
    fi

    if ! grep -q '\<avx[^ ]*\>' /proc/cpuinfo; then
        echo >&2 "Sorry! Kite only runs on processor architectures with AVX support. Exiting now."
        exit 12
    fi

    if [ -f /etc/centos-release ] && [ "$(cat /etc/centos-release | tr -dc '0-9.' | cut -d \. -f1)" -lt 8 ]; then
        echo >&2 "Sorry! This installer is not compatible with CentOS 7 and earlier due to incomplete systemd support."
        echo >&2 "See https://bugzilla.redhat.com/show_bug.cgi?id=1173278 for details. Exiting now."
        exit 10
    fi

    if [ -f /etc/redhat-release ] && [ "$(cat /etc/redhat-release | tr -dc '0-9.' | cut -d \. -f1)" -lt 8 ]; then
        echo >&2 "Sorry! This installer is not compatible with RHEL 7 and earlier due to incomplete systemd support."
        echo >&2 "See https://bugzilla.redhat.com/show_bug.cgi?id=1173278 for details. Exiting now."
        exit 10
    fi
}

function promptX11Dependencies() {
    echo "Checking to see if all dependencies are installed...."
    echo

    if command -v yum >/dev/null 2>&1; then
        if ! yum list installed libXScrnSaver &> /dev/null; then
            echo "Did not find libXScrnSaver on your system. We can install it now or you can install and re-run this script"
            read -r -e -p "Install it now? (you might be asked for your sudo password) [Y/n] " INSTALL
            INSTALL=${INSTALL:-Y}
            if [[ $INSTALL == "Y" || $INSTALL == "y" ]]; then
                sudo yum install -y -q libXScrnSaver
            else
                echo "Please run 'sudo yum install libXScrnSaver' and rerun this script! Exiting now."
                exit 15
            fi
        fi
    elif command -v zypper >/dev/null 2>&1; then
        if ! zypper se -i -x libXss1 >/dev/null 2>&1; then
            echo "Did not find libXss1 on your system. We can install it now or you can install and re-run this script"
            read -r -e -p "Install it now? (you might be asked for your sudo password) [Y/n] " INSTALL
            INSTALL=${INSTALL:-Y}
            if [[ $INSTALL == "Y" || $INSTALL == "y" ]]; then
                sudo zypper -n -q install libXss1
            else
                echo "Please run 'sudo zypper install libXss1' and rerun this script! Exiting now."
                exit 15
            fi
        fi
    elif command -v pacman >/dev/null 2>&1; then
        if ! pacman -Qs 'libxss' >/dev/null 2>&1; then
            echo "Did not find libxss on your system. we can install it now or you can install and re-run this script"
            read -r -e -p "Install it now? (you might be asked for your sudo password) [Y/n] " INSTALL
            INSTALL=${INSTALL:-Y}
            if [[ $INSTALL == "Y" || $INSTALL == "y" ]]; then
                sudo pacman -q --noconfirm -S libxss
            else
                echo "Please run 'sudo pacman -S libxss' and rerun this script! Exiting now."
                exit 15
            fi
        fi
    elif command -v dpkg >/dev/null 2>&1 && command -v apt-get >/dev/null 2>&1; then
        if ! dpkg -S libxss1 &> /dev/null; then
            echo "Did not find libxss1 on your system. We can install it now or you can install and re-run this script"
            read -r -e -p "Install it now? (you might be asked for your sudo password) [Y/n] " INSTALL
            INSTALL=${INSTALL:-Y}
            if [[ $INSTALL == "Y" || $INSTALL == "y" ]]; then
                sudo apt-get install -y -qq libxss1
            else
                echo "Please run 'sudo apt-get install libxss1' and rerun this script! Exiting now."
                exit 15
            fi
        fi
    else
        echo
        echo "Unable to determine if libxss1/libXScrnSaver is installed on your system. Please use your "
        echo "system's package manager to verify this package is installed and manually run:"
        echo
        echo "    ./kite-installer install"
        echo
        echo "Exiting now."
        exit 1
    fi
}

# Download the binary kite-installer and store it at the location defined by the first parameter
# It sets the executable flag after a successful download
function downloadKiteInstaller() {
    local target="$1"

    if command -v wget >/dev/null 2>&1; then
        echo "Downloading $target binary using wget..."
        wget -q -O "$target" "$URL" || { echo >&2 "Failed to download $target. Run 'wget -O \"$target\" \"$URL\"' for more information. Exiting now."; exit 1; }
    elif command -v curl >/dev/null 2>&1; then
        echo "Downloading $target binary using curl..."
        curl -L --output "$target" "$URL" || { echo >&2 "Failed to download $target. Run 'curl -L --output \"$target\" \"$URL\" for more information. Exiting now."; exit 1; }
    else
        echo >&2 "Sorry! either wget or curl have to be available to download the installer. terminating."
        exit 50
    fi

    [ -f "$target" ] || { echo >&2 "Unable to locate downloaded file $target. terminating."; exit 1; }
    chmod u+x "$downloadFile" || { echo >&2 "Failed to make $downloadFile executable. Run 'chmod u+x $downloadFile' for more information. Exiting now."; exit 1; }
}

# Uses the kite-installer passed as first argument to download the Kite installation package
function downloadKitePackage() {
    local target="$1"
    case "$target" in
      /*) ;; # absolute path
       *) target="./$target" ;; # relative path
    esac
    [ -x "$target" ] || { echo >&2 "Unable to locate executable file $target. terminating."; exit 1; }

    "$target" install --download || { echo >&2 "Unable to download Kite executable package. terminating."; exit 1; }
}

function installKite() {
    local downloadFile="$1"
    shift # we're using the remaining args later on
    case "$downloadFile" in
      /*) ;; # absolute path
       *) downloadFile="./$downloadFile" ;; # relative path
    esac

    [ -f "$downloadFile" ] || { echo "Unable to locate kite-installer at $downloadFile. Exiting now."; exit 1; }

    echo "Running $downloadFile install $*"
    "$downloadFile" install "$@"
    status="$?"
    if [ "$status" != "0" ]; then
        echo
        echo "There was an error installing kite. Please visit https://help.kite.com/article/106-linux-install-issues for possible solutions."
        echo
        echo "Keeping kite-installer in the current directory in case you'd like to try again by running:"
        echo
        echo "    $downloadFile install"
        echo
        echo "Exiting now."
        exit "$status"
    else
        rm -rf "$downloadFile"
    fi
}

mode="all"
downloadFile="$PWD/kite-installer"
while [[ $# -gt 0 ]]; do
    key="$1"
    shift
    case "$key" in
    "--help")
        cat - << EOF
    $(basename "$0") [--download [path] | --install [path]]
    Usage:
        --download [path]:  Downloads the binary installer of Kite and stores it at the given path. path defaults to ./kite-installer.
        --install [path]:   Installs Kite using the provided path to the binary installer. path defaults to ./kite-installer.
EOF
        exit 0
        ;;
    "--download")
        mode="download"
        [ -n "$1" ] && { downloadFile="$1"; shift; }
        ;;
    "--install")
        mode="install"
        [ -n "$1" ] && { downloadFile="$1"; shift; }
        ;;
    *) shift ;;
    esac
done

checkPrerequisites

case "$mode" in
"all")
    echo
    echo "This script will install Kite!"
    echo
    echo "We hope you enjoy! If you run into any issues, please report them at https://github.com/kiteco/issue-tracker."
    echo
    echo "- The Kite Team"
    echo
    read -r -e -p "Press enter to continue..."

    if [ "$(id -u)" = "0" ]; then
        echo >&2
        echo >&2 "You're installing Kite as root."
        echo >&2 "Installing as root is strongly discouraged."
        echo >&2
        echo >&2 "Please make sure that you really want to install as root."
        echo >&2
        read -r -e -p "Do you want to continue? [y/N] " ROOT_INSTALL
        if [[ "$ROOT_INSTALL" != [Yy] ]]; then
          exit 20
        fi
    fi

    echo
    if [[ -z "$DISPLAY" && -z "$WAYLAND_DISPLAY" ]]; then
      echo "No X11 or Wayland session was found."
      echo "Kite Copilot UI won't be launched after the installation."
      echo "To login, run this command in a terminal:"
      echo -e "\t~/.local/share/kite/login-user"
      echo
      read -r -e -p "Press enter to continue..."
    else
      promptX11Dependencies
    fi
    downloadKiteInstaller "$downloadFile"
    installKite "$downloadFile"
    echo "Removing kite-installer"
    rm -f "$downloadFile"
    ;;
"download")
    downloadKiteInstaller "$downloadFile"
    downloadKitePackage "$downloadFile"
    ;;
"install")
    [ -f "$downloadFile" ] || downloadKiteInstaller "$downloadFile"
    installKite "$downloadFile" "--no-launch"
    ;;
esac
