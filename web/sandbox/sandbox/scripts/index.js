import {
  TAB_NUMERICS,
  TAB_SCRIPTING,
  TAB_WEB,
} from '../constants'

export const TYPE_CHARS = 'chars'
export const TYPE_COMPLETION = 'completion'
export const TYPE_PAUSE = 'pause'
export const TYPE_HELPER = 'helper'
export const TYPE_CAPTION = 'caption'

export const AMOUNT = 'amount'

export const RIGHT_CURSOR_CAPTION_PLACEMENT = 'right'
export const BOTTOM_CURSOR_CAPTION_PLACEMENT = 'bottom'

// ms
export const DEFAULT_LATENCY_CHAR = 100
export const FALLBACK_LATENCY_CHAR = 100
export const DEFAULT_LATENCY_PAUSE = 1000
export const DEFAULT_LATENCY_COMPLETION = 100
export const DEFAULT_LATENCY_INTER_ACTION = 0
export const DEFAULT_PRE_COMPLETION_LATENCY = 500
export const DEFAULT_LATENCY_FINAL_SELECTION = 100
export const TIMEOUT_BUFFER_RATIO = 0.15

export const scripts = {
  [TAB_NUMERICS]: {
    script: [
      // {
      //   type: TYPE_CHARS,
      //   sequence: '# Let\'s plot a graph using Kite\'s completions to speed up our development time\n\n# Start by importing the usual suspects\n',
      //   [AMOUNT]: 5
      // },
      // {
      //   type: TYPE_PAUSE
      // },
      {
        type: TYPE_CHARS,
        sequence: 'import n',
        [AMOUNT]: 100
      },
      {
        type: TYPE_COMPLETION,
        select: 'numpy',
        complete: 'umpy',
        finalSelectionWait: 2500,
        cursorCaption: 'A coding engine that knows about the entire Python universe',
        cursorCaptionPlacement: RIGHT_CURSOR_CAPTION_PLACEMENT,
        marginTop: -2,
        marginLeft: -15,
      },
      {
        type: TYPE_CHARS,
        sequence: ' '
      },
      {
        type: TYPE_COMPLETION,
        select: 'numpy as np',
        complete: 'as np',
        finalSelectionWait: 2000,
        cursorCaption: 'And common patterns from Github',
        cursorCaptionPlacement: RIGHT_CURSOR_CAPTION_PLACEMENT,
        marginLeft: -10,
      },
      // {
      //   type: TYPE_PAUSE,
      //   [AMOUNT]: 1000
      // },
      {
        type: TYPE_CHARS,
        sequence: '\nimport m',
        [AMOUNT]: 100
      },
      // {
      //   type: TYPE_CHARS,
      //   sequence: 'atploblib',
      //   [AMOUNT]: 0
      // },
      {
        type: TYPE_COMPLETION,
        select: 'matplotlib',
        complete: 'atplotlib',
        [AMOUNT]: 10
      },
      {
        type: TYPE_CHARS,
        sequence: '.'
      },
      {
        type: TYPE_COMPLETION,
        select: 'pyplot',
        complete: 'pyplot',
        finalSelectionWait: 2000,
        cursorCaption: 'With intelligently ranked and more relevant completions.',
        cursorCaptionPlacement: RIGHT_CURSOR_CAPTION_PLACEMENT,
        marginLeft: -5,
      },
      {
        type: TYPE_CHARS,
        sequence: ' as plt\n\nx = np.linspace(-1, 1)\ny = np.sin(x)',
        [AMOUNT]: 30,
        skipCompletions: true,
      },
      {
        type: TYPE_CHARS,
        sequence: '\npl',
        [AMOUNT]: 250
      },
      {
        type: TYPE_COMPLETION,
        select: 'plt.plot(x, y)',
        complete: 't.plot(x, y)',
        finalSelectionWait: 8000,
        cursorCaption: 'Whoa — Kite knows I want to call “plot”, and which arguments to pass in.\n\nWith one keystroke, Kite’s Line-of-Code Completions save me typing and keep me in flow',
        cursorCaptionPlacement: RIGHT_CURSOR_CAPTION_PLACEMENT,
        marginTop: -1,
        marginLeft: -20,
        afterClass: 'exploding-head',
      },
      {
        type: TYPE_CHARS,
        sequence: '\n\ntitle = "Plot"\nfilename = "plot.jpg"\n\n',
        [AMOUNT]: 30,
        skipCompletions: true,
      },
      {
        type: TYPE_CHARS,
        sequence: 'plt.titl',
        [AMOUNT]: 70,
      },
      {
        type: TYPE_COMPLETION,
        select: 'title(title)',
        complete: 'e(title)',
        finalSelectionWait: 2000,
        cursorCaption: "Kite knows which variable fits here.",
        cursorCaptionPlacement: RIGHT_CURSOR_CAPTION_PLACEMENT,
        marginLeft: -40,
      },
      {
        type: TYPE_CHARS,
        sequence: '\nplt.sa',
        [AMOUNT]: 150
      },
      {
        type: TYPE_COMPLETION,
        select: 'savefig(filename)',
        complete: 'vefig(filename)',
        finalSelectionWait: 4000,
        cursorCaption: "Once again Kite's AI helps me jump several steps ahead.",
        cursorCaptionPlacement: RIGHT_CURSOR_CAPTION_PLACEMENT,
        marginLeft: -20,
      },
      {
        type: TYPE_PAUSE,
        [AMOUNT]: 1500
      },
    ],
    filledBuffer: 'import numpy as np\nimport matplotlib.pyplot as plt\n\nx = np.linspace(-1, 1)\ny = np.sin(x)\nplt.plot(x, y)\n\ntitle = "Plot"\nfilename = "plot.jpg"\n\nplt.title(title)\nplt.savefig(filename)',
    startBuffer: "",
  },
  [TAB_SCRIPTING]: {
    script: [
      {
        type: TYPE_CHARS,
        sequence: "# WELCOME TO OUR SANDBOX!\n\n# THE SCRIPT HERE IS JUST A STUB.\n# BUT YOU CAN SIMPLY CLICK AND START TYPING!\n",
        [AMOUNT]: 7
      },
      {
        type: TYPE_CHARS,
        sequence: 'import json',
      },
      {
        type: TYPE_COMPLETION,
        select: 'jsonrpclib',
        complete: 'rpclib',
      },
      {
        type: TYPE_CHARS,
        sequence: '\njso',
      },
      {
        type: TYPE_COMPLETION,
        select: 'jsonrpc',
        complete: 'nrpc'
      },
      {
        type: TYPE_CHARS,
        sequence: '.',
      },
      {
        type: TYPE_PAUSE,
        amount: 2000
      }
    ],
    filledBuffer: "# WELCOME TO OUR SANDBOX!\n\n# THE SCRIPT HERE IS JUST A STUB.\n# BUT YOU CAN SIMPLY CLICK AND START TYPING!\nimport jsonrpc\njsonrpc.",
    startBuffer: "",
  },
  [TAB_WEB]: {
    script: [
      {
        type: TYPE_CHARS,
        sequence: "# WELCOME TO OUR SANDBOX!\n\n# THE SCRIPT HERE IS JUST A STUB.\n# BUT YOU CAN SIMPLY CLICK AND START TYPING!\n",
        [AMOUNT]: 7
      },
      {
        type: TYPE_CHARS,
        sequence: 'import json',
      },
      {
        type: TYPE_COMPLETION,
        select: 'jsonrpclib',
        complete: 'rpclib',
      },
      {
        type: TYPE_CHARS,
        sequence: '\njso',
      },
      {
        type: TYPE_COMPLETION,
        select: 'jsonrpc',
        complete: 'nrpc'
      },
      {
        type: TYPE_CHARS,
        sequence: '.',
      },
      {
        type: TYPE_PAUSE,
        amount: 2000
      }
    ],
    filledBuffer: "# WELCOME TO OUR SANDBOX!\n\n# THE SCRIPT HERE IS JUST A STUB.\n# BUT YOU CAN SIMPLY CLICK AND START TYPING!\nimport jsonrpc\njsonrpc.",
    startBuffer: "",
  },
  simple: {
    script: [
      {
        type: TYPE_CHARS,
        sequence: 'import j',
      },
      {
        type: TYPE_COMPLETION,
        select: 'jsonrpc',
        complete: 'sonrpc',
      },
      {
        type: TYPE_CHARS,
        sequence: '\njso',
      },
      {
        type: TYPE_COMPLETION,
        select: 'jsonrpc',
        complete: 'nrpc'
      },
      {
        type: TYPE_CHARS,
        sequence: '.',
      },
      {
        type: TYPE_PAUSE,
        amount: 2000
      }
    ],
    filledBuffer: "# TRY IT YOURSELF!\nimport jsonrpc\njsonrpc.",
    startBuffer: "# SIMPLE SCRIPT\n",
  }
}
