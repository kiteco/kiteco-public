function updateProxySettings(session, mode, url) {
    if (!session) {
        return
    }

    let config;
    if (mode === "direct") {
        config = {
            proxyRules: "direct://",
            proxyBypassRules: "<local>",
        }
    } else if (mode === "manual" && url !== "") {
        // Electron follows the same scheme as we do for Kite
        // e.g. http://127.0.0.1:8080 and socks5://127.0.0.1:8080 are valid
        config = {
            proxyRules: url,
            proxyBypassRules: "<local>",
        }
    } else {
        // fallback is "environment", aka "system"
        // we're defaulting to use all_proxy, http_proxy and https_proxy before
        // delegating to the system's proxy settings.
        // Delegation to the system's settings is only available on macOS and Windows.
        config = getEnvironmentProxySettings()
    }

    console.log("setting proxy config " + JSON.stringify(config))
    session.setProxy(config, function(){})
}

/**
 *
 * Electrons supported settings are documented at https://electronjs.org/docs/api/session#sessetproxyconfig-callback
 *
 * @return An object in the format expected by Electron's setProxy()
 * with the values http_proxy|HTTP_PROXY and https_proxy|HTTPS_PROXY in the style used by electron. If no such values
 * are present, then Electron will delegate to the systems networking layer on Windows and Linux.
 */
function getEnvironmentProxySettings() {
    let proxyRules = ""

    // http_proxy in CURL style is only a single value of the form [protocol://]hostname:port, with the
    // protocol being optional.
    // Curl also support ALL_PROXY and NO_PROXY
    var all = anyEnvValue("ALL_PROXY")
    if (all !== "") {
        // proxyRules with a single value and without a scheme specifier means that
        // the url is used for all requests
        proxyRules = all
    } else {
        let parts = []
        let http = anyEnvValue("http_proxy", "HTTP_PROXY")
        if (http !== "") {
            parts.push(`http=${http}`)
        }
        let https = anyEnvValue("https_proxy", "HTTPS_PROXY")
        if (https !== "") {
            parts.push(`https=${https}`)
        }
        proxyRules = parts.join(";")
    }

    // fallback settings
    return {
        proxyRules: proxyRules,
        proxyBypassRules: "<local>",
    }
}

/**
 * @param keys Names of the keys to lookup
 * @return {string} The first available value for the the given environment name variables
 */
function anyEnvValue(...keys) {
    for (var i = 0; i < keys.length; i++) {
        let v = process.env[keys[i]]
        if (v) {
            return v
        }
    }
    return ""
}

module.exports = {
    updateProxySettings: updateProxySettings
}
