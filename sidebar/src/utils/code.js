/**
 * Code blocks that we get from the backend are plaintext.
 *
 * Sometimes these also include a list of references,
 * which are demarcated by a list of positions.
 *
 * This utility package uses Prism to perform syntax highlighting
 * through tokenization.
 * We then attach references to each token based on position
 * of the token and references from the backend.
 *
 */
import Prism from 'prismjs'

/* ==={ TOKENIZERS }=== */
/**
 * From a block of code from the backend, create a list of tokens
 * using Prism.
 */

const createPythonTokens = code => {
  return Prism.tokenize(code, Prism.languages.python).map(token => {
    if (typeof token === 'object') {
      return {
        content: token.content,
        type: token.type,
      }
    } else {
      return {
        content: token,
        type: "unassigned",
      }
    }
  });
}

const languageToTokenizer = {
  "python": createPythonTokens,
}


/* ==={ CODE PROCESSING }=== */

// From tokenized lines and refs, create references
const assignRefs = (lines, refs) => {
  if (refs) {
    const newLines = [];
    let currentPos = 0;
    for (let i = 0; i < lines.length; i += 1) {
      let newTokens = [];
      for (let j = 0; j < lines[i].length; j += 1) {
        let currentToken = lines[i][j];
        for (let k = 0; k < refs.length; k += 1) {
          let ref = refs[k];
          if (ref.begin >= currentPos &&
            ref.end <= (currentPos + currentToken.content.length)) {
            const start = ref.begin - currentPos;
            const end = ref.end - currentPos;
            const startCap = {
              type: currentToken.type,
              content: currentToken.content.slice(0, start),
            }
            const reference = {
              type: currentToken.type,
              content: currentToken.content.slice(start, end),
              ref: ref,
            }
            const endCap = {
              type: currentToken.type,
              content: currentToken.content.slice(end, currentToken.content.length),
            }
            if (startCap.content) {
              newTokens.push(startCap);
            }
            newTokens.push(reference);
            currentToken = endCap;
            currentPos = ref.end;
          }
        }
        // remove empty tokens
        if (currentToken.content) {
          newTokens.push(currentToken);
          currentPos += currentToken.content.length;
        }
      }
      newLines.push(newTokens);
    }
    return newLines;
  } else {
    return lines;
  }
}

// From a list of tokens, sort these into a list of lines
// [tokens] => [[tokens], [tokens], [tokens]] where each sublist
// is a new line.
//
// This allows us to easily render the code block in a table
const splitTokensByLine = (tokens) => {
  let currentLineTokens = [];
  let lines = [];
  let currentToken = null;
  const newToken = type => content => ({type, content});
  const arrayWrap = t => ([t]);
  for (let i = 0; i < tokens.length; i += 1) {
    currentToken = tokens[i];
    if (currentToken.content.indexOf("\n") > -1) {
      let newTokens = currentToken.content.split("\n").map(newToken(currentToken.type));
      currentLineTokens.push(newTokens.shift());
      lines.push(currentLineTokens);
      currentToken = newTokens.pop();
      lines = lines.concat(newTokens.map(arrayWrap));
      currentLineTokens = []
    }
    currentLineTokens.push(currentToken);
  }
  // add back new lines
  lines.forEach(line => line.push({content: "\n", type: "unassigned"}));
  lines.push(currentLineTokens);
  return lines;
}


/* ==={ PROCESSING FLOW }=== */

/**
 * This is the main processing flow.
 *
 * We tokenize using Prism first because one expression
 * may span multiple lines; we need to tokenize all lines at
 * once. This thens requires us to use splitTokensByLine.
 *
 * Finally, we take a look at the position of each token
 * and assign references if they exist.
 *
 * Ultimately this returns a list of list of objects, where
 * each object represents a token.
 */
export const processCode = (code, language, references) => {
  const tokens = languageToTokenizer[language](code)
  const lines = splitTokensByLine(tokens)
  const linesWithRefs = assignRefs(lines, references)
  return linesWithRefs
}
