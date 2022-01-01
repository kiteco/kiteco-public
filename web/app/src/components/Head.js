import React from 'react'

import Helmet from 'react-helmet'

import { Domains } from '../utils/domains'

const Head = () => <Helmet>
  <title>Kite - AI Autocomplete for Python</title>
  {/* Default meta tag values are provided below -> see this comment https://github.com/nfl/react-helmet/issues/341#issuecomment-456222290  
      Any property getting added to the <head> in possibly multiple contexts should add a default entry here and in the index.html
      with a `data-react-helmet="true"` property
  */}
  <meta name="description" content="Kite is a free autocomplete for Python developers. Code faster with the Kite plugin for your code editor, featuring Line-of-Code Completions and cloudless processing."/>
  <meta property="og:url" content={`https://${Domains.WWW}`}/>
  <meta property="og:image" content={`https://${Domains.WWW}/share-image-2.png`}/>
  <meta property="og:title" content="Code Faster with Line-of-Code Completions, Cloudless Processing"/>
  <meta property="og:description" content="Kite is a free autocomplete for Python developers. Code faster with the Kite plugin for your code editor, featuring Line-of-Code Completions and cloudless processing."/>
</Helmet>

export default Head
