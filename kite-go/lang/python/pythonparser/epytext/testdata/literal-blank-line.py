# Blank lines are preserved, but the whitespace on a blank
# line isn't. Tab-indented line is replaced by spaces.
def example():
  """
  This introduces a literal::
    First line, followed by blank...

         With a 5-space indented line;
    				And a 4-tab indented line;
    Then followed by 2 blank lines with indent...
    

    Then followed by other paragraph.
  Other paragraph.
  """
  return 1
