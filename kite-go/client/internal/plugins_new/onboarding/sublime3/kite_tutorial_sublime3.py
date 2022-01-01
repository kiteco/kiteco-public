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

    Kite's Sublime package will by default try to start the Kite backend when
    the editor first starts. You can change this behavior by opening the Kite
    package's settings, and changing "start_kite_engine_on_startup".

    Look for the 𝕜𝕚𝕥𝕖 symbol in the bottom left corner of Sublime's status
    bar — It will tell you if Kite is ready and working. If the indicator reads
    "Connection error", then you'll have to start the Kite Engine manually
    before proceeding with the rest of this tutorial.
"""




""" PART 1: CODE COMPLETIONS ===================================================

    Kite analyzes your code and uses machine learning to show you completions
    for what you're going to type next.

    If you have your editor configured to show autocompletions, then Kite will
    show you completions automatically as you type.

    If you don't have autocompletions on, you can press ctrl+space to request
    completions at any time.

    You can toggle autocompletions by changing Sublime's native "auto_complete"
    setting to true or false.

    Look for Kite's "⟠" symbol on the right-hand side to see which completions
    are coming from Kite.
"""


# 1a. Conflicting packages
#
# If you have other 3rd party packages which provide completions, your
# experience with Kite may not be ideal because these packages may conflict
# with Kite's behavior. We suggest temporarily disabling completions from other
# packages while you try out Kite. You won't be disappointed!


# 1b. Name completions
#
# Kite can suggest names of symbols to use, such as names of packages or names
# of variables in scope.

# TRY IT
# ------
# • Put your cursor at the end of the line marked with "<--".
# • Type "s" and select the completion for "json". (The rest of this tutorial
#   depends on you doing so!)
# • Remember to press ctrl+space if autocompletions aren't on.

import j  # <--


# 1c. Attribute completions
#
# Type a "." after a name and Kite will show you the attributes available.

# TRY IT
# ------
# • Put your cursor at the end line of the line marked with "<--".
# • Type "." and select the completion for "dumps".
# • Remember to press ctrl+space if autocompletions aren't on.

json  # <--


# 1d. Code completions on demand
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

    By default, Kite will show you this information automatically as you're
    coding when it detects that your cursor is inside a function call.

    You can prevent this UI from being shown automatically by changing Kite's
    "show_function_signatures" setting to false.

    You can manually request function assist at any time by pressing
    ctrl+alt+u. However, your cursor must be inside a function call for the UI
    to appear.

    You can hide the function assist UI by pressing escape.
"""


# 2a. Function signatures and more
#
# When you're calling a function, Kite will show you the function's signature
# to help you remember what arguments you need to pass in. It may also show you
# examples of how other developers use the function and the keyword arguments
# you can use.

# TRY IT
# ------
# • Put your cursor at the end of line marked with "<--".
# • Type "(" to start the function call, and Kite will show you how to call
#   json.dumps.
# • Remember to press ctrl+alt+u after typing "(" if you've disabled function
#   assist from happening automatically.
#
# • Within the UI, click on the "Examples" link to see how other developers use
#   the function.
# • You can hide this information by clicking on the "Hide" link.

json.dumps  # <--


# 2b. Learning from your own code
#
# Kite will also show you signatures, example usages, and keyword arguments for
# functions that you have defined in your own code.

# TRY IT
# ------
# • Put your cursor at the end of the line marked with "<--".
# • Type "(" to get assistance for your locally defined pretty_print function.
# • Remember to press ctrl+alt+u after typing "(" if you've disabled function
#   assist from happening automatically.

def pretty_print(obj, indent=2):
    print(json.dumps(obj, indent=indent))

pretty_print(obj, indent=4)

pretty_print  # <--


# 2c. Function assist on demand
#
# Remember that you can use a keyboard shortcut at any time to view information
# about a function.

# TRY IT
# ------
# • Put your cursor between the "(" and ")" on the line marked with "<--".
# • Press ctrl+alt+u to access function assist.

pretty_print()  # <--



""" PART 3: INSTANT DOCUMENTATION ==============================================

    Kite can also show you documentation for the symbols in your code.

    If Kite's "show_hover" setting is true, then you can access documentation
    by hovering your mouse over a symbol and then clicking the "Docs" link.

    Otherwise, you can also position your cursor over a symbol, and then press
    ctrl+alt+d to access documentation.
"""


# TRY IT
# ------
# • Hover your mouse over "dumps" and then click "Docs".
# • Or put your cursor over "dumps" and then press ctrl+alt+d.

json.dumps




""" That's it!

    Now you know how to use Kite's core features to boost your productivity as
    you code. You can access this tutorial at any time by running the command
    "Kite: Python Tutorial" from the command palette.

    You can learn more about Kite's Sublime package at its GitHub repo:

    https://github.com/kiteco/KiteSublime


    ____________________________________________________________________________

    Kite is under active development. You can expect its features to become
    smarter and more featured over time.

    We love hearing from you! Vist https://github.com/kiteco/issue-tracker at
    any time to report issues or submit feature requests.
"""
