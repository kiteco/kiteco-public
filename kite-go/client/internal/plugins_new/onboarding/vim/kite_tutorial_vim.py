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


""" Kite is your programming copilot. Kite will try to show you the right 
    information at the right time as you code to prevent you from context
    switching out of your current line of thought.

    This tutorial will teach you how to use all of Kite's core features. You
    should be able to learn everything in 5 minutes.

    If you get stuck at any point, please visit https://help.kite.com/ or file
    an issue at https://github.com/kiteco/issue-tracker.
"""


""" PYTHON TUTORIAL ============================================================

    Kite's Vim plugin only supports Python right now. Support for other 
    languages is coming soon. 
"""



""" PART 0: BEFORE WE START ====================================================


    Kite's Vim plugin will by default try to start the Kite backend when the
    editor first starts. You can disable this behavior with the command
    "KiteDisableAutoStart" and enable it with "KiteEnableAutoStart".

    Add "%{kite#statusline()}" to your statusline to see the status of the Kite
    Engine. If you don't have a statusline, run these commands to get a basic
    statusline:

    set statusline=%<%f\ %h%m%r%{kite#statusline()}%=%-14.(%l,%c%V%)\ %P
    set laststatus=2

    Make sure that Kite's status reads "Kite" - This means that Kite is ready
    and working. If the indicator reads "not running", then you'll have to
    start the Kite Engine manually before proceeding with the rest of this
    tutorial.
"""



""" PART 1: CODE COMPLETIONS ===================================================

    Kite analyzes your code and uses machine learning to show you completions
    for what you're going to type next.

    If you have your editor configured to show autocompletions, then Kite will
    show you completions automatically as you type.

    If you don't have autocompletions on, you can press <C-x><C-u> while in
    insert mode to request completions at any time.

    You can disable autocompletions with "let g:kite_auto_complete=0" and
    enable it with "let g:kite_auto_complete=1".

    Use <C-n> to choose the next completion and <C-p> to choose the previous
    completion. (You can also use the up and down arrows for this if you're
    using a GUI.)

    Press <C-x> to hide the completions UI.

    Use <C-y> to select a completion. You can also use <Tab> to select a
    completion by setting "let g:kite_tab_complete=1".

    Look for the Kite symbol on the right-hand side to see which completions
    are coming from Kite.
"""


# 1a. Conflicting plugins
#
# If you have other 3rd party plugins which provide completions, your
# experience with Kite may not be ideal because these plugins may conflict
# with Kite's behavior. We suggest temporarily disabling completions from other
# plugins while you try out Kite. You won't be disappointed!


# 1b. Configuring completions
#
# You can configure how completions behave with the completeopt option. If you
# haven't configured completeopt yourself, Kite configures it like so:
#
# set completeopt-=menu
# set completeopt+=menuone   " Show the completions UI even with only 1 item
# set completeopt-=longest   " Don't insert the longest common text
# set completeopt-=preview   " Hide the documentation preview window
# set completeopt+=noinsert  " Don't insert text automatically
# set completeopt-=noselect  " Highlight the first completion automatically
#
# Make sure that you either have "menu" or "menuone" set. Otherwise Kite will
# not be able to display the completions UI.
#
# If you have "preview" set, Kite will show you documentation for the currently
# highlighted completion in a separate window. If you have this set and would
# like the documentation window to be closed automatically after a completion
# is selected, use "autocmd CompleteDone * if !pumvisible() | pclose | endif".
#
# Note that setting "longest" and/or unsetting "noinsert" may cause undesired
# behavior where Vim automatically inserts text into the buffer without any
# confirmation. Therefore, we recommend using "set completeopt-=longest" and
# "set completeopt+=noinsert".


# 1c. Name completions
#
# Kite can suggest names of symbols to use, such as names of packages or names
# of variables in scope.

# TRY IT
# ------
# * Put your cursor at the end of the line marked with "<--".
# * Type "s" and select the completion for "json" with <C-y>. (The rest of this
#   tutorial depends on you doing so!)
# * Remember to press <C-x><C-u> if autocompletions aren't on.

import j  # <--


# 1d. Attribute completions
#
# Type a "." after a name and Kite will show you the attributes available.

# TRY IT
# ------
# * Put your cursor at the end of the line marked with "<--".
# * Type "." and select the completion for "dumps" with <C-y>.
# * Remember to press <C-x><C-u> if autocompletions aren't on.

json  # <--


# 1e. Code completions on demand
#
# Remember that you can use a keyboard shortcut at any time to request code
# completions.

# TRY IT
# ------
# * Put your cursor at the end of the line marked with "<--".
# * Enter insert mode.
# * Press <C-x><C-u> to request code completions to see the attributes in the
#   json module.

json.  # <--




""" PART 2: FUNCTION ASSIST ====================================================

    Kite can also show you how to use a function as you're calling it in your
    code.

    By default, Kite will show you this information automatically as you're
    coding when it detects that your cursor is inside a function call.

    Function assist is governed by the same logic as completions. This means:

    * Use <C-x><C-u> to request function assist
    * Use <C-x> to hide the UI
    * Set "let g:kite_auto_complete=0" to prevent function assist from
      occurring automatically (this will also stop autocompletions!)
"""


# 2a. Function signatures
#
# When you're calling a function, Kite will show you the function's signature
# to help you remember what arguments you need to pass in.

# TRY IT
# ------
# * Put your cursor at the end of line marked with "<--".
# * Type "(" to start the function call, and Kite will show you how to call
#   json.dumps.
# * Remember to press <C-x><C-u> after typing "(" if you've disabled function
#   assist from happening automatically.

json.dumps  # <--


# 2b. Learning from your own code
#
# Kite will also show you signatures for functions that you have defined in
# your own code.

# TRY IT
# ------
# * Put your cursor at the end of the line marked with "<--".
# * Type "(" to get assistance for your locally defined pretty_print function.
# * Remember to press <C-x><C-u> after typing "(" if you've disabled function
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
# * Put your cursor between the "(" and ")" on the line marked with "<--".
# * Enter insert mode.
# * Press <C-x><C-u> to access function assist.

pretty_print()  # <--




""" PART 3: INSTANT DOCUMENTATION ==============================================

    Kite can also show you documentation for the symbols in your code.

    To do so, position your cursor over a symbol, and press "K" in normal mode.
"""


# TRY IT
# ------
# * Put your cursor over "dumps".
# * Enter normal mode.
# * Press "K" to view the documentation for json.dumps.

json.dumps




""" That's it!

    Now you know how to use Kite's core features to boost your productivity as
    you code. You can access this tutorial at any time by running the command
    "KiteTutorial" from the command line.

    You can learn more about Kite's Vim plugin at its GitHub repo:

    https://github.com/kiteco/vim-plugin

    If you get stuck at any point, please visit https://help.kite.com/ or file
    an issue at https://github.com/kiteco/issue-tracker.

    ____________________________________________________________________________

    Kite is under active development. You can expect its features to become
    smarter and more featured over time.
"""
