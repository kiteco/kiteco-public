# Welcome to...
#
#         `hmy+.               ://:
#        .mMMMMMNho:`          NMMm
#       :NMMMMMMMMMMMds/.`     NMMm            :ss:
#      +NMMMMMMMMMMMMMMMMmy+   NMMm           -MMMM-   ---
#    `oMMMMMMMMMMMMMMMMMMMMo   NMMm            /ss/   :MMM+
#   `yMMMMMMMMNshmNMMMMMMMN`   NMMm                   /MMM+
#  .dMMMMMMMMm/hmhssydmMMM+    NMMm    `/yhhy. shhy ohmMMMmhhhh.  ./ydmmmdho-
#  omMMMMMMMd/mMMMMMmhsosy`    NMMm  .omMMmo.  mMMN odmMMMmdddd. omMNdsoshNMNy`
#   .+dMMMMy/mMMMMMMMMMMm-     NMMm-yNMMh/`    mMMN   /MMM+     sMMN:`   `:NMMy
#     `-ymo/NMMMMMMMMMMMd      NMMMNMMN/       mMMN   :MMM+     MMMNdddddddNMMN
#        ``hMMMMMMMMMMMM:      NMMm+mMMNs.     mMMN   :MMM+     MMMh//////////:
#          `:yNMMMMMMMMh       NMMm `/dMMNy-   mMMN   :MMM+  `. sMMNo`    `-:
#             .+mMMMMMM-       NMMm   `/dMMNy- mMMN   .MMMNddNN/ +NMMNdhydNNMs
#               `:yMMMy        yhhs     `/hhhh shhs    :ymmmdho:  `/sdmmmmhs/`
#                  `om.


""" Kite is your Python programming copilot. Kite will try to show you the
    right information at the right time as you code to prevent you from context
    switching out of your current line of thought.

    This tutorial will teach you how to use all of Kite's core features. You
    should be able to learn everything in 5 minutes.

    If you get stuck at any point, please visit https://help.kite.com/ or file
    an issue at https://github.com/kiteco/issue-tracker.
"""




""" PART 0: BEFORE WE START ===================================================

    Spyder will by default try to start the Kite backend when the editor first
    starts. You can change this behavior by opening settings, clicking on
    "Completion and linting", "Advanced", and then changing Kite's "Start Kite
    Engine on editor startup" setting.

    Look for the Kite indicator in the bottom left corner of Spyder's status
    bar — It will tell you if Kite is ready and working. If the indicator reads
    "not running", then you'll have to start the Kite Engine manually before
    proceeding with the rest of this tutorial.
"""




""" PART 1: CODE COMPLETIONS ==================================================

    Kite analyzes your code and uses machine learning to show you completions
    for what you're going to type next.

    If you have your editor configured to show autocompletions, then Kite will
    show you completions automatically as you type.

    If you don't have autocompletions on, you can press ctrl+space to request
    completions at any time.

    You can toggle autocompletions in the editor settings by clicking on
    "Completion and linting", and then changing the "Show completions on the
    fly" setting.

    IMPORTANT: We also recommend changing the "Show automatic completions after
    characters entered" setting to 1 and the "Show automatic completions after
    keyboard idle (ms)" setting to 100 or less. The rest of this tutorial may
    not work properly until you have done so!
"""


# 1a. Name completions
#
# Kite can suggest names of symbols to use, such as names of packages or names
# of variables in scope.

# TRY IT
# ------
# • Put your cursor at the end of the line marked with "<--".
# • Type "a" and select the completion for "matplotlib". (The rest of this
#   tutorial depends on you doing so!)
# • Remember to press ctrl+space if autocompletions aren't on.

import m  # <--


# 1b. Attribute completions
#
# Type a "." after a name and Kite will show you the attributes available.

# TRY IT
# ------
# • Put your cursor at the end line of the line marked with "<--".
# • Type "." and select the completion for "pyplot".
# • Remember to press ctrl+space if autocompletions aren't on.

import matplotlib  # <--


# 1c. Many, many more completions than the language server
#
# Kite analyzes data analysis libraries such as matplotlib much more
# intelligently than Spyder's builtin language server. As a result, you will
# see many more completions when coding with Kite.

# TRY IT
# ------
# • Put your cursor at the end of the line marked with "<--".
# • Type "." and see the completions available for the Figure object.
# • Remember to press ctrl+space if autocompletions aren't on.
# • Typing the same code without Kite enabled would result in no completions
#   being shown because the builtin language server cannot analyze the code
#   properly.

import matplotlib.pyplot as plt
fig = plt.figure()
fig  # <--


# 1d. Code completions on demand
#
# Remember that you can use a keyboard shortcut at any time to request code
# completions.

# TRY IT
# ------
# • Put your cursor at the end of the line marked with "<--".
# • Press ctrl+space to request code completions to see the attributes in the
#   plt module.

plt.  # <--




""" PART 2: FUNCTION ASSIST ===================================================

    Kite can also show you how to use a function as you're calling it in your
    code.
"""


# 2a. Function signatures
#
# When you're calling a function, Kite will show you the function's signature
# to help you remember what arguments you need to pass in.

# TRY IT
# ------
# • Put your cursor at the end of line marked with "<--".
# • Type "(" to start the function call, and Kite will show you how to call
#   plt.plot.

plt.plot  # <--


# 2b. Learning from your own code
#
# Kite will also show you signatures for functions that you have defined in
# your own code.

# TRY IT
# ------
# • Put your cursor at the end of the line marked with "<--".
# • Type "(" to get assistance for your locally defined pretty_print function.


def pretty_print(obj, indent=2):
    print(json.dumps(obj, indent=indent))

pretty_print(obj, indent=4)

pretty_print  # <--



""" PART 3: INSTANT DOCUMENTATION =============================================

    Kite can also show you documentation for the symbols in your code in the
    Copilot application.

    To do so, open Kite's Copilot, ensure that the button labeled "Click for
    docs to follow cursor" in the upper right corner is enabled, and then
    simply position your cursor over a symbol.

    To open Kite's Copilot, visit the URL kite://home in your browser.
"""


# TRY IT
# ------
# • Position your cursor over "fig" by either clicking on it or using your
#   keyboard's arrow keys.
# • Documentation for the Figure class will be shown in Kite's Copilot.

fig



""" That's it!

    Now you know how to use Kite's core features to boost your productivity as
    you code. You can learn more about Kite's Spyder integration at our help
    page:

    https://help.kite.com/category/89-spyder-integration

    If you get stuck at any point, please visit https://help.kite.com/ or file
    an issue at https://github.com/kiteco/issue-tracker.
    
    ____________________________________________________________________________

    Kite is under active development. You can expect its features to become
    smarter and more featured over time.
"""
