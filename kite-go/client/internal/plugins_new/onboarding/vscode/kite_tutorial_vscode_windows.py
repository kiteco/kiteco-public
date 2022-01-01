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


""" Kite is your programming copilot. Kite will try to show you the
    right information at the right time as you code to prevent you from context
    switching out of your current line of thought.

    This tutorial will teach you how to use all of Kite's core features. You
    should be able to learn everything in 5 minutes.

    If you get stuck at any point, please visit https://help.kite.com/ or file
    an issue at https://github.com/kiteco/issue-tracker.
"""


""" PYTHON TUTORIAL ============================================================

    Not writing Python? Open tutorials for other languages by running the
    following commands from the command palette:

    * For Javascript, run   "Kite: Javascript Tutorial"
    * For Go, run           "Kite: Go Tutorial"
"""


""" PART 0: BEFORE WE START ====================================================

    Kite's VS Code extension will by default try to start the Kite backend when
    the editor first starts. You can change this behavior by opening settings,
    searching for "Kite:", and changing "Kite: Start Kite Engine On Startup".

    Look for the 𝕜𝕚𝕥𝕖 indicator in the bottom right corner of VS Code's status
    bar — It will tell you if Kite is ready and working. If the indicator reads
    "not running", then you'll have to start the Kite Engine manually before
    proceeding with the rest of this tutorial.
"""




""" PART 1: CODE COMPLETIONS ===================================================

    Kite analyzes your code and uses machine learning to show you completions
    for what you're going to type next.

    If you have your editor configured to show autocompletions, then Kite will
    show you completions automatically as you type.

    If you don't have autocompletions on, you can press ctrl+space to request
    completions at any time.

    You can toggle autocompletions in the editor settings by changing
    "Editor: Suggest On Trigger Characters".

    Look for a stylized 𝕜𝕚𝕥𝕖 on the right-hand side to see which completions
    are coming from Kite.
"""


# 1a. Name completions
#
# Kite can suggest names of symbols to use, such as names of packages or names
# of variables in scope.

# TRY IT
# ------
# • Put your cursor at the end of the line marked with "<--".
# • Type "s" and select the completion for "json". (The rest of this tutorial
#   depends on you doing so!)
# • Remember to press ctrl+space if autocompletions aren't on.

import j  # <--


# 1b. Attribute completions
#
# Type a "." after a name and Kite will show you the attributes available.

# TRY IT
# ------
# • Put your cursor at the end line of the line marked with "<--".
# • Type "." and select the completion for "dumps".
# • Remember to press ctrl+space if autocompletions aren't on.

json  # <--


# 1c. Code completions on demand
#
# Remember that you can use a keyboard shortcut at any time to request code
# completions.

# TRY IT
# ------
# • Put your cursor at the end of the line marked with "<--".
# • Press ctrl+space to request code completions to see the attributes in the
#   json module.

json.  # <--




""" PART 2: FUNCTION ASSIST ====================================================

    Kite can also show you how to use a function as you're calling it in your
    code.

    If you have your editor configured to show parameter info automatically,
    then Kite will show you this information automatically as you're coding
    when it detects that your cursor is inside a function call.

    You can prevent this UI from being shown automatically in the editor
    preferences by changing "Editor > Parameter Hints".

    You can manually request function assist at any time by pressing
    ctrl+shift+space. However, your cursor must be inside a function call for
    the UI to appear.

    You can hide the function assist UI by pressing escape (or shift+escape if
    you are using the Vim extension).
"""


# 2a. Conflicting extensions
#
# If you are using Microsoft's Python extension, Kite will not be able to help
# you call functions, even when the Microsoft extension has no information to
# show you. Unfortunately there is no workaround for this at the moment.


# 2b. Function signatures
#
# When you're calling a function, Kite will show you the function's signature
# to help you remember what arguments you need to pass in.

# TRY IT
# ------
# • Put your cursor at the end of line marked with "<--".
# • Type "(" to start the function call, and Kite will show you how to call
#   json.dumps.
# • Remember to press ctrl+shift+space after typing "(" if you've disabled
#   function assist from happening automatically.

json.dumps  # <--


# 2c. Learning from your own code
#
# Kite will also show you signatures for functions that you have defined in
# your own code.

# TRY IT
# ------
# • Put your cursor at the end of the line marked with "<--".
# • Type "(" to get assistance for your locally defined pretty_print function.
# • Remember to press ctrl+shift+space after typing "(" if you've disabled
#   function assist from happening automatically.

def pretty_print(obj, indent=2):
    print(json.dumps(obj, indent=indent))

pretty_print(obj, indent=4)

pretty_print  # <--


# 2d. Function assist on demand
#
# Remember that you can use a keyboard shortcut at any time to view information
# about a function.

# TRY IT
# ------
# • Put your cursor between the "(" and ")" on the line marked with "<--".
# • Press ctrl+shift+space to access function assist.

pretty_print()  # <--




""" PART 3: INSTANT DOCUMENTATION ==============================================

    Kite can also show you documentation for the symbols in your code.

    If the "Editor > Hover" setting is enabled, then you can access
    documentation by hovering your mouse over a symbol and then clicking on
    Kite's "Docs" link.

    Otherwise, you can also position your cursor over a symbol, and run the
    "Kite: Docs At Cursor" command to access documentation.
"""


# TRY IT
# ------
# • Hover your mouse over "dumps" and then click "Docs".
# • Or put your cursor over "dumps", press ctrl+shift+p, and run the command
#   "Kite: Docs At Cursor".

json.dumps




""" That's it!

    Now you know how to use Kite's core features to boost your productivity as
    you code. You can access this tutorial at any time by running the command
    "Kite: Python Tutorial" from the command palette.

    You can learn more about Kite's VS Code extension at its GitHub repo:

    https://github.com/kiteco/vscode-plugin


    ____________________________________________________________________________

    Kite is under active development. You can expect its features to become
    smarter and more featured over time.

    We love hearing from you! Vist https://github.com/kiteco/issue-tracker at
    any time to report issues or submit feature requests.
"""
