import constants from './theme-constants'
const { PAGE_KIND } = constants

export const USER_ID_INDEX = 1
export const PATH_ID_INDEX = 4

export const isLocalId = (id) => {
  return id.split(';')[1] !== ''
}

export const queryIdToRoute = (id) => {
  return isLocalId(id)
    ? `/${id}`
    : id.replace('python', '')
      .replace(/;/g, '.')
      .replace(/\.{2,}/, '/')
}

export const oldToNewPath = (pathname) => {
  if(pathname.includes(';')) {
    const patharr = pathname.split(";")
    switch(patharr.length) {
      case 2:
        //e.g. "python;re.compile"
        return `/python/docs/${pathname.substr(pathname.lastIndexOf(";") + 1)}`
      case 5:
        //form: [language];[userID];[machineID];[colon delineated filepath];[dottedPath]
        //local example: python;7;3e51855fc9b0accd7bbe1eafebc9b967;:Users:dane:Desktop:stuff:whitelist:foo.py;Foo.foo
        //global example: python;;;;re
        //here we assume a value id... which could be global or local
        const valueIdPath = patharr[1] === ''
          ? patharr.slice(PATH_ID_INDEX).join(".") //global
          : patharr.slice(USER_ID_INDEX).join(';') //local
        return `/python/docs/${valueIdPath}`
      case 6:
        //form: [language];[userID];[machineID];[colon delineated filepath];[dottedPath];[symbol]
        //local example: python;7;3e51855fc9b0accd7bbe1eafebc9b967;:Users:dane:Desktop:stuff:whitelist:foo.py;Foo;foo
        //global example: python;;;;;re
        //here we assume a symbol id

        const symbolIdPath = patharr[1] === ''
          ? patharr.slice(PATH_ID_INDEX).filter(part => part !== '').join('.') //global assumption => no userId
          : patharr.slice(USER_ID_INDEX).join(';') //local assumption - recreate symbolId w/o language

        return `/python/docs/${symbolIdPath}`
      default:
        return '/python'
    }
  } else if(/\/docs\/python\/\w+/.test(pathname) || /\/docs\/\w+/.test(pathname)) {
    //for old google indexed route cases
    //and routes of the form /docs/<symbol.dotted.path>
    return `/python/docs/${pathname.substr(pathname.lastIndexOf('/') + 1)}`
  } else {
    return '/python/docs'
  }
}

export const oldToNewExamplesPath = (pathname) => {
  //for old google indexed route cases
  // with an exact `/examples` match, we redirect to the docs page, as we don't have an
  // `/examples` index page currently
  if(pathname === '/examples') {
    return '/python/docs'
  }
  // distinguish between /examples/<id>/<title> and /examples/<id> case
  if(/^\/examples\/\d+\/[^/\s]+$/.test(pathname)) {
    return `/python${pathname}`
  }
  return `/python/examples/${pathname.substr(pathname.lastIndexOf('/') + 1)}`
}

export const transformQueryCompletions = (data) => {
  return data.results.map(result => ({
    ...result.result,
    path: `/${data.language}/docs${queryIdToRoute(result.result.id)}`
  }))
}

export const idFromPath = (language, valuePath) => {
  return `${language};${valuePath.replace(/\//g, '.')}`
}

export const parsePageKind = (match) => {
  //then HOWTO route
  if(match.params.exampleId) {
    return PAGE_KIND.HOWTO
  }
  //then IDENTIFIER route
  if(match.params.valuePath) {
    return PAGE_KIND.IDENTIFIER
  }
}

export const getIdFromLocation = (location) => {
  const tokens = location.pathname.split("/")
  //looking for numeric id
  return tokens.find(token => token && !isNaN(token))
}

export const hasRawId = (pathname) => {
  pathname = pathname.substring(pathname.lastIndexOf('/') + 1)
  if(pathname.includes(';')) {
    const tokens = pathname.split(';')
    return tokens[0] === 'python'
  }
  return false
}
